package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	mdb "SepTaf/internal/mongo"
)

const wikiAPI = "https://en.wikipedia.org/w/api.php?action=parse&format=json&formatversion=2&prop=text&page=List_of_flight_information_regions_and_area_control_centers"

type wikiParseResp struct {
	Parse struct {
		Title string `json:"title"`
		Text  string `json:"text"` // HTML (tables are here)
	} `json:"parse"`
}

// ---- HTTP fetch via API (with proper headers) ----
func fetchWikipediaHTMLViaAPI(ctx context.Context) (string, error) {

	req, _ := http.NewRequestWithContext(ctx, "GET", wikiAPI, nil)
	req.Header.Set("User-Agent", "SepTaf-FIR-Ingest/1.0 (contact: you@example.com)")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		// ساده‌ترین backoff
		time.Sleep(2 * time.Second)
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("wiki API http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var w wikiParseResp
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return "", err
	}
	if w.Parse.Text == "" {
		return "", errors.New("empty wiki parse text")
	}
	return w.Parse.Text, nil // this is HTML snippet containing tables
}

// ---- light HTML helpers (no external packages) ----
func stripTags(s string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func splitTables(html string) []string {
	out := []string{}
	i := 0
	for {
		start := strings.Index(html[i:], "<table")
		if start < 0 {
			break
		}
		start += i
		end := strings.Index(html[start:], "</table>")
		if end < 0 {
			break
		}
		end = start + end + len("</table>")
		out = append(out, html[start:end])
		i = end
	}
	return out
}

type colIdx struct {
	ICAO int
	FIR  int
	Cntr int
	Type int
}

func findHeaderIndexes(tbl string) (colIdx, bool) {
	// first <tr> (header)
	trStart := strings.Index(tbl, "<tr")
	if trStart < 0 {
		return colIdx{}, false
	}
	trEnd := strings.Index(tbl[trStart:], "</tr>")
	if trEnd < 0 {
		return colIdx{}, false
	}
	tr := tbl[trStart : trStart+trEnd]

	// extract header cells (th/td)
	cells := []string{}
	j := 0
	for {
		next := -1
		tag := ""
		posTh := strings.Index(tr[j:], "<th")
		posTd := strings.Index(tr[j:], "<td")
		if posTh >= 0 {
			next = posTh
			tag = "th"
		}
		if posTd >= 0 && (next < 0 || posTd < next) {
			next = posTd
			tag = "td"
		}
		if next < 0 {
			break
		}
		next += j
		closeTag := "</th>"
		if tag == "td" {
			closeTag = "</td>"
		}
		end := strings.Index(tr[next:], closeTag)
		if end < 0 {
			break
		}
		end = next + end + len(closeTag)
		cells = append(cells, tr[next:end])
		j = end
	}

	idx := colIdx{-1, -1, -1, -1}
	for i, c := range cells {
		h := strings.ToLower(stripTags(c))
		switch {
		case strings.Contains(h, "icao"):
			idx.ICAO = i
		case h == "fir" || strings.Contains(h, "fir"):
			idx.FIR = i
		case strings.Contains(h, "country"):
			idx.Cntr = i
		case strings.Contains(h, "type"):
			idx.Type = i
		}
	}
	ok := idx.ICAO >= 0 && idx.FIR >= 0 && idx.Cntr >= 0
	return idx, ok
}

func extractRows(tbl string, idx colIdx) []mdb.FIRDoc {
	rows := []mdb.FIRDoc{}

	trs := strings.Split(tbl, "<tr")
	for _, raw := range trs {
		tr := "<tr" + raw
		if !strings.Contains(tr, "</tr>") {
			continue
		}

		// all cells
		cells := []string{}
		j := 0
		for {
			which := ""
			posTh := strings.Index(tr[j:], "<th")
			posTd := strings.Index(tr[j:], "<td")
			if posTh < 0 && posTd < 0 {
				break
			}
			if posTd >= 0 && (posTh < 0 || posTd < posTh) {
				which = "td"
				posTd += j
				end := strings.Index(tr[posTd:], "</td>")
				if end < 0 {
					break
				}
				end = posTd + end + len("</td>")
				cells = append(cells, tr[posTd:end])
				j = end
			} else {
				which = "th"
				posTh += j
				end := strings.Index(tr[posTh:], "</th>")
				if end < 0 {
					break
				}
				end = posTh + end + len("</th>")
				cells = append(cells, tr[posTh:end])
				j = end
			}
			_ = which
		}
		if len(cells) == 0 {
			continue
		}

		get := func(i int) string {
			if i < 0 || i >= len(cells) {
				return ""
			}
			return stripTags(cells[i])
		}

		icao := strings.ToUpper(strings.TrimSpace(get(idx.ICAO))) // often 4-letter FIR code
		firName := get(idx.FIR)
		country := get(idx.Cntr)
		typ := ""
		if idx.Type >= 0 {
			typ = strings.ToLower(get(idx.Type))
		}

		// skip headers/invalids
		if strings.EqualFold(icao, "icao code") || country == "" {
			continue
		}
		// only core FIRs; skip oceanic/UIR/mil/eurocontrol (optional rule)
		if strings.Contains(typ, "oceanic") || strings.Contains(typ, "uir") ||
			strings.Contains(typ, "eurocontrol") || strings.Contains(typ, "mil") {
			continue
		}

		if len(icao) != 4 {
			continue
		}

		rows = append(rows, mdb.FIRDoc{
			Country:   country,
			FIRCode:   icao,
			FIRName:   firName,
			Source:    "wikipedia_api",
			UpdatedAt: time.Now().UTC(),
		})
	}
	return rows
}

// ---- Public: fetch + parse + return list ----
func FetchFIRListFromWikipedia(ctx context.Context) ([]mdb.FIRDoc, error) {
	html, err := fetchWikipediaHTMLViaAPI(ctx)
	if err != nil {
		return nil, err
	}
	// html now is the body (tables included)
	tables := splitTables(html)
	for _, tbl := range tables {
		idx, ok := findHeaderIndexes(tbl)
		if !ok {
			continue
		}
		rows := extractRows(tbl, idx)
		if len(rows) > 0 {
			return rows, nil
		}
	}
	return nil, errors.New("no matching table found in wiki API response")
}

// ---- Entry: upsert into Mongo ----
func ParseWikipediaFIRsAndUpsert(ctx context.Context, mc *mdb.Client) error {
	items, err := FetchFIRListFromWikipedia(ctx)
	if err != nil {
		return err
	}
	if err := mc.EnsureCountriesIndexes(ctx); err != nil {
		return err
	}
	return mc.BulkUpsertFIRs(ctx, items)
}
