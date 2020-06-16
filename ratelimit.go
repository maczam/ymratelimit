package ymratelimit

import (
	"encoding/json"
	"sync/atomic"
	"time"
	"unsafe"
)

/**
  抽象接口
*/
type Limiter interface {
	TakeAvailableWithNow(now int64) bool
	TakeAvailable() bool
	GetCapacity() int64
	GetLegacyCapacity() int64
}

/**
  漏桶算法能够强行限制数据的传输速率。
*/
type leakyBucket struct {
	capacity           int64
	fillInterval       int64 //统计周期
	lastTokenTimestamp unsafe.Pointer
	perRequest         int64 //计算出每次token占用的时间片段
}

/**
  如果能获取，那么不用判断时间， time.Now().UnixNano() 必须使用UnixNano
*/
func (t *leakyBucket) TakeAvailableWithNow(now int64) bool {

	// for 是为了保证LoadPointer和CompareAndSwapPointer是处于原子状态
	taken := false
	//sb := strings.Builder{}
	//t2 := time.Unix(0, now)

	//sb.WriteString(fmt.Sprintf("now %d;nowTime:%v;PerRequest:%d;", now, t2, t.PerRequest))
	for !taken {
		var newLast int64 = 0
		previousStatePointer := atomic.LoadPointer(&t.lastTokenTimestamp)
		lastTokenTimestamp := (*int64)(previousStatePointer)
		// 本次需要需要到达时间  当前时间戳-上次获取的时间戳 + 每次请求时间片段
		newLast = *lastTokenTimestamp + t.perRequest
		//sb.WriteString(fmt.Sprintf("lastTokenTimestamp %d;newLast:%v;", *lastTokenTimestamp, newLast))

		if now < newLast {
			break
			//sb.WriteString("now < newLast;")
		} else {
			// 如果下一个线程
			//sb.WriteString("now < newLast;")
			taken = atomic.CompareAndSwapPointer(&t.lastTokenTimestamp, previousStatePointer, unsafe.Pointer(&newLast))
		}
	}
	//sb.WriteString(fmt.Sprintf("最终结果:%t", taken))
	//fmt.Println(sb.String())
	return taken
}

func (t *leakyBucket) TakeAvailable() bool {
	return t.TakeAvailableWithNow(time.Now().UnixNano())
}

func (t *leakyBucket) GetCapacity() int64 {
	return t.capacity
}
func (t *leakyBucket) GetLegacyCapacity() int64 {
	return -1
}

func (t *leakyBucket) MarshalJSON() ([]byte, error) {
	object := map[string]interface{}{}
	object["capacity"] = t.capacity
	return json.Marshal(object)
}

func NewLeakyBucket(fillInterval time.Duration, capacity int64) Limiter {
	fillIntervalInt := int64(fillInterval)
	l := &leakyBucket{
		fillInterval: fillIntervalInt,
		perRequest:   fillIntervalInt / capacity,
		capacity:     capacity,
	}
	lastTokenTimestamp := time.Now().UnixNano()
	l.lastTokenTimestamp = unsafe.Pointer(&lastTokenTimestamp)
	return l
}

/**
令牌桶算法能够在限制数据的平均传输速率的同时还允许某种程度的突发传输。
*/
type tokenBucket struct {
	capacity        int64
	fillInterval    int64          //统计周期
	tokenBucketStat unsafe.Pointer //当前状态
	perRequest      int64          //计算出每次token占用的时间片段
}

/**
  当前状态
*/
type tokenBucketStat struct {
	nextTokenTimestamp int64
	keepCapacity       int64 //本次time window 还剩下多少次
}

