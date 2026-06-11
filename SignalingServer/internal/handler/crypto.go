package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
)

// hmacSHA1Base64 returns base64(HMAC-SHA1(key, data)). Used to build coturn
// shared-secret credentials (§GetICEConfig).
func hmacSHA1Base64(key, data string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
