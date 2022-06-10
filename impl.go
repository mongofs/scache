package Scache

import (
	"container/list"
	"errors"
	"fmt"
	"math"
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

	ErrKeyAlreadyExist = errors.New("sCache : key already exist ")
	ErrKeyNotExist     = errors.New("sCache : key is not  exist ")
)

type cacheImpl struct {
	rw       sync.RWMutex
	maxBytes int64
	nBytes   int64
	ll       *list.List
	interval time.Duration
	cache    map[string]*list.Element

	// 当某个key被删除的时候的回调函数
	OnCaller func(key string, v Value)

	// 程序内部发生一些定时器调用错误，注册函数调用错误的时候会调用此函数
	// 实现此方法的时候需要判断param的length
	//	attention:
	// 当参数大于两个的时候，第一个参数是描述，第二个参数是详细的描述（包含recover的panic ）
	// 当参数为一个的时候就是error消息
	// 当参数为nil的时候可以不处理
	OnError func(...interface{})

	// singleFlight 管理器，
	regularManger RegularManger
}

func New(maxByte int64, clearInterval time.Duration, clearCall func(key string, value Value)) Cache {
	c := &cacheImpl{
		maxBytes: maxByte,
		nBytes:   0,
		ll:       list.New(),
		interval: clearInterval,
		cache:    make(map[string]*list.Element),
		OnError: func(i ...interface{}) {
			fmt.Println(i)
		},
		OnCaller:      clearCall,
		regularManger: NewRegularManager(),
	}
	c.clear()
	return c
}

func (c *cacheImpl) SetErrorHandler(handler func(...interface{})) {
	c.OnError = handler
}

func (c *cacheImpl) Get(key string) (Value, error) {
	return c.get(key)
}

func (c *cacheImpl) Set(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.set(key, value, 0)
}

func (c *cacheImpl) SetNX(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.setNx(key, value)
}

func (c *cacheImpl) SetEX(key string, value Value) error {
	if value == nil || key == "" {
		return ErrInValidParam
	}
	return c.setNx(key, value)
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
	return c.set(key, value, ttl)
}

func (c *cacheImpl) Register(regulation string, expire int, f /* slow way func */ func() (Value, error)) {
	if regulation == "" || f == nil || expire < 0 {
		panic(ErrInValidParam)
	}
	c.regularManger.Register(regulation, expire, f)
}

func (c *cacheImpl) RegisterCron(regulation string, flushInterval int, f /* slow way func */ func() (Value, error)) {
	if regulation == "" || f == nil || flushInterval < 0 {
		panic(ErrInValidParam)
	}
	err := c.ticker(regulation, flushInterval, f)
	if err != nil {
		panic(err)
	}
}

// =============================================concurrency safe =========================================

func (c *cacheImpl) ticker(regulation string, flushInterval int, f /* slow way func */ func() (Value, error)) error {
	if flushInterval <= 0 || regulation == "" {
		return ErrInValidParam
	}
	val, _, _, err := c.regularManger.Get(regulation)
	if err != nil {
		return err
	}
	if val != nil {
		return ErrKeyAlreadyExist
	}
	t := time.NewTicker(time.Duration(flushInterval) * time.Second)
	go func() {
		defer func() {
			// issue error #9
			if err := recover(); err != nil {
				c.OnError(fmt.Sprintf("regulation is %s", regulation), err)
			}
		}()
		for {
			v, er := f()
			if er != nil {
				if c.OnError != nil {
					c.OnError(er)
				}
			} else {
				c.set(regulation, v, 0)
			}

			<-t.C
		}
	}()
	return nil
}

func (c *cacheImpl) get(key string) (Value, error) {
	// 去获取值是否存在于map中且状态不为过期状态
	if val, ok := c.getDetection(key); ok {
		return val, nil
	}
	// 如果key 不存在cache中， 去查询regulation查看是否存在key
	val, shouldSave, expire, err := c.regularManger.Get(key)
	if err != nil {
		return nil, err
	}
	if shouldSave {
		if err = c.set(key, val, expire); err != nil {
			return nil, err
		}
	}
	return val, nil
}

// expire 拥有两个值0 ，大于0
// 1. 当值为0 的时候表示用不过期
// 2. 当值为大于0的时候表示，过期时间表示： time_now + expire
func (c *cacheImpl) set(key string, value Value, expire int) error {
	c.rw.Lock()
	defer c.rw.Unlock()
	if int64(value.Len()) > c.maxBytes {
		return ErrValueIsBiggerThanMaxByte
	}
	if ele, ok := c.getElem(key); ok {
		// 如果说这个值存在于Element，有两种情况：
		// 1. 这个值存在 ，但是已经过期
		// 2. 这个值正常
		kv := ele.Value.(*sds)
		oldLen := kv.Value.Len()
		kv.ReUse()
		if expire > 0 {
			kv.expire = int64(expire) + time.Now().Unix()
		} else {
			kv.expire = 0
		}
		kv.Value = value

		//当oldLen小于value.Len()的时候，相减变成负数，此时c.nBytes就有可能等于负数
		newLen := int64(oldLen - value.Len())
		if newLen < 0 {
			c.nBytes += int64(math.Abs(float64(newLen)))
		} else {
			c.nBytes -= newLen
			if c.nBytes < 0 {
				c.nBytes = 0
			}
		}
		//c.nBytes += int64(oldLen - value.Len())
	} else {
		// 创建新的sds结构体
		newSds := NewSDS(key, value, expire)
		eles := c.ll.PushFront(newSds)
		c.cache[key] = eles
		c.nBytes += int64(newSds.Calculation())
	}
	var freeBytes, freeElems int64
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		freeBytes += c.removeOldest()
		freeElems++
	}
	if freeBytes != 0 {
		fmt.Printf(" sCache : Garbage.Collection.removeOldest， free bytes  %v ,free Element %v \n\r", freeBytes, freeElems)
	}
	return nil
}

