package httpx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"SepTaf/internal/config"
	mdb "SepTaf/internal/mongo"
)

// ====== کانتکست ======
type ctxKey string

const CtxClientID ctxKey = "client_id"

// ====== ابزار ======

// IP allowlist: پشتیبانی از IP و CIDR
func ipAllowed(remote string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	ip := net.ParseIP(remote)
	if ip == nil {
		return false
	}
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if strings.Contains(a, "/") {
			_, cidr, err := net.ParseCIDR(a)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) {
				return true
			}
		} else {
			if ip.Equal(net.ParseIP(a)) {
				return true
			}
		}
	}
	return false
}

// بدنه خالی → SHA256("")
func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// // ساخت canonical query
//
//	func canonicalQuery(raw string) string {
//		if raw == "" {
//			return ""
//		}
//		m, _ := url.ParseQuery(raw)
//		keys := make([]string, 0, len(m))
//		for k := range m {
//			keys = append(keys, k)
//		}
//		sort.Strings(keys)
//		parts := make([]string, 0, len(keys))
//		for _, k := range keys {
//			vs := m[k]
//			sort.Strings(vs)
//			for _, v := range vs {
//				parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
//			}
//		}
//		return strings.Join(parts, "&")
//	}
func canonicalQuery(raw string) string {
	if raw == "" {
		return ""
	}
	v, err := url.ParseQuery(raw)
	if err != nil {
		return "" // یا 400 بده؛ ولی برای امضا بهتره پایدار باشه
	}
	for k := range v {
		sort.Strings(v[k]) // مقادیر هر کلید هم پایدار شوند
	}
	return v.Encode() // کلیدها را خودش سورت می‌کند؛ space => '+'
}

func canonicalQueryRaw(raw string) string { return raw }

// استخراج IP واقعی (اگر پشت LB نیستی همون RemoteAddr)
func getRemoteIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ====== حافظهٔ درون‌پردازه‌ای برای nonce و rate ======

type nonceStore struct {
	mu   sync.Mutex
	seen map[string]time.Time // key = clientID + "|" + nonce
	ttl  time.Duration
}

func newNonceStore(ttlSec int) *nonceStore {
	ns := &nonceStore{
		seen: make(map[string]time.Time),
		ttl:  time.Duration(ttlSec) * time.Second,
	}
	// پاکسازی دوره‌ای
	go func() {
		t := time.NewTicker(time.Minute)
		for range t.C {
			now := time.Now()
			ns.mu.Lock()
			for k, exp := range ns.seen {
				if now.After(exp) {
					delete(ns.seen, k)
				}
			}
			ns.mu.Unlock()
		}
	}()
	return ns
}

func (n *nonceStore) addOnce(clientID, nonce string) bool {
	k := clientID + "|" + nonce
	now := time.Now()
	n.mu.Lock()
	defer n.mu.Unlock()
	if exp, ok := n.seen[k]; ok && now.Before(exp) {
		return false // replay
	}
	n.seen[k] = now.Add(n.ttl)
	return true
}

type tokenBucket struct {
	mu         sync.Mutex
	capacity   int
	tokens     int
	refillInt  time.Duration
	lastRefill time.Time
}

