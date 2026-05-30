package timex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeRange(t *testing.T) {
	t.Run("empty strings", func(t *testing.T) {
		var from, to *time.Time
		err := NormalizeRange("", "", &from, &to)
		assert.NoError(t, err)
		assert.Nil(t, from)
		assert.Nil(t, to)
	})

	// Note: We don't test full parsing here as that relies on scopes.ParseTimeRange which is tested elsewhere.
	// This ensures the mapping logic works.
}
