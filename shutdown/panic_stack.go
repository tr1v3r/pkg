package shutdown

import (
	"fmt"
	"runtime"
)

// CatchStack catch stack info
func CatchStack() string {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	return fmt.Sprintf("==> %s\n", string(buf[:n]))
}
