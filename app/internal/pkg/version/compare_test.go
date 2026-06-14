package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		name          string
		engineVersion string
		minVersion    string
		want          bool
		wantErr       bool
	}{
		{"same version", "1.0.0", "1.0.0", true, false},
		{"newer major", "2.0.0", "1.0.0", true, false},
		{"newer minor", "1.1.0", "1.0.0", true, false},
		{"newer patch", "1.0.1", "1.0.0", true, false},
		{"older major", "0.9.0", "1.0.0", false, false},
		{"older minor", "1.0.0", "1.1.0", false, false},
		{"older patch", "1.0.0", "1.0.1", false, false},
		{"pre-release", "1.0.0-alpha", "1.0.0", false, false},
		{"invalid engine version", "abc", "1.0.0", false, true},
		{"invalid min version", "1.0.0", "xyz", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsCompatible(tt.engineVersion, tt.minVersion)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
