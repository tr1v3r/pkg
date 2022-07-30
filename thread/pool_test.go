package thread

import (
	"fmt"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool := NewTimeoutPoolWithDefaults()
	for i := 0; i < 20; i++ {
		// 任务放入池中
		pool.Submit(&Job{
			Handler: func(v ...interface{}) {
				fmt.Println(v)
			},
			Params: []interface{}{i},
		})
	}

	pool.StartAndWait(time.Second * 5)
	fmt.Println("finished")
}