func (c *cacheImpl) del(key string, del bool) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if del {
		c.RealDel()
		return
	}
	if v, ok := c.getElem(key); ok {
		s := v.Value.(*sds)
		c.fakeDel(s)
	}
	return
}

func (c *cacheImpl) expire(key string, ttl int) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if v, ok := c.getElem(key); ok {
		v.Value.(*sds).expire = time.Now().Unix() + int64(ttl)
	}
}

func (c *cacheImpl) clear() {
	go func() {
		fmt.Printf("sCache : start the backend goroutine , intarvel is %v\n\r", c.interval)
		for {
			time.Sleep(c.interval)
			sin := time.Now()
			counter, free := c.RealDel()
			escape := time.Since(sin)
			if counter > 0 {
				fmt.Printf("sCache : clear once spend %v , clear %v element ,clear memory %v byte   \n\r", escape, counter, free)
			}
		}
	}()
}

func (c *cacheImpl) getDetection(key string) (Value, bool) {
	c.rw.RLock()
	defer c.rw.RUnlock()
	if ele, ok := c.getElem(key); ok {

		// flushKey 在读取elem的时候判断key值过期了没有，这里会出现一个问题
		// 如果某个值一直没被访问只能依靠lru进行淘汰，这里是需要改进的一个地方
		// todo 设置一个阈值，超过这个阈值的时候主动开启扫描过期的值，并清除掉

		s := ele.Value.(*sds)
		// 1. 这个key 标记为被删除,如果被标记删除了直接返回
		if s.Status() == SDSStatusDelete {
			return nil, false
		}
		// 2.查看是否过期，如果过期了，将key标注一下更新为过期
		if s.expire != 0 && s.expire < time.Now().Unix() {
			// todo 此时将值进行标准为删除
			// 第一个准则是存储的所有的内容都先不能删除，进行内存复用
			// 但是先进行回调删除方法，让用户感知
			// 内部内存是对用户不可见的，所以不需要告诉用户

			c.fakeDel(s)
			return nil, false
		}

		// 当key值存在的时候，需要将值的访问记录进行更新，
		c.ll.MoveToFront(ele)
		return s.Value, true
	}
	return nil, false
}

func (c *cacheImpl) setNx(key string, value Value) error {
	if _, ok := c.getDetection(key); ok {
		return ErrKeyAlreadyExist
	}
	return c.set(key, value, 0)
}

func (c *cacheImpl) setEx(key string, value Value) error {
	if _, ok := c.getDetection(key); !ok {
		return ErrKeyNotExist
	}
	c.rw.Lock()
	defer c.rw.Unlock()
	return c.set(key, value, 0)
}

// realDel For testing ,请勿直接调用 真删除，将标记出来的内容删除, 这是一个On操作，需要在后台线程上进行操作
// 删除，是一个On操作，需要判断所有element 的内容是否为delete
// 1. 需要将内容删除
// 2. 需要释放空间
// 3. 将元素释放
// todo 待优化
func (c *cacheImpl) RealDel() (int, int) {
	c.rw.Lock()
	defer c.rw.Unlock()
	fmt.Printf("sCache : RealDel current element counter %v ,current size %v  ,max cap %v \n\r", len(c.cache), c.nBytes, c.maxBytes)
	counter := 0
	free := 0
	for k, v := range c.cache {
		tev := v
		sd := tev.Value.(*sds)
		st := sd.Status()
		if st == SDSStatusDelete {
			c.ll.Remove(v)
			counter++
			delete(c.cache, k)
			freeCount := sd.Calculation()
			c.nBytes -= int64(freeCount)
			free += freeCount
			sd.Destroy()
		}

		if sd.expire != 0 && sd.expire < time.Now().Unix() && st == SDSStatusNormal {
			sd.Delete()
		}
	}
	return counter, free
}

//  =============================================concurrency not safe =========================================

// removeOldest 直接真删除，
// todo 之前的版本存在一个问题，realDeal后，实际上没有删除掉链表节点
func (c *cacheImpl) removeOldest() (freeByte int64) {
	ele := c.ll.Back()
	if ele != nil {
		// 删除 链表节点
		c.ll.Remove(ele)
		kv := ele.Value.(*sds)
		delete(c.cache, kv.key)
		freeByte = int64(kv.Calculation())
		c.nBytes -= freeByte
		if c.OnCaller != nil {
			c.OnCaller(kv.key, kv.Value)
		}
	}
	return
}

// getElem 并发不安全，需要加锁操作
func (c *cacheImpl) getElem(key string) (*list.Element, bool) {
	if ele, ok := c.cache[key]; ok {
		return ele, true
	}
	return nil, false
}

// fakeDel 假删除，将内容标记为删除
func (c *cacheImpl) fakeDel(sd *sds) {
	sd.Delete()
	if c.OnCaller != nil {
		c.OnCaller(sd.key, sd.Value)
	}
}
