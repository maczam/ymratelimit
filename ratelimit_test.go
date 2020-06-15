package ymratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {

	rl := NewTokenBucket(time.Minute, 50) // per second

	startTime := time.Now()

	wg := sync.WaitGroup{}

	count := int32(0)
	stop := false
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			for !stop {
				if rl.TakeAvailable() {
					atomic.AddInt32(&count, 1)
				}
				time.Sleep(time.Millisecond * 100)
			}
			wg.Done()
		}()
	}

	time.Sleep(time.Second * 70)
	stop = true

	endTime := time.Now()

	wg.Wait()

	fmt.Println("t2与t1相差：", endTime.Sub(startTime), ";count：", count) //t2与t1相差： 50s
}

func TestNewLeakyBucket(t *testing.T) {

	rl := NewLeakyBucket(time.Second, 15) // per second

	startTime := time.Now()

	wg := sync.WaitGroup{}

	count := int32(0)

	stop := false
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			for !stop {
				if rl.TakeAvailable() {
					atomic.AddInt32(&count, 1)
				}
				//time.Sleep(time.Millisecond * 100)
			}
			wg.Done()
		}()
	}

	time.Sleep(time.Second * 5)
	stop = true

	endTime := time.Now()

	wg.Wait()

	fmt.Println("t2与t1相差：", endTime.Sub(startTime), ";count：", count) //t2与t1相差： 50s
}

