package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWebSocketOriginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		{name: "no origin", origin: "", allowed: []string{"https://admin.example.com"}, want: true},
		{name: "explicit origin", origin: "https://admin.example.com", allowed: []string{"https://admin.example.com"}, want: true},
		{name: "wildcard", origin: "https://evil.example.com", allowed: []string{"*"}, want: true},
		{name: "rejected origin", origin: "https://evil.example.com", allowed: []string{"https://admin.example.com"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/api/v1/ws", nil)
			require.NoError(t, err)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			require.Equal(t, tt.want, isWebSocketOriginAllowed(req, tt.allowed))
		})
	}
}
