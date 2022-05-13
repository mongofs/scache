/*
 * Copyright 2022 steven
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *    http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package Scache

import (
	"sync"
)

type OnCall func() (Value, error)

type Topic struct {

	// 保护notify的用户切片集合
	rw sync.RWMutex

	// notify 是订阅了这个主题的所有请求的集合
	notify []chan *notify

	// OnCall 观察者慢函数，所有订阅对象其实就是为了获取到这个方法的返回
	OnCall func() (Value, error)
}

// publish 发布具体的内容
func (o *Topic) publish(value *notify) {
	if len(o.notify) == 0 {
		return
	}
	o.rw.Lock()
	for _, v := range o.notify {
		tev := v
		tev <- value
	}
	o.rw.Unlock()
}

// subscribe 订阅这个频道
func (o *Topic) subscribe(notify chan *notify) {
	o.rw.Lock()
	o.notify = append(o.notify, notify)
	o.rw.Unlock()
}

type notify struct {
	value Value
	err   error
}

// 这里是一个分布式锁的快捷实现 ， short for distributed lock
type DLock struct {
	rw *sync.RWMutex

	// 这里是一个观察者，用户可以订阅，内部调用push 方法,用户请求到内部大概率会被
	// 阻塞到observer 当中，最好使
	obs map[string]*Topic

	// 标记是否存在用户
	flags map[string]struct{}

	// 超时时间，如果存在update方法特别慢，超过了expiretime的最大等待时间，那么
	// 就会返回超时错误
	expireTime int

	// 请求进入，会优先走到这个方法，这个方法后才会实现具体阻塞的方案，整体调用逻辑
	// subscribe -> observer.subscribe ,用户可以在这里去截断，去添加一些中间层
	// 比如添加整体耗时记录等操作
	Subscribe func(key string) (Value, error)
}

func NewLocker(expire int) *DLock {
	l := &DLock{
		rw:         &sync.RWMutex{},
		obs:        map[string]*Topic{},
		flags:      map[string]struct{}{},
		expireTime: 5,
		Subscribe:  nil,
	}
	return l
}

func (l *DLock) defaultSubscribe() {
	if l.Subscribe != nil {
		return
	}
	subscribe := func(key string) (Value, error) {
		ch := make(chan *notify, 1)
		err := l.RegisterNotify(key, ch)
		if err != nil {
			return nil, err
		}
		res := <-ch
		return res.value, res.err
	}
	l.Subscribe = subscribe
}

// RegisterObserver 必须提前定义好slow函数
func (l *DLock) RegisterObserver(call OnCall, topic string) (*Topic, error) {
	l.rw.Lock()
	defer l.rw.Unlock()
	if _, ok := l.obs[topic]; ok {
		return nil, ErrNotifyNotExist
	}
	ob := &Topic{
		notify: []chan *notify{},
		OnCall: call,
	}
	l.obs[topic] = ob
	return ob, nil
}

// RegisterNotify 注册异步通知的回调函数
func (l *DLock) RegisterNotify(topic string, ch chan *notify) error {
	l.rw.RLock()
	defer l.rw.RUnlock()
	ob, ok := l.obs[topic]
	if !ok {
		return ErrTopicAlreadyExist
	}
	ob.subscribe(ch)
	return nil
}

// Notify
func (l *DLock) Notify(topicName string, no *notify) {
	l.rw.Lock()
	defer l.rw.Unlock()
	v, ok := l.obs[topicName]
	if ok {
		v.publish(no)
	}
	delete(l.flags, topicName)
	delete(l.obs, topicName)
}

// 创建锁进行建立请求 ,set topic if not exist
func (l *DLock) setNx(topic string) bool {
	l.rw.Lock()
	defer l.rw.Unlock()
	if _, ok := l.flags[topic]; ok {
		// 当锁存在的时候，则不能简单的删除
		return false
	} else {
		// 当锁不存在的时候，
		l.flags[topic] = struct{}{}
		return true
	}
}

// 调用这个的时候可以是本地缓存中没有这个内容了
// if !exist {
// 	 locker.Get(targetKey)
// }
func (l *DLock) Get(topicName string, items ...interface{}) (Value, error) {

	if l.Subscribe == nil {
		l.defaultSubscribe()
	}

	if topicName == "" {
		return nil, ErrInValidParam
	}
	if len(items) <1  {
		return nil, ErrInValidParam
	}
	Call, ok := items[0].(OnCall)
	if !ok {
		return nil, ErrBadConvertParamToCall
	}

	if l.setNx(topicName) {
		// 设置锁成功，执行慢操作
		slow, err := l.RegisterObserver(Call, topicName)
		if err != nil {
			return nil, err
		}
		v, err := slow.OnCall()
		if err != nil {
			return nil, err
		}
		// 发布成功内容
		niy := &notify{
			value: v,
			err:   err,
		}
		l.Notify(topicName, niy)
		return v, err
	} else {
		// 设置锁失败发起订阅
		return l.Subscribe(topicName)
	}
}
