package httpx

import (
	"net/http"
	"strconv"
)

type PageMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func getPage(r *http.Request) int {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if p < 1 {
		p = 1
	}
	return p
}
func getLimit(r *http.Request, def, max int64) int64 {
	l, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if l <= 0 {
		l = def
	}
	if l > max {
		l = max
	}
	return l
}