/**
如果能获取，那么不用判断时间， time.Now().UnixNano() 必须使用UnixNano
*/
func (t *tokenBucket) TakeAvailableWithNow(now int64) bool {

	// for 是为了保证LoadPointer和CompareAndSwapPointer是处于原子状态
	taken := false
	//sb := strings.Builder{}
	//t2 := time.Unix(0, now)
	//
	//sb.WriteString(fmt.Sprintf("now %d;nowTime:%v", now, t2))
	for !taken {
		lastTokenBucketStatPointer := atomic.LoadPointer(&t.tokenBucketStat)
		lastTokenBucketStat := (*tokenBucketStat)(lastTokenBucketStatPointer)

		//sb.WriteString(fmt.Sprintf("下个窗口时间:%d;", lastTokenBucketStat.NextTokenTimestamp))
		//sb.WriteString(fmt.Sprintf("距离下一次:%d;", now-lastTokenBucketStat.NextTokenTimestamp))
		//sb.WriteString(fmt.Sprintf("lastKeepCapacity:%d;", lastTokenBucketStat.KeepCapacity))

		if now > lastTokenBucketStat.nextTokenTimestamp {
			newStat := tokenBucketStat{}
			newStat.nextTokenTimestamp = lastTokenBucketStat.nextTokenTimestamp + t.fillInterval
			newStat.keepCapacity = t.capacity - 1
			//sb.WriteString(fmt.Sprintf("改写下一次时间:%d;下一次容量:%d;", newStat.NextTokenTimestamp, newStat.KeepCapacity))
			taken = atomic.CompareAndSwapPointer(&t.tokenBucketStat, lastTokenBucketStatPointer, unsafe.Pointer(&newStat))
		} else {
			//sb.WriteString(fmt.Sprintf("在时间窗口之内;"))

			// 已经没有了
			if lastTokenBucketStat.keepCapacity > 0 {
				newStat := tokenBucketStat{}
				newStat.nextTokenTimestamp = lastTokenBucketStat.nextTokenTimestamp
				newStat.keepCapacity = lastTokenBucketStat.keepCapacity - 1
				//sb.WriteString(fmt.Sprintf(fmt.Sprintf("修改结余:%d;", newStat.KeepCapacity)))
				taken = atomic.CompareAndSwapPointer(&t.tokenBucketStat, lastTokenBucketStatPointer, unsafe.Pointer(&newStat))
			} else {
				break
			}
		}
	}
	//sb.WriteString(fmt.Sprintf("最终结果:%t", taken))
	//fmt.Println(sb.String())
	return taken
}

func (t *tokenBucket) TakeAvailable() bool {
	return t.TakeAvailableWithNow(time.Now().UnixNano())
}

func (t *tokenBucket) GetCapacity() int64 {
	return t.capacity
}

func (t *tokenBucket) GetLegacyCapacity() int64 {
	lastTokenBucketStatPointer := atomic.LoadPointer(&t.tokenBucketStat)
	lastTokenBucketStat := (*tokenBucketStat)(lastTokenBucketStatPointer)
	return lastTokenBucketStat.keepCapacity
}

func (t *tokenBucket) MarshalJSON() ([]byte, error) {
	object := map[string]interface{}{}
	object["capacity"] = t.capacity
	lastTokenBucketStatPointer := atomic.LoadPointer(&t.tokenBucketStat)
	lastTokenBucketStat := (*tokenBucketStat)(lastTokenBucketStatPointer)
	object["keepCapacity"] = lastTokenBucketStat.keepCapacity
	return json.Marshal(object)
}

/**
令牌桶算法能够在限制数据的平均传输速率的同时还允许某种程度的突发传输。
*/
func NewTokenBucket(fillInterval time.Duration, capacity int64) Limiter {
	fillIntervalInt := int64(fillInterval)
	l := &tokenBucket{
		fillInterval: fillIntervalInt,
		capacity:     capacity,
	}
	tokenBucketStat := tokenBucketStat{
		nextTokenTimestamp: time.Now().UnixNano(),
		keepCapacity:       capacity,
	}
	l.tokenBucketStat = unsafe.Pointer(&tokenBucketStat)
	return l
}
