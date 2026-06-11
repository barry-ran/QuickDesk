package handler

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterAdminUI wires the embedded SPA (`web/dist/*`) into the Gin
// router. The rules mirror the plain old behaviour:
//
//   /admin/            → index.html
//   /admin/assets/...  → files from web/dist/assets/*
//   /admin/<anything>  → index.html (SPA fallback)
//
// The embed.FS is passed in so the signaling package owns the //go:embed
// directive (we can't use //go:embed inside an internal package).
func RegisterAdminUI(r *gin.Engine, distFS embed.FS) {
	subFS, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		// Embed failed — either the dist dir is missing or the caller
		// passed the wrong FS. Log once and fall back to a simple 404 so
		// the server still starts cleanly.
		r.GET("/admin/*path", func(c *gin.Context) {
			c.String(http.StatusNotFound, "admin UI bundle missing (%v)", err)
		})
		return
	}
	httpFS := http.FS(subFS)
	fileServer := http.FileServer(httpFS)

	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})
	r.GET("/admin/*path", func(c *gin.Context) {
		p := c.Param("path")
		if p == "" || p == "/" {
			serveSPAIndex(c, httpFS)
			return
		}
		p = strings.TrimPrefix(p, "/")

		// Only serve files that exist; otherwise fall through to index
		// (SPA routing).
		if _, err := fs.Stat(subFS, p); err != nil {
			serveSPAIndex(c, httpFS)
			return
		}
		c.Request.URL.Path = "/" + p
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

func serveSPAIndex(c *gin.Context, httpFS http.FileSystem) {
	f, err := httpFS.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "admin UI index.html missing")
		return
	}
	defer f.Close()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	_, _ = copyReader(c.Writer, f)
}

func copyReader(w http.ResponseWriter, r interface {
	Read([]byte) (int, error)
}) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			return total, err
		}
	}
}
