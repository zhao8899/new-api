package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const safeLogPayloadPreviewLimit = 512

func SafeLogSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "[empty]"
	}
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("[redacted,len=%d,sha256=%s]", len(value), hex.EncodeToString(sum[:])[:12])
}

func SafeLogPayload(payload []byte) string {
	if len(payload) == 0 {
		return "[empty]"
	}
	sum := sha256.Sum256(payload)
	preview := string(payload)
	if len(preview) > safeLogPayloadPreviewLimit {
		preview = preview[:safeLogPayloadPreviewLimit]
	}
	preview = MaskSensitiveInfo(preview)
	return fmt.Sprintf("[len=%d,sha256=%s,preview=%q]", len(payload), hex.EncodeToString(sum[:])[:12], preview)
}
