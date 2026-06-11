package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination helpers. The v1 API is uniformly cursor-based per §3.1:
// every list endpoint accepts `?cursor=…&limit=…` (plus per-endpoint
// filter params like ?search=&sort=) and returns
//   { "items": [...], "next_cursor": "..."|null }
//
// The cursor is an opaque base64 envelope around a small JSON payload —
// callers never introspect it. `EncodeCursor` and `DecodeCursor` are the
// only legitimate ways to cross the wire boundary.

// -----------------------------------------------------------------------
// Request parsing
// -----------------------------------------------------------------------

// CursorParams carries the parsed list query.
type CursorParams struct {
	// Cursor is an opaque string produced by EncodeCursor; empty means
	// "first page".
	Cursor string
	Limit  int

	// Generic filter knobs used by most admin list endpoints. Free-form
	// field-specific filters (e.g. ?level=V1) are still read directly
	// via c.Query in the handler.
	Search string
	Sort   string
	Order  string
}

// Defaults per §2.2 list endpoints. The hard upper bound keeps the server
// safe from clients requesting pathological page sizes.
const (
	defaultCursorLimit = 50
	maxCursorLimit     = 200
)

// ParseCursor reads ?cursor/?limit/?search/?sort/?order. Invalid values
// are silently clamped so lists stay usable even with junk input.
func ParseCursor(c *gin.Context) CursorParams {
	p := CursorParams{
		Limit:  defaultCursorLimit,
		Cursor: c.Query("cursor"),
		Search: c.Query("search"),
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	}
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > maxCursorLimit {
				n = maxCursorLimit
			}
			p.Limit = n
		}
	}
	if p.Sort == "" {
		p.Sort = "id"
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
	return p
}

// OrderClause returns the SQL "col dir" fragment for GORM .Order().
func (p CursorParams) OrderClause() string { return p.Sort + " " + p.Order }

// -----------------------------------------------------------------------
// Cursor payload (opaque to clients)
// -----------------------------------------------------------------------

// CursorPayload is what we embed inside a cursor. Keeping the wire token
// opaque means we can evolve this later without breaking round-trips.
type CursorPayload struct {
	// OffsetID is the greatest `id` already returned. Simple next-page
	// cursors usually need nothing else.
	OffsetID uint `json:"o,omitempty"`
	// OffsetAt is used when we sort by a timestamp instead of ID.
	OffsetAt string `json:"a,omitempty"`
	// Extra is a free-form map for handlers with custom keyset needs.
	Extra map[string]interface{} `json:"x,omitempty"`
}

func EncodeCursor(p CursorPayload) string {
	b, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(cursor string) (CursorPayload, error) {
	if cursor == "" {
		return CursorPayload{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return CursorPayload{}, fmt.Errorf("decode cursor: %w", err)
	}
	var p CursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return CursorPayload{}, fmt.Errorf("parse cursor: %w", err)
	}
	return p, nil
}

// -----------------------------------------------------------------------
// Response envelope
// -----------------------------------------------------------------------

// CursorPage is the canonical list envelope `{items, next_cursor, total?}`
// used by every v1 list endpoint. `Total` is optional and only populated
// when the caller can compute it cheaply (usually a single COUNT(*)).
type CursorPage struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
	Total      *int64      `json:"total,omitempty"`
}

// NewCursorPage builds the envelope. Pass total=-1 when unknown.
func NewCursorPage(items interface{}, nextCursor string, total int64) CursorPage {
	page := CursorPage{Items: items, NextCursor: nextCursor}
	if total >= 0 {
		page.Total = &total
	}
	return page
}
