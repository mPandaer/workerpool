package workerpool

import (
	"fmt"
	"testing"
	"time"
)

func TestPoolV1(t *testing.T) {
	p := New(5)

	for i := 0; i < 10; i++ {
		err := p.Schedule(func() {
			time.Sleep(time.Second * 3)
		})
		fmt.Printf("启动第%d个任务\n", i+1)
		if err != nil {
			println("task: ", i, "err: ", err)
		}
	}
	p.Free()
}
