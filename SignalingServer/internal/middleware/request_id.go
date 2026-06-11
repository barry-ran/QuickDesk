package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID populates `request_id` in the gin context (consumed by
// handler.WriteProblem to emit `trace_id`) and echoes it back as the
// `X-Request-ID` response header. If the client already sent one (e.g.
// from an upstream tracer) we keep it; otherwise we mint a fresh UUID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			// `traceparent` (W3C tracecontext) takes priority when set.
			if tp := c.GetHeader("traceparent"); tp != "" {
				id = tp
			} else {
				id = uuid.NewString()
			}
		}
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}
