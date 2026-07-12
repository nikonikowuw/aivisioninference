package service

import "testing"

func TestEdgeCommandTopicUsesNodeID(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"start_stream", "aivision/edge/node-1/cmd/start_stream"},
		{"stop_stream", "aivision/edge/node-1/cmd/stop_stream"},
		{"start_playback", "aivision/edge/node-1/cmd/start_playback"},
		{"stop_playback", "aivision/edge/node-1/cmd/stop_playback"},
		{"stream_status", "aivision/edge/node-1/cmd/stream_status"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := edgeCommandTopic("node-1", tt.command); got != tt.want {
				t.Fatalf("edgeCommandTopic() = %q, want %q", got, tt.want)
			}
		})
	}
}
