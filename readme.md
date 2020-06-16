# ymratelimit

目前在流量控制方面常用到的两个算法分别是，漏桶(Leaky bucket)[https://en.wikipedia.org/wiki/Leaky_bucket]与令牌桶(Token bucket)[https://en.wikipedia.org/wiki/Token_bucket]算法。
这两个算法在实现稍微有一点点不一样，链接里面wikipedia有详细解释，只是漏桶在实现过程中，会产生一个队列，允许瞬时并发。

为了提高并发，在性能方面优化:

> *  禁止使用锁
> *  每次请求最少只要一次cas操作
> *  所有计数都转化成int64的操作，尽量减少cpu额外计算浪费

测试对比对象： github.com/juju/ratelimit
```
➜  ymratelimit git:(master) ✗ go test -bench=. -run=none
goos: darwin
goarch: amd64
pkg: github.com/maczam/ymratelimit
BenchmarkYmretelimit-4                  14109680                79.9 ns/op
BenchmarkParallelYmretelimit-4          44515245                28.5 ns/op
BenchmarkJujuRatelimit-4                10214019               111 ns/op
BenchmarkParallelJujuRatelimit-4         6336103               160 ns/op
PASS
ok      github.com/maczam/ymratelimit   4.978s

➜  ymratelimit git:(master) ✗ go test -bench=. -benchmem -run=none
goos: darwin
goarch: amd64
pkg: github.com/maczam/ymratelimit
BenchmarkYmretelimit-4                  14484910                80.0 ns/op             0 B/op          0 allocs/op
BenchmarkParallelYmretelimit-4          42125070                27.6 ns/op             0 B/op          0 allocs/op
BenchmarkJujuRatelimit-4                10546452               111 ns/op               0 B/op          0 allocs/op
BenchmarkParallelJujuRatelimit-4         6592738               171 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/maczam/ymratelimit   5.034s

```

单线程串行，差不多，但是多线程并发是JujuRatelimit性能4倍。


# 使用
>  go get github.com/maczam/ymretelimit

## LeakyBucket
``` go
	rl := ymretelimit.NewLeakyBucket(time.Second, 15) // per second
    rl.TakeAvailable()
```

## TokenBucket
``` go
	rl := ymretelimit.NewTokenBucket(time.Microsecond, 15) // per Microsecond
    rl.TakeAvailable()
```

## LeakyBucket.TakeAvailable
``` go
for !taken {
		var newLast int64 = 0
		previousStatePointer := atomic.LoadPointer(&t.lastTokenTimestamp)
		lastTokenTimestamp := (*int64)(previousStatePointer)
		// 本次需要需要到达时间  当前时间戳-上次获取的时间戳 + 每次请求时间片段
		newLast = *lastTokenTimestamp + t.perRequest
		//sb.WriteString(fmt.Sprintf("lastTokenTimestamp %d;newLast:%v;", *lastTokenTimestamp, newLast))

		if now < newLast {
			break
		} else {
			// 如果下一个线程
			taken = atomic.CompareAndSwapPointer(&t.lastTokenTimestamp, previousStatePointer, unsafe.Pointer(&newLast))
		}
	}
```
## TokenBucket.TakeAvailable
``` go
for !taken {
		newStat := tokenBucketStat{}
		lastTokenBucketStatPointer := atomic.LoadPointer(&t.tokenBucketStat)
		lastTokenBucketStat := (*tokenBucketStat)(lastTokenBucketStatPointer)
		if now > lastTokenBucketStat.nextTokenTimestamp {
			newStat.nextTokenTimestamp = lastTokenBucketStat.nextTokenTimestamp + t.fillInterval
			newStat.keepCapacity = t.capacity - 1
		} else {

			// 已经没有了
			if lastTokenBucketStat.keepCapacity <= 0 {
				break
			} else {
				newStat.nextTokenTimestamp = lastTokenBucketStat.nextTokenTimestamp
				newStat.keepCapacity = lastTokenBucketStat.keepCapacity - 1
			}
		}
		taken = atomic.CompareAndSwapPointer(&t.tokenBucketStat, lastTokenBucketStatPointer, unsafe.Pointer(&newStat))

	}
```