func newBucket(ratePerMin int) *tokenBucket {
	if ratePerMin <= 0 {
		ratePerMin = 29
	}
	return &tokenBucket{
		capacity:   ratePerMin,
		tokens:     ratePerMin,
		refillInt:  time.Minute / time.Duration(ratePerMin),
		lastRefill: time.Now(),
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	// refill
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	steps := int(elapsed / b.refillInt)
	if steps > 0 {
		b.tokens += steps
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.lastRefill = b.lastRefill.Add(time.Duration(steps) * b.refillInt)
	}
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

type rateRegistry struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket // per client
	defRate int
}

func newRateRegistry(def int) *rateRegistry {
	return &rateRegistry{
		buckets: make(map[string]*tokenBucket),
		defRate: def,
	}
}
func (rr *rateRegistry) allow(clientID string, rate int) bool {
	rr.mu.Lock()
	b, ok := rr.buckets[clientID]
	if !ok {
		if rate <= 0 {
			rate = rr.defRate
		}
		b = newBucket(rate)
		rr.buckets[clientID] = b
	}
	rr.mu.Unlock()
	return b.allow()
}

// canonicalQueryOrderInsensitive:
// - RawQuery را parse می‌کند
// - کلیدها را سورت می‌کند (Encode خودش انجام می‌دهد)
// - مقدارهای هر کلید را هم سورت می‌کند تا ترتیب پارامترها/مقادیر بی‌اثر شود
// - خروجی با سیاست form-encoding گو (space => '+')
func canonicalQueryOrderInsensitive(raw string) string {
	if raw == "" {
		return ""
	}
	v, err := url.ParseQuery(raw)
	if err != nil {
		return "" // یا می‌تونی 400 بدهی؛ ولی برای پایداری امضاء، خالی هم ok است
	}
	for k := range v {
		sort.Strings(v[k]) // مقدارهای تکراری یک کلید هم پایدار می‌شوند
	}
	// Encode: کلیدها را به‌صورت الفبایی مرتب می‌کند، escape به سبک form (space => '+')
	return v.Encode()
}

// buildCanonical:
// - method دست‌نخورده (همون چیزی که کلاینت می‌فرسته؛ معمولاً "GET")
// - path دقیق و بدون دستکاری با EscapedPath (lowercase نکن)
// - query: سورتِ کلید و مقدار (ترتیب ورودی بی‌اثر)
// - bodyHashHex را lowercase می‌کنیم که پایدار باشد
// - بدون newline انتهایی (policy ثابت)

func buildCanonical(r *http.Request, bodyHashHex, xDate, xNonce, keyVer string) string {
	method := r.Method                         // همانی که آمده (GET/POST/…)
	path := r.URL.EscapedPath()                // دقیق و بدون lowercase
	query := canonicalQueryRaw(r.URL.RawQuery) // همانی که کلاینت فرستاده
	body := strings.ToLower(bodyHashHex)       // برای GET خالی: e3b0...b855

	return strings.Join([]string{
		method,
		path,
		query, // توجه: می‌تونه خالی باشه؛ در این صورت این خط خالی می‌مونه
		body,
		xDate,
		xNonce,
		keyVer,
	}, "\n") // بدون newline نهایی
}

func verifyHMAC(secretRawBase64 string, canonical string, sigB64 string) error {

	debugCanonical(canonical, secretRawBase64)

	secret, err := base64.StdEncoding.DecodeString(secretRawBase64)
	if err != nil {
		return fmt.Errorf("bad secret encoding: %w", err)
	}
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(canonical))
	want := m.Sum(nil)
	got, err := base64.StdEncoding.DecodeString(sigB64)

	if err != nil {
		return errors.New("bad signature encoding")
	}
	// constant-time compare
	if !hmac.Equal(want, got) {
		return errors.New("signature mismatch")
	}
	return nil
}

func debugCanonical(canonical, secretRawBase64 string) {
	fmt.Printf("[DEBUG] canonical.len=%d\n", len(canonical))
	fmt.Printf("[DEBUG] canonical.b64=%s\n", base64.StdEncoding.EncodeToString([]byte(canonical)))

	secret, _ := base64.StdEncoding.DecodeString(secretRawBase64)
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(canonical))
	fmt.Printf("[DEBUG] server_sig=%s\n", base64.StdEncoding.EncodeToString(m.Sum(nil)))
}

// ====== Middleware ======

type AuthMiddleware struct {
	cfg    config.Config
	mc     *mdb.Client
	nonces *nonceStore
	rates  *rateRegistry
}

func NewAuthMiddleware(cfg config.Config, mc *mdb.Client) *AuthMiddleware {
	ttl := cfg.NonceTTLSeconds
	if ttl <= 0 {
		ttl = 600
	}
	defRate := cfg.DefaultRatePerMin
	if defRate <= 0 {
		defRate = 29
	}
	return &AuthMiddleware{
		cfg:    cfg,
		mc:     mc,
		nonces: newNonceStore(ttl),
		rates:  newRateRegistry(defRate),
	}
}

