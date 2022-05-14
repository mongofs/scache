package Scache

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrValueIsBiggerThanMaxByte = errors.New("sCache : value size is bigger than maxBytes  ")
	ErrInValidParam             = errors.New("sCache : the param is invalid")
	ErrNotifyNotExist           = errors.New("sCache : notify is not exist")
	ErrTopicAlreadyExist        = errors.New("sCache : Topic  already exist")
	ErrBadConvertParamToCall    = errors.New("sCache : Can't Convert Param item[0] to Call")

	ErrRegulationAlreadyExist = errors.New("sCache : regulation already exist ")

	ErrKeyAlreadyExist =errors.New("sCache : key already exist ")
	ErrKeyNotExist =errors.New("sCache : key is not  exist ")
)

type cacheImpl struct {
	rw       sync.RWMutex
	maxBytes int64
	nBytes   int64
	ll       *list.List
	interval time.Duration
	cache    map[string]*list.Element

	// 当某个key被删除的时候的回调函数
	OnCaller      func(key string, v Value)
	RegularManger RegularManger
}




func New(maxByte int64, clearInterval time.Duration, OnCaller func(key string, value Value)) *cacheImpl {
	c := &cacheImpl{
		maxBytes: maxByte,
		nBytes:   0,
		ll:       list.New(),
		interval: clearInterval,
		cache:    make(map[string]*list.Element),
		OnCaller: OnCaller,
		RegularManger: NewRegularManager(),
	}
	c.clearParallel()
	return c
}

func (c *cacheImpl) Get(key string) (Value, error) {
	return c.get(key)
}

func (c *cacheImpl) Set(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.set(key, value)
}

func (c *cacheImpl) SetNX(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.setNx(key,value)
}

func (c *cacheImpl) SetEX(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.setNx(key,value)
}

func (c *cacheImpl) Del(key string) {
	c.del(key, false)
}

func (c *cacheImpl) Expire(key string, ttl int) {
	if ttl <= 0 || key == "" {
		return
	}
	c.expire(key, ttl)
}

func (c *cacheImpl) SetWithTTL(key string, value Value, ttl int) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.setWithTTL(key, value, ttl)
}

func (c *cacheImpl) Register(regulation string, expire int, f func() (Value, error)) error {
	if regulation == "" || f == nil {
		return ErrInValidParam
	}
	return c.RegularManger.Register(regulation, expire, f)
}

// =============================================concurrency safe =========================================

// get 并发安全，查询key 对应的value值，并在查询的时候进行值状态判断
func (c *cacheImpl) get(key string) (Value, error) {
	if val, ok := c.beforeGet(key); ok {
		return val, nil
	}
	// 如果key 不存在cache中， 去查询regulation查看是否存在key
	val, shouldSave, err := c.RegularManger.Get(key)
	if err != nil {
		return nil, err
	}
	if shouldSave {
		c.rw.Lock()
		defer c.rw.Unlock()
		if err = c.unsafeSet(key, val, 0); err != nil {
			return nil, err
		}
	}
	return val, nil
}

// set 并发安全 ,设置一个值，需要考虑值存在的时候更新和值不存在的时候
// 添加
func (c *cacheImpl) set(key string, value Value) (err error) {
	c.rw.Lock()
	defer c.rw.Unlock()
	return c.unsafeSet(key, value, 0)
}

// 设置一个值并携带ttl 的过期时间
func (c *cacheImpl) setWithTTL(key string, value Value, ttl int) error {
	if err := c.set(key, value); err != nil {
		return err
	}
	c.expire(key, ttl)
	return nil
}

// del 删除某个key，
func (c *cacheImpl) del(key string, del bool) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if del {
		c.RealDel()
		return
	}
	c.fakeDel(key)
}

// expire 过期某个key
func (c *cacheImpl) expire(key string, ttl int) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if v, ok := c.getElem(key); ok {
		v.Value.(*sds).expire = time.Now().Unix() + int64(ttl)
	}
}

// monitor监控
func (c *cacheImpl) clearParallel() {
	go func() {
		fmt.Printf("sCache : start the backend goroutine , intarvel is %v\n\r", c.interval)
		for {
			time.Sleep(c.interval)
			sin := time.Now()
			c.rw.Lock()
			counter := c.RealDel()
			c.rw.Unlock()
			escape := time.Since(sin)
			fmt.Printf("sCache : clear once spend %v , clear %v element  \n\r", escape, counter)
		}
	}()
}

