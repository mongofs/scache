package Scache

import (
	"container/list"
	"errors"
	"time"
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
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*sds)
		return kv.Value, true
	}
	return nil, false
}

func (c cache) Set(key string, value Value) {
	if value == nil || key == "" { return }
	c.set(key,value)
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

func (c cache) SetWithTTL(key string, content []byte, ttl time.Duration) error {
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
		c.expire(key,10 )
		// 如果查询不存在的话，调用闭包函数
		slow, err := f()
		if err != nil {
			return nil, err
		}
		c.set(targetKey, slow)
		return slow, nil
	} else {
		// 如果存在之前的请求设置了这个内容，需要阻塞等待消息返回
		return c.subscribe(targetKey)
	}

}

func (c cache) subscribe(key string) (Value, error) {
	ch := make (chan notify,1)
	if no ,ok := c.notify[key];ok {
		no.subscribe(ch)
	}
	res := <- ch
	return res.value,res.err
}

func (c cache) get(key string) (Value, bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*sds)
		return kv.Value, true
	}
	return nil, false
}

func (c cache) set(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*sds)
		c.nBytes += int64(value.Len()) - int64(kv.Value.Len())
		kv.Value = value
	} else {
		ele := c.ll.PushFront(&sds{key, 0,value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.removeOldest()
	}
}

func (c cache) expire(key string, ttl int64) {
	if v ,ok:=  c.cache[key]; ok{
		v.Value.(*sds).expire = time.Now().Unix() +ttl
	}
}

func (c cache) setNX(key string, value Value) bool {
	if _ ,ok:=  c.cache[key]; ok{
		return false
	}else{
		c.set(key,value)
	}
	return true
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

type observer struct {
	counter int // 订阅人数
	notify  []chan notify
}

func (o observer) publish(value notify) {
	for _ ,v := range  o.notify {
		v<- value
	}
}

func (o observer) subscribe(notify chan notify) {
	o.notify = append(o.notify, notify)
}

type notify struct {
	value Value
	err error
}