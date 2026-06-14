package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func (r Release) Fingerprint() string {
	parts := []string{
		fmt.Sprintf("%d", r.SourceID),
		normalizeFingerprintPart(r.Title),
		normalizeFingerprintPart(r.Link),
		normalizeFingerprintPart(r.DownloadURL),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func normalizeFingerprintPart(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}
