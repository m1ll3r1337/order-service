package mzerolog

import (
	"net/http"

	"github.com/rs/zerolog"
)

type Option = func(m *middleware)

func WithLogger(l zerolog.Logger) Option {
	return func(m *middleware) {
		m.log = l
	}
}

func WithSkipper(skipper func(r *http.Request) bool) Option {
	if skipper != nil {
		return func(m *middleware) {
			m.fromOptions.skipper = skipper
		}
	}

	return nil
}
