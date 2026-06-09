package service

import "testing"

func TestBuildAlgoDownloadURL(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		token     string
		want      string
	}{
		{
			name:      "absolute uploads public url",
			publicURL: "http://localhost:8080/uploads",
			token:     "abc",
			want:      "http://localhost:8080/api/v1/internal/algo/download?token=abc",
		},
		{
			name:      "relative uploads public url",
			publicURL: "/uploads",
			token:     "abc",
			want:      "/api/v1/internal/algo/download?token=abc",
		},
		{
			name:      "token escaped",
			publicURL: "http://localhost:8080/uploads/",
			token:     "a+b c",
			want:      "http://localhost:8080/api/v1/internal/algo/download?token=a%2Bb+c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAlgoDownloadURL(tt.publicURL, tt.token); got != tt.want {
				t.Fatalf("buildAlgoDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
