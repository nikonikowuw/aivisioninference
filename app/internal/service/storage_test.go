package service

import (
	"testing"
)

func TestStorageConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       StorageConfig
		wantErr   bool
	}{
		{
			name: "valid: cron mode",
			cfg: StorageConfig{
				CleanupMode:    CleanupModeCron,
				RetentionDays:  7,
				CronExpression: "0 2 * * *",
				Enabled:        true,
			},
			wantErr: false,
		},
		{
			name: "valid: threshold mode",
			cfg: StorageConfig{
				CleanupMode:        CleanupModeThreshold,
				RetentionDays:      7,
				ThresholdValue:     90,
				TargetPercentage:   70,
				ThresholdCheckCron: "*/5 * * * *",
				Enabled:            true,
			},
			wantErr: false,
		},
		{
			name: "invalid: threshold target >= trigger",
			cfg: StorageConfig{
				CleanupMode:        CleanupModeThreshold,
				ThresholdValue:     70,
				TargetPercentage:   90,
				ThresholdCheckCron: "*/5 * * * *",
				Enabled:            true,
			},
			wantErr: true,
		},
		{
			name: "invalid: missing cleanup_mode",
			cfg: StorageConfig{
				RetentionDays: 7,
				Enabled:       true,
			},
			wantErr: true,
		},
		{
			name: "invalid: cron mode missing expression",
			cfg: StorageConfig{
				CleanupMode:   CleanupModeCron,
				RetentionDays: 7,
				Enabled:       true,
			},
			wantErr: true,
		},
		{
			name: "invalid: threshold mode missing check cron",
			cfg: StorageConfig{
				CleanupMode:    CleanupModeThreshold,
				ThresholdValue: 90,
				TargetPercentage: 70,
				Enabled:        true,
			},
			wantErr: true,
		},
		{
			name: "invalid: bad cron expression syntax",
			cfg: StorageConfig{
				CleanupMode:    CleanupModeCron,
				CronExpression: "not a cron",
				Enabled:        true,
			},
			wantErr: true,
		},
		{
			name: "invalid: bad threshold check cron syntax",
			cfg: StorageConfig{
				CleanupMode:        CleanupModeThreshold,
				ThresholdValue:     90,
				TargetPercentage:   70,
				ThresholdCheckCron: "invalid",
				Enabled:            true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