func (c *cacheImpl) beforeGet(key string) (Value, bool) {
	c.rw.RLock()
	defer c.rw.RUnlock()
	if ele, ok := c.getElem(key); ok {

		// flushKey 在读取elem的时候判断key值过期了没有，这里会出现一个问题
		// 如果某个值一直没被访问只能依靠lru进行淘汰，这里是需要改进的一个地方
		// todo 设置一个阈值，超过这个阈值的时候主动开启扫描过期的值，并清除掉

		if c.flushKey(ele) {
			// 当key值存在的时候，需要将值的访问记录进行更新，
			c.ll.MoveToFront(ele)
			kv := ele.Value.(*sds)
			return kv.Value, true
		} else {
			// 这里代表我们的key存在但是已经过期
			return nil, false
		}
	}
	return nil, false
}

func  (c * cacheImpl)setNx (key string,value Value)  error{
	if _,ok := c.beforeGet(key);ok {
		return ErrKeyAlreadyExist
	}
	c.rw.Lock()
	defer c.rw.Unlock()
	return c.unsafeSet(key,value,0)
}


func (c *cacheImpl) setEx (key string ,value Value)error {
	if _,ok := c.beforeGet(key);!ok {
		return ErrKeyNotExist
	}
	c.rw.Lock()
	defer c.rw.Unlock()
	return c.unsafeSet(key,value,0)
}

//  =============================================concurrency not safe =========================================

func (c *cacheImpl) unsafeSet(key string, value Value, expire int) error {
	if int64(value.Len()) > c.maxBytes {
		return ErrValueIsBiggerThanMaxByte
	}
	if ele, ok := c.getElem(key); ok {
		// 如果说这个值存在，那么需要进行值的覆盖
		kv := ele.Value.(*sds)
		oldSt := kv.Status()
		oldLen := kv.Value.Len()
		kv.ReUse()
		kv.expire = int64(expire)
		kv.Value = value
		if oldSt == SDSStatusDelete {
			// 表示之前的内容被删除过了，新值直接增加长度即可
			c.nBytes += int64(kv.Calculation())
		} else {
			// 如果之前的内容存在的话
			c.nBytes += int64(oldLen - value.Len())
		}
	} else {
		// 创建新的sds结构体
		newSds := NewSDS(key, value, expire)
		ele := c.ll.PushFront(newSds)
		c.cache[key] = ele
		c.nBytes += int64(newSds.Calculation())
	}

	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.removeOldest()
	}
	return nil
}

// removeOldest 移除最老的内容
func (c *cacheImpl) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*sds)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.Value.Len())
		if c.OnCaller != nil {
			c.OnCaller(kv.key, kv.Value)
		}
	}
}

// getElem 并发不安全，需要加锁操作
func (c *cacheImpl) getElem(key string) (*list.Element, bool) {
	if ele, ok := c.cache[key]; ok {
		return ele, true
	}
	return nil, false
}

// fakeDel 假删除，将内容标记为删除
func (c *cacheImpl) fakeDel(key string) {
	if ele, ok := c.getElem(key); ok {
		v := ele.Value.(*sds)
		v.Delete()
		c.ll.Remove(ele)
		c.nBytes -= int64(v.Value.Len())
	}
	return
}

// realDel For testing ,请勿直接调用 真删除，将标记出来的内容删除, 这是一个On操作，需要在后台线程上进行操作
// 这个并非暴露的接口
func (c *cacheImpl) RealDel() int {
	counter := 0
	for k, v := range c.cache {
		tev := v
		st := tev.Value.(*sds).Status()
		if st == SDSStatusDelete {
			counter++
			delete(c.cache, k)
		}
	}
	return counter
}

// flushKey 更新key的状态，如果状态是ok的
func (c *cacheImpl) flushKey(ele *list.Element) bool {
	s := ele.Value.(*sds)
	// 这个key 标记为被删除
	if s.Status() == SDSStatusDelete {
		return false
	}
	// 将这个key 标记为删除，后台线程去进行删除
	if s.expire != 0 && s.expire < time.Now().Unix() {
		s.Delete()
		c.ll.Remove(ele)
		c.nBytes -= int64(s.Calculation())
		return false
	}
	return true
}
