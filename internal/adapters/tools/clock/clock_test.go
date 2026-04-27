package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.com/gh-models-agent/internal/adapters/tools/clock"
)

// fixedNow returns a deterministic clock for tests.
func fixedNow(t *testing.T) func() time.Time {
	t.Helper()
	ts := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return ts }
}

func TestTool_Schema(t *testing.T) {
	s := clock.New().Schema()

	assert.Equal(t, clock.ToolName, s.Name)
	assert.NotEmpty(t, s.Description)
	assert.Equal(t, "object", s.Parameters["type"])
}

func TestTool_Execute(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		want    string
		wantErr bool
	}{
		{
			name: "defaults to UTC when timezone omitted",
			args: map[string]any{},
			want: "2026-04-25T12:00:00Z",
		},
		{
			name: "defaults to UTC when timezone is empty string",
			args: map[string]any{"timezone": ""},
			want: "2026-04-25T12:00:00Z",
		},
		{
			name: "honors explicit IANA timezone",
			args: map[string]any{"timezone": "Asia/Tokyo"},
			want: "2026-04-25T21:00:00+09:00",
		},
		{
			name:    "rejects unknown timezone",
			args:    map[string]any{"timezone": "Mars/Olympus"},
			wantErr: true,
		},
	}

	tool := clock.New(clock.WithNow(fixedNow(t)))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tool.Execute(context.Background(), tc.args)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
