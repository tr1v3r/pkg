package log

import (
	"fmt"
	"strings"
	"time"
)

// NewFormatter create new formatter
func NewFormatter(color bool) *Formatter {
	return &Formatter{color: color}
}

type Formatter struct {
	color bool
}

func (f *Formatter) Format(l Level, logid, format string) string {
	var buf strings.Builder

	if f.color {
		buf.WriteString("\033[38;5;")
		buf.WriteString(fmt.Sprint(l.Color()))
		buf.WriteString("m")
	}

	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteByte(' ')

	buf.WriteByte('[')
	buf.WriteString(strings.ToUpper(l.String()))
	buf.WriteByte(']')
	buf.WriteByte(' ')

	if logid != "" {
		buf.WriteString(logid)
		buf.WriteByte(' ')
	}

	buf.WriteString(format)

	if f.color {
		buf.WriteString("\033[0m")
	}

	buf.WriteByte('\n')

	return buf.String()
}
