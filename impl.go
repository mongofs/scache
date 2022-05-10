package Scache

import (
	"container/list"
	"errors"
	"time"
)

var (
	ErrValueIsBiggerThanMaxByte = errors.New("sCache : value size is bigger than maxBytes  ")
	ErrInValidParam             = errors.New("sCache : the param is invalid")
)

type cache struct {
	maxBytes int64
	nBytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	OnCaller func(key string, v Value)

	notify map[string]observer
}

func New(maxByte int64, OnCaller func(key string, value Value)) Cache {
	return &cache{
		maxBytes: maxByte,
		nBytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		OnCaller: OnCaller,
	}
}

func (c cache) Get(key string) (Value, bool) {
	return c.get(key)
}

func (c cache) Set(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.set(key, value)
}

func (c cache) Del(key string) (err error) {
	//删除某个key
	if v, ok := c.cache[key]; !ok {
		return errors.New("key is not exist ")
	} else {
		res := v.Value.(*sds)
		c.nBytes -= int64(res.Value.Len())
		c.ll.Remove(v)
		delete(c.cache, key)
		if c.OnCaller != nil {
		}
	}
	return
}

func (c cache) Expire(key string, ttl int64) {
	c.expire(key, ttl)
}

func (c cache) SetWithTTL(key string, content []byte, ttl int64) error {
	panic("implement me")
}

func (c cache) GetTargetKeyLockerWithTimeOut(targetKey string, f func() (Value, error)) (Value, error) {
	if key, ok := c.get(targetKey); ok {
		return key, nil
	}

	// 这里启用单线程更新
	key := "update:" + targetKey
	var v defaultIntValue = 1
	if ok := c.setNX(key, v); ok {
		c.expire(key, 10)
		// 如果查询不存在的话，调用闭包函数
		slow, err := f()
		if err != nil {
			return nil, err
		}
		c.set(targetKey, slow)
		// 需要将内容进行publish
		notify := &notify{
			value: slow,
			err:   err,
		}
		c.publish(targetKey, notify)
		return slow, nil
	} else {
		// 如果存在之前的请求设置了这个内容，需要阻塞等待消息返回
		return c.subscribe(targetKey)
	}

}

func (c cache) publish(key string, notify *notify) {
	v, ok := c.notify[key]
	if ok {
		v.publish(notify)
	}
	// todo log the other side ,warning
}

func (c cache) subscribe(key string) (Value, error) {
	ch := make(chan *notify, 1)
	if no, ok := c.notify[key]; ok {
		no.subscribe(ch)
	}
	res := <-ch
	return res.value, res.err
}

// =============================================concurrency safe =========================================

// get 并发安全
func (c cache) get(key string) (Value, bool) {
	if ele, ok := c.getElem(key); ok {
		if c.flushKey(ele) {
			c.moveElemToFront(ele)
			kv := ele.Value.(*sds)
			return kv.Value, true
		} else {
			return nil, false
		}
	}
	return nil, false
}

// set 并发安全 todo 长度依旧是并发不安全的内容
func (c cache) set(key string, value Value) (err error) {
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
		newSds := &sds{key, 0, SDSStatusNormal, value}
		ele := c.pushElemFront(newSds)
		c.cache[key] = ele
		c.nBytes += int64(newSds.Calculation())
	}

	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.removeOldest()
	}
	return
}

//  =============================================concurrency not safe =========================================

// 并发不安全的
func (c cache) moveElemToFront(element *list.Element) {
	c.ll.MoveToFront(element)
}

func (c cache) pushElemFront(content interface{}) *list.Element {
	return c.ll.PushFront(content)
}

func (c cache) removeOldest() {
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
func (c cache) getElem(key string) (*list.Element, bool) {
	if ele, ok := c.cache[key]; ok {
		return ele, true
	}
	return nil, true
}

// fakeDel 假删除，将内容标记为删除
func (c cache) fakeDel(key string) {
	if ele, ok := c.getElem(key); ok {
		// 如果内容存在的话，需要将内容标记为删除,这里需要加锁 todo
		v := ele.Value.(*sds)
		v.Delete()
		c.ll.Remove(ele)
		c.nBytes -= int64(v.Value.Len())
	}
	return
}

// realDel 真删除，将标记出来的内容删除, 这是一个On操作，需要在后台线程上进行操作
func (c cache) realDel() {
	// 需要加锁，todo
	for k, v := range c.cache {
		tev := v
		st := tev.Value.(*sds).Status()
		if st == SDSStatusDelete {
			delete(c.cache, k)
		}
	}
}

func (c cache) flushKey(ele *list.Element) bool {
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

func (c cache) expire(key string, ttl int64) {
	if ttl <= 0 {
		return
	}
	if v, ok := c.getElem(key); ok {
		v.Value.(*sds).expire = time.Now().Unix() + ttl
	}
}

func (c cache) setNX(key string, value Value) bool {
	if _, ok := c.getElem(key); ok {
		return false
	} else {
		c.set(key, value)
	}
	return true
}

type observer struct {
	counter int // 订阅人数
	notify  []chan *notify
}

func (o observer) publish(value *notify) {
	for _, v := range o.notify {
		v <- value
	}
}

func (o observer) subscribe(notify chan *notify) {
	o.notify = append(o.notify, notify)
}

type notify struct {
	value Value
	err   error
}