func (a *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := strings.TrimSpace(r.Header.Get("X-Client-Id"))
		if clientID == "" {
			http.Error(w, `{"error":"invalid_client","message":"missing X-Client-Id"}`, http.StatusUnauthorized)
			return
		}

		// 1) load client
		client, err := a.mc.GetAPIClient(r.Context(), clientID)
		if err != nil {
			http.Error(w, `{"error":"invalid_client","message":"unknown or disabled client"}`, http.StatusUnauthorized)
			return
		}

		// 2) IP allowlist
		remoteIP := getRemoteIP(r)
		if !ipAllowed(remoteIP, client.AllowedIPs) {
			http.Error(w, `{"error":"ip_not_allowed","message":"source IP not allowed"}`, http.StatusForbidden)
			return
		}

		// 3) Auth paths
		strict := a.cfg.AuthStrictMode
		xSig := strings.TrimSpace(r.Header.Get("X-Signature"))
		if strict && xSig == "" {
			http.Error(w, `{"error":"invalid_client","message":"signature required (strict mode)"}`, http.StatusUnauthorized)
			return
		}

		keyVer := strings.TrimSpace(r.Header.Get("X-Key-Version"))
		ver, secretEnc, ok := client.FindSecret(keyVer)
		if !ok {
			http.Error(w, `{"error":"invalid_client","message":"unknown or inactive key version"}`, http.StatusUnauthorized)
			return
		}
		_ = ver // الان برای لاگ/دیباگ می‌تونی استفاده کنی

		// اگر permissive و signature نبود → خامِ secret بپذیر
		if xSig == "" {
			if !a.cfg.AuthStrictMode {
				raw := strings.TrimSpace(r.Header.Get("X-Client-Secret"))
				if raw == "" || raw != secretEnc { // در MVP: secretEnc همان Base64 خام است
					http.Error(w, `{"error":"invalid_client","message":"bad client secret"}`, http.StatusUnauthorized)
					return
				}
			} else {
				http.Error(w, `{"error":"invalid_client","message":"missing signature"}`, http.StatusUnauthorized)
				return
			}
		} else {
			// 4) Anti-replay: X-Date & X-Nonce
			xDate := strings.TrimSpace(r.Header.Get("X-Date"))
			xNonce := strings.TrimSpace(r.Header.Get("X-Nonce"))
			if xDate == "" || xNonce == "" {
				http.Error(w, `{"error":"bad_request","message":"missing X-Date or X-Nonce"}`, http.StatusBadRequest)
				return
			}
			// skew check
			var ts time.Time
			// تلاش برای RFC3339
			tRFC, err1 := time.Parse(time.RFC3339, xDate)
			if err1 == nil {
				ts = tRFC
			} else {
				// تلاش برای epoch seconds
				if secs, err2 := time.ParseDuration(xDate + "s"); err2 == nil {
					ts = time.Unix(int64(secs.Seconds()), 0)
				} else {
					http.Error(w, `{"error":"bad_request","message":"bad X-Date"}`, http.StatusBadRequest)
					return
				}
			}
			skew := a.cfg.DateSkewSeconds
			if skew <= 0 {
				skew = 60
			}
			if abs := time.Since(ts); abs > time.Duration(skew)*time.Second || abs < -time.Duration(skew)*time.Second {
				http.Error(w, `{"error":"bad_request","message":"date skew too large"}`, http.StatusBadRequest)
				return
			}
			// nonce reuse?
			if ok := a.nonces.addOnce(clientID, xNonce); !ok {
				http.Error(w, `{"error":"replay_detected","message":"nonce already used"}`, http.StatusUnauthorized)
				return
			}
			// canonical + verify
			bodyHash := sha256Hex(nil)
			if r.Body != nil && (r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH") {
				// بدنه را دوباره‌خوانی نمی‌کنیم که handler مشکل نخورد: فرض ساده اینکه بدنه کوچک است و قبلا خوانده نشده
				// اگر نیاز به ریدرپلی داری، از middleware مخصوص copy body استفاده کن.
			}
			canon := buildCanonical(r, bodyHash, xDate, xNonce, ver)
			if err := verifyHMAC(secretEnc, canon, xSig); err != nil {
				http.Error(w, `{"error":"invalid_client","message":"signature mismatch"}`, http.StatusUnauthorized)
				return
			}
		}

		// 5) Rate limit per-client
		rate := client.RatePerMinute
		if rate <= 0 {
			rate = a.cfg.DefaultRatePerMin
		}
		if !a.rates.allow(clientID, rate) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate_limited","message":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// 6) گذار به هندلر
		ctx := context.WithValue(r.Context(), CtxClientID, clientID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
