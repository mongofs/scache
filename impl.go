package Scache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var (
	ErrValueIsBiggerThanMaxByte = errors.New("sCache : value size is bigger than maxBytes  ")
	ErrInValidParam             = errors.New("sCache : the param is invalid")
	ErrNotifyNotExist           = errors.New("sCache : notify is not exist")
	ErrTopicAlreadyExist           = errors.New("sCache : Topic  already exist")
	ErrBadConvertParamToCall 	= errors.New("sCache : Can't Convert Param item[0] to Call")
)

type cacheImpl struct {
	rw       sync.RWMutex
	maxBytes int64
	nBytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	OnCaller func(key string, v Value)

	locker *Locker
}

func New(maxByte int64, OnCaller func(key string, value Value)) *cacheImpl {
	return &cacheImpl{
		maxBytes: maxByte,
		nBytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		OnCaller: OnCaller,
	}
}

func (c *cacheImpl) Get(key string) (Value, bool) {
	return c.get(key)
}

func (c *cacheImpl) Set(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.set(key, value)
}

func (c *cacheImpl) Del(key string) {
	c.del(key,false)
}

func (c *cacheImpl) Expire(key string, ttl int64) {
	c.expire(key, ttl)
}

func (c *cacheImpl) SetWithTTL(key string, content []byte, ttl int64) error {
	panic("implement me")
}

// 一个自带分布式锁的内容
func (c *cacheImpl) GetTargetKeyLockerWithTimeOut(targetKey string, f func() (Value, error)) (Value, error) {
	return c.locker.Get(targetKey,f)
}

// =============================================concurrency safe =========================================



// get 并发安全
func (c *cacheImpl) get(key string) (Value, bool) {
	c.rw.RLock()
	defer c.rw.RUnlock()
	if ele, ok := c.getElem(key); ok {
		if c.flushKey(ele) {
			c.ll.MoveToFront(ele)
			kv := ele.Value.(*sds)
			return kv.Value, true
		} else {
			return nil, false
		}
	}
	return nil, false
}

// set 并发安全 ,设置一个值，需要考虑值存在的时候更新和值不存在的时候
// 添加
func (c *cacheImpl) set(key string, value Value) (err error) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if int64(value.Len()) > c.maxBytes {
		return ErrValueIsBiggerThanMaxByte
	}
	if ele, ok := c.getElem(key); ok {
		kv := ele.Value.(*sds)
		// 可能key存在，但是被标记为已删除，此时只需要从新覆盖这个值
		if kv.Status() == SDSStatusDelete {
			kv.ReUse()
			kv.expire = 0
			kv.Value = value
			c.nBytes += int64(kv.Calculation())
		} else {
			oldV := kv.Value
			kv.Value = value
			// todo 重新设置后将过期时间置零，有待考证
			kv.expire = 0
			c.nBytes += int64(oldV.Len() - value.Len())
		}
	} else {
		// 创建新的sds结构体
		newSds := NewSDS(key, value)
		ele := c.ll.PushFront(newSds)
		c.cache[key] = ele
		c.nBytes += int64(newSds.Calculation())
	}

	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.removeOldest()
	}
	return
}


func (c *cacheImpl) del(key string, del bool) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if del {
		c.RealDel()
		return
	}
	c.fakeDel(key)
}

//  =============================================concurrency not safe =========================================

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
func (c *cacheImpl) RealDel() {
	for k, v := range c.cache {
		tev := v
		st := tev.Value.(*sds).Status()
		if st == SDSStatusDelete {
			delete(c.cache, k)
		}
	}
}

// flushKey 更新key的状态，如果状态是ok的
func (c *cacheImpl) flushKey(ele *list.Element) bool {
	s := ele.Value.(*sds)
	// 这个key 标记为被删除
	if s.Status() == SDSStatusDelete {
		return false
	}
	// 将这个key 标记为删除，后台线程去进行删除
	if s.expire != 0 && s.expire > time.Now().Unix() {
		s.Delete()
		c.ll.Remove(ele)
		return false
	}
	return true
}

func (c *cacheImpl) expire(key string, ttl int64) {
	if ttl <= 0 {
		return
	}
	if v, ok := c.getElem(key); ok {
		v.Value.(*sds).expire = time.Now().Unix() + ttl
	}
}

func (c *cacheImpl) setNX(key string, value Value) bool {
	if _, ok := c.getElem(key); ok {
		return false
	} else {
		c.set(key, value)
	}
	return true
}

