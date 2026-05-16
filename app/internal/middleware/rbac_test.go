package middleware

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/api/v1/roles/*", "/api/v1/roles/42", true},
		{"/api/v1/roles/*", "/api/v1/roles/42/permissions", false},
		{"/api/v1/roles/*/permissions", "/api/v1/roles/42/permissions", true},
		{"/api/v1/roles/*/permissions", "/api/v1/roles/42", false},
		{"/api/v1/roles/*/permissions", "/api/v1/roles/42/permissions/extra", false},
		{"/api/v1/files/upload/**", "/api/v1/files/upload/init", true},
		{"/api/v1/files/upload/**", "/api/v1/files/upload/123/chunk", true},
		{"/api/v1/files/upload/**", "/api/v1/files/upload/123/complete", true},
		{"/api/v1/files/upload/**", "/api/v1/files/upload", true},
		{"/api/v1/files/upload/**", "/api/v1/files/42", false},
		{"/api/v1/files/*/download", "/api/v1/files/42/download", true},
		{"/api/v1/files/*/download", "/api/v1/files/42", false},
		{"/api/v1/files/*", "/api/v1/files/42", true},
		{"/api/v1/files/*", "/api/v1/files/upload/init", false},
		{"/api/v1/users", "/api/v1/users", true},
		{"/api/v1/users", "/api/v1/users/42", false},
	}

	for _, tt := range tests {
		got := matchPath(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMatchPermission(t *testing.T) {
	perms := []permission{
		{Path: "/api/v1/roles/*", Method: "PUT"},
		{Path: "/api/v1/roles/*/permissions", Method: "PUT"},
		{Path: "/api/v1/roles/*", Method: "GET"},
		{Path: "/api/v1/files/upload/**", Method: "POST"},
		{Path: "/api/v1/files/*", Method: "DELETE"},
		{Path: "/api/v1/files/*/download", Method: "GET"},
	}

	tests := []struct {
		path   string
		method string
		want   bool
	}{
		{"/api/v1/roles/42", "PUT", true},
		{"/api/v1/roles/42/permissions", "PUT", true},
		{"/api/v1/roles/42", "GET", true},
		{"/api/v1/roles/42/permissions", "GET", false},
		{"/api/v1/files/upload/init", "POST", true},
		{"/api/v1/files/upload/123/chunk", "POST", true},
		{"/api/v1/files/42", "DELETE", true},
		{"/api/v1/files/42/download", "GET", true},
		{"/api/v1/files/upload/init", "DELETE", false},
	}

	for _, tt := range tests {
		got := matchPermission(perms, tt.path, tt.method)
		if got != tt.want {
			t.Errorf("matchPermission(%q, %q) = %v, want %v", tt.path, tt.method, got, tt.want)
		}
	}
}

func TestMatchPermissionSuperAdmin(t *testing.T) {
	perms := []permission{{Path: "*", Method: "*"}}
	if !matchPermission(perms, "/any/path", "DELETE") {
		t.Error("super-admin wildcard should match any path and method")
	}
}