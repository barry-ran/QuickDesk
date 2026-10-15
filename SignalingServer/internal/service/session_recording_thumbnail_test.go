package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
)

func TestSniffThumbnailType(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want string
	}{
		{"jpeg", jpegMagic, "image/jpeg"},
		{"png", pngMagic, "image/png"},
		{"gif", []byte("GIF89a\x01\x00"), ""},
		{"html", []byte("<!DOCTYPE"), ""},
		{"empty", nil, ""},
		{"short", []byte{0xFF, 0xD8}, ""},
	}
	for _, tc := range cases {
		got, err := SniffThumbnailType(tc.head)
		if tc.want == "" {
			if !errors.Is(err, ErrInvalidThumbnailContent) {
				t.Errorf("%s: error = %v, want ErrInvalidThumbnailContent", tc.name, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: type = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestNormalizeThumbnailExtension(t *testing.T) {
	ok := map[string]string{
		"shot.JPG":        ".jpg",
		"shot.jpeg":       ".jpeg",
		"shot.png":        ".png",
		" /tmp/shot.PNG ": ".png",
	}
	for in, want := range ok {
		got, err := NormalizeThumbnailExtension(in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q: ext = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"", "shot.gif", "shot.mp4", "shot", "shot.jpg.exe"} {
		if _, err := NormalizeThumbnailExtension(in); !errors.Is(err, ErrArtifactTypeDenied) {
			t.Errorf("%q: error = %v, want ErrArtifactTypeDenied", in, err)
		}
	}
}

func TestThumbnailExtensionForType(t *testing.T) {
	if got := ThumbnailExtensionForType("image/jpeg"); got != ".jpg" {
		t.Errorf("image/jpeg -> %q, want .jpg", got)
	}
	if got := ThumbnailExtensionForType("IMAGE/PNG"); got != ".png" {
		t.Errorf("IMAGE/PNG -> %q, want .png", got)
	}
	if got := ThumbnailExtensionForType("image/gif"); got != "" {
		t.Errorf("image/gif -> %q, want empty", got)
	}
}

// The stored name must come from the ids alone, never from the upload.
func TestThumbnailPathIgnoresUntrustedInput(t *testing.T) {
	got := ThumbnailPath("/data/recordings", 7, 42, ".jpg")
	want := filepath.Join("/data/recordings", "7", "thumb-42.jpg")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if strings.Contains(got, "..") {
		t.Fatalf("path %q escaped the store", got)
	}
	if url := ThumbnailURL(42); url != "/v1/me/recordings/42/thumbnail" {
		t.Fatalf("url = %q", url)
	}
}
