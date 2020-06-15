package ymratelimit

import (
	"github.com/juju/ratelimit"
	"testing"
	"time"
)

func BenchmarkYmretelimit(b *testing.B) {
	rl := NewTokenBucket(time.Second, 15) // per second

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rl.TakeAvailable()
	}
}

func BenchmarkParallelYmretelimit(b *testing.B) {
	rl := NewTokenBucket(time.Second, 15) // per second

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {

		for pb.Next() {
			rl.TakeAvailable()
		}
	})
}

func BenchmarkJujuRatelimit(b *testing.B) {
	rl := ratelimit.NewBucket(time.Second, 15)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rl.TakeAvailable(1)
	}
}

func BenchmarkParallelJujuRatelimit(b *testing.B) {
	rl := ratelimit.NewBucket(time.Second, 15)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {

		for pb.Next() {
			rl.TakeAvailable(1)
		}
	})
}
