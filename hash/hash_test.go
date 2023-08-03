package hash

import "testing"

func TestCalcHash(t *testing.T) {
	data := []byte("hello world")

	var hashMap = map[string]func(content []byte) string{
		"md5":        CalcMD5,
		"sha1":       CalcSHA1,
		"sha224":     CalcSHA224,
		"sha256":     CalcSHA256,
		"sha384":     CalcSHA384,
		"sha3_224":   CalcSHA3_224,
		"sha3_256":   CalcSHA3_256,
		"sha3_384":   CalcSHA3_384,
		"sha3_512":   CalcSHA3_512,
		"sha512":     CalcSHA512,
		"sha512_224": CalcSHA512_224,
		"sha512_256": CalcSHA512_256,
	}

	for name, hash := range hashMap {
		t.Logf("clac hash %s\t: %s", name, hash(data))
	}
}

func BenchmarkCalcMD5(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcMD5(data)
	}
}
func BenchmarkCalcSHA1(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA1(data)
	}
}
func BenchmarkCalcSHA224(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA224(data)
	}
}
func BenchmarkCalcSHA256(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA256(data)
	}
}
func BenchmarkCalcSHA384(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA384(data)
	}
}
func BenchmarkCalcSHA3_224(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA3_224(data)
	}
}
func BenchmarkCalcSHA3_256(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA3_256(data)
	}
}
func BenchmarkCalcSHA3_384(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA3_384(data)
	}
}
func BenchmarkCalcSHA3_512(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA3_512(data)
	}
}
func BenchmarkCalcSHA512(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA512(data)
	}
}
func BenchmarkCalcSHA512_224(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA512_224(data)
	}
}
func BenchmarkCalcSHA512_256(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalcSHA512_256(data)
	}
}
