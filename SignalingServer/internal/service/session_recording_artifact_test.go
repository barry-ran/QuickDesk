package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeArtifactExtension(t *testing.T) {
	valid := []struct {
		in   string
		want string
	}{
		{"clip.mp4", ".mp4"},
		{"CLIP.MP4", ".mp4"},
		{"clip.Mkv", ".mkv"},
		{"clip.webm", ".webm"},
		{"clip.h264", ".h264"},
		{"clip.grf", ".grf"},
		{".mp4", ".mp4"},
	}
	for _, tc := range valid {
		got, err := NormalizeArtifactExtension(tc.in)
		if err != nil {
			t.Fatalf("NormalizeArtifactExtension(%q) returned error %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeArtifactExtension(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalid := []string{"clip.exe", "clip.sh", "clip", "clip.mp4.exe", "clip.js", "../clip.mp4.bak"}
	for _, in := range invalid {
		if _, err := NormalizeArtifactExtension(in); !errors.Is(err, ErrArtifactTypeDenied) {
			t.Fatalf("NormalizeArtifactExtension(%q) error = %v, want ErrArtifactTypeDenied", in, err)
		}
	}
}

func TestArtifactPathIsDerivedNotUploaded(t *testing.T) {
	// The stored name must come from the recording id, so an attacker-supplied
	// filename can never escape the configured directory.
	path := ArtifactPath("/data/recordings", 7, 42, ".mp4")
	want := filepath.Join("/data/recordings", "7", "rec-42.mp4")
	if path != want {
		t.Fatalf("ArtifactPath() = %q, want %q", path, want)
	}
	if strings.Contains(path, "..") {
		t.Fatalf("ArtifactPath() = %q, must not contain traversal segments", path)
	}
}

func TestArtifactURLIsServerRelative(t *testing.T) {
	if got, want := ArtifactURL(42), "/v1/me/recordings/42/artifact"; got != want {
		t.Fatalf("ArtifactURL() = %q, want %q", got, want)
	}
}

func TestArtifactFormat(t *testing.T) {
	cases := map[string]string{
		".mp4":  "mp4",
		".MP4":  "mp4",
		".m4v":  "mp4",
		".mkv":  "mkv",
		".webm": "webm",
		".h264": "h264",
		".grf":  "grf",
		".exe":  "",
	}
	for ext, want := range cases {
		if got := ArtifactFormat(ext); got != want {
			t.Fatalf("ArtifactFormat(%q) = %q, want %q", ext, got, want)
		}
	}
}
