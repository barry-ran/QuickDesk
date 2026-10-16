package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// An uploaded installer's name decides the on-disk leaf, and a crafted one must
// not be able to walk out of the update store or name a hidden file.
func TestSanitizeUpdateArtifactName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"QuickDesk-4.2.5.exe", "QuickDesk-4.2.5.exe"},
		{"../../etc/passwd", "passwd"},
		{"/absolute/QuickDesk.dmg", "QuickDesk.dmg"},
		{`C:\Users\me\QuickDesk.exe`, "C:UsersmeQuickDesk.exe"},
		{"  spaced name.exe  ", "spaced name.exe"},
		{"\u0000\u0007", ""},
		{"...", ""},
		{"", ""},
		{"..", ""},
	}
	for _, tc := range cases {
		if got := sanitizeUpdateArtifactName(tc.in); got != tc.want {
			t.Fatalf("sanitizeUpdateArtifactName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// A long name keeps a usable extension instead of being cut off mid-suffix.
	deep := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.exe"
	got := sanitizeUpdateArtifactName(deep)
	if len([]rune(got)) > 120 {
		t.Fatalf("long name was not shortened: %d runes", len([]rune(got)))
	}
	if ext := got[len(got)-4:]; ext != ".exe" {
		t.Fatalf("long name lost its extension: %q", got)
	}
}

// The stored download_url has to be absolute, because that is what the contract
// promises and what a client that is not sitting behind the same proxy needs.
func TestUpdateArtifactDownloadURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newCtx := func(host string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/admin/update-releases/7/artifact", nil)
		c.Request.Host = host
		return c
	}

	derived := NewAdminUpdateReleaseHandler(nil, nil, UpdateArtifactStore{Dir: t.TempDir()})
	if got, want := derived.artifactDownloadURL(newCtx("quickdesk.example.com"), 7),
		"http://quickdesk.example.com/v1/update-releases/7/artifact"; got != want {
		t.Fatalf("derived url = %q, want %q", got, want)
	}

	configured := NewAdminUpdateReleaseHandler(nil, nil, UpdateArtifactStore{
		Dir:           t.TempDir(),
		PublicBaseURL: "https://updates.example.com/",
	})
	if got, want := configured.artifactDownloadURL(newCtx("internal:8000"), 7),
		"https://updates.example.com/v1/update-releases/7/artifact"; got != want {
		t.Fatalf("configured url = %q, want %q", got, want)
	}
}
