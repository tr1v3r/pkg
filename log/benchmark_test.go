package log

import "testing"

func BenchmarkFormatter_Format_withColor(b *testing.B) {
	formatter := NewFormatter(true)
	var logData string
	for i := 0; i < 99; i++ {
		logData += "this is a log formatter benchmark test string"
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatter.Format(WarnLevel, "", logData)
	}
}

func BenchmarkFormatter_Format_withoutColor(b *testing.B) {
	formatter := NewFormatter(false)
	var logData string
	for i := 0; i < 99; i++ {
		logData += "this is a log formatter benchmark test string"
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatter.Format(WarnLevel, "", logData)
	}
}
