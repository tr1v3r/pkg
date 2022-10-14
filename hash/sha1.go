package hash

import (
	"crypto/sha1"
	"encoding/hex"
)

// CalcSha1 calculate sha1
func GetSha1(content []byte) string {
	h := sha1.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
