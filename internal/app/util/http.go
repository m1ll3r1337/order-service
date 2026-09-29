package util

import (
	"net/http"
	"strings"
)

// IsFilteredHttpRoute возвращает true для маршрутов, которые не нужно логировать
func IsFilteredHttpRoute(r *http.Request) bool {
	switch {
	case strings.Contains(r.RequestURI, "health"):
		return true
	case strings.Contains(r.RequestURI, "debug"):
		return true
	case strings.Contains(r.RequestURI, "metric"):
		return true
	}

	return false
}
