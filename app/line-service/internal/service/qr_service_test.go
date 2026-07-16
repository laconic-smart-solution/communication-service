package service

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateEncodesAddFriendURL(t *testing.T) {
	res, err := NewQRService().Generate("@136laxkx", 0)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if want := "https://line.me/R/ti/p/%40136laxkx"; res.AddFriendURL != want {
		t.Errorf("AddFriendURL = %q, want %q", res.AddFriendURL, want)
	}

	img, err := png.Decode(bytes.NewReader(res.PNG))
	if err != nil {
		t.Fatalf("PNG is not decodable: %v", err)
	}
	if got := img.Bounds().Dx(); got != DefaultQRSize {
		t.Errorf("PNG width = %d, want %d", got, DefaultQRSize)
	}
}

func TestGenerateNormalizesMissingAtPrefix(t *testing.T) {
	res, err := NewQRService().Generate("136laxkx", 128)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if res.BasicID != "@136laxkx" {
		t.Errorf("BasicID = %q, want %q", res.BasicID, "@136laxkx")
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		basicID string
		size    int
	}{
		"empty":          {"", 0},
		"bad chars":      {"@has spaces", 0},
		"size too small": {"@136laxkx", 8},
		"size too big":   {"@136laxkx", 4096},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewQRService().Generate(tc.basicID, tc.size); err == nil {
				t.Error("Generate() error = nil, want error")
			}
		})
	}
}
