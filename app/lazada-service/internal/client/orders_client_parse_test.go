package client

import (
	"testing"
	"time"
)

func TestParseLazadaOrderTime(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		country string
		want    time.Time
	}{
		{
			name:    "rfc3339 with colon offset",
			raw:     "2024-01-15T10:30:00+07:00",
			country: "th",
			want:    time.Date(2024, 1, 15, 3, 30, 0, 0, time.UTC),
		},
		{
			name:    "iso without colon offset",
			raw:     "2024-01-15T10:30:00+0700",
			country: "th",
			want:    time.Date(2024, 1, 15, 3, 30, 0, 0, time.UTC),
		},
		{
			name:    "space separated with offset",
			raw:     "2024-01-15 10:30:00 +0700",
			country: "th",
			want:    time.Date(2024, 1, 15, 3, 30, 0, 0, time.UTC),
		},
		{
			name:    "utc z",
			raw:     "2024-01-15T10:30:00Z",
			country: "th",
			want:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:    "unix seconds",
			raw:     "1705312200",
			country: "th",
			want:    time.Unix(1705312200, 0).UTC(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLazadaOrderTime(tt.raw, tt.country)
			if got.IsZero() {
				t.Fatalf("parseLazadaOrderTime(%q) returned zero time", tt.raw)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("parseLazadaOrderTime(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
