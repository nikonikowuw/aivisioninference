// Package version provides version comparison utilities for engine compatibility checks.
package version

import (
	"fmt"

	goversion "github.com/hashicorp/go-version"
)

// IsCompatible checks whether engineVersion meets the minimum required version.
// Returns true if engineVersion >= minVersion.
func IsCompatible(engineVersion string, minVersion string) (bool, error) {
	v1, err := goversion.NewVersion(engineVersion)
	if err != nil {
		return false, fmt.Errorf("invalid engine version %q: %w", engineVersion, err)
	}

	v2, err := goversion.NewVersion(minVersion)
	if err != nil {
		return false, fmt.Errorf("invalid minimum version %q: %w", minVersion, err)
	}

	return v1.GreaterThanOrEqual(v2), nil
}
