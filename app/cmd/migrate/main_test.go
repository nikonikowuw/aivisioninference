package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootUsernameConstraintSQLRequiresRootUsernameForRootUser(t *testing.T) {
	sql := rootUsernameConstraintSQL()

	assert.Contains(t, sql, "chk_root_username")
	assert.Contains(t, strings.Join(strings.Fields(sql), " "), "CHECK (is_root = false OR username = 'root')")
}

func TestShouldCreateSeedAdminSkipsWhenUsernameOrEmailAlreadyExists(t *testing.T) {
	tests := []struct {
		name                string
		adminUsernameExists bool
		adminEmailExists    bool
		want                bool
	}{
		{name: "creates when neither username nor email exists", want: true},
		{name: "skips when username exists", adminUsernameExists: true},
		{name: "skips when email exists", adminEmailExists: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldCreateSeedAdmin(tt.adminUsernameExists, tt.adminEmailExists)

			assert.Equal(t, tt.want, got)
		})
	}
}
