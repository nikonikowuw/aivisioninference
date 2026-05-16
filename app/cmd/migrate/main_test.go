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
