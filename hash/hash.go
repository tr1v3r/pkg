package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

// CalcMD5 calculate md5
func CalcMD5(content []byte) string {
	h := md5.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA1 calculate sha1
func CalcSHA1(content []byte) string {
	h := sha1.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA224 calculate sha224
func CalcSHA224(content []byte) string {
	h := sha256.New224()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA256 calculate sha256
func CalcSHA256(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA384 calculate sha384
func CalcSHA384(content []byte) string {
	h := sha512.New384()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA3_224 calculate sha3_224
func CalcSHA3_224(content []byte) string {
	h := sha3.New224()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA3_256 calculate sha3_256
func CalcSHA3_256(content []byte) string {
	h := sha3.New256()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA3_384 calculate sha3_384
func CalcSHA3_384(content []byte) string {
	h := sha3.New384()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA3_512 calculate sha3_512
func CalcSHA3_512(content []byte) string {
	h := sha3.New512()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA512 calculate sha512
func CalcSHA512(content []byte) string {
	h := sha512.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA512_224 calculate sha512_224
func CalcSHA512_224(content []byte) string {
	h := sha512.New512_224()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CalcSHA512_256 calculate sha512_256
func CalcSHA512_256(content []byte) string {
	h := sha512.New512_256()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
