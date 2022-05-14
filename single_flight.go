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
	"errors"
	"sync"
	"time"
)

var (
	ErrTopicNotExist = errors.New("sCache : topic not exist ")
)


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

type SingleFlight interface {
	Get(regulation string, slowWay func()(Value,error)) (Value, bool, error)
}

// 这里是一个分布式锁的快捷实现 ， short for distributed lock
type defaultSingleFlight struct {
	rw *sync.RWMutex

	// 这里是一个观察者，用户可以订阅，内部调用push 方法,用户请求到内部大概率会被
	// 阻塞到observer 当中，最好使
	obs map[string]*Topic

	flag map[string]struct{}

	// 超时时间，如果存在update方法特别慢，超过了expiretime的最大等待时间，那么
	// 就会返回超时错误
	MaxWaitTime int
}

func NewSingleFlight(maxWaitTime int) SingleFlight {
	l := &defaultSingleFlight{
		rw:          &sync.RWMutex{},
		obs:         map[string]*Topic{},
		flag:        map[string]struct{}{},
		MaxWaitTime: maxWaitTime,
	}
	return l
}

func (l *defaultSingleFlight) subscribe(key string) (Value, bool, error) {
	ch := make(chan *notify, 1)
	err := l.registerNotify(key, ch)
	if err != nil {
		return nil, false, err
	}
	res := <-ch
	return res.value, false, res.err
}

// RegisterObserver 必须提前定义好slow函数
func (l *defaultSingleFlight) registerTopic(call func()(Value,error), topic string) (*Topic, error) {
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
func (l *defaultSingleFlight) registerNotify(topic string, ch chan *notify) error {
	l.rw.RLock()
	defer l.rw.RUnlock()
	ob, ok := l.obs[topic]
	if !ok {
		// 如果两个请求拿锁，临界时间非常短，如果此时slow的注册topic时间耗时比较长
		// 超过了注册notify的时间，那么很有可能没有拿到锁的请求到达这里，所以会导致
		// 报错，这里处理方案是休眠10ms，等待对象创建成功，进行重试
		time.Sleep(10 * time.Microsecond)
		ob, ok = l.obs[topic]
		if !ok {
			return ErrTopicNotExist
		}
	}
	ob.subscribe(ch)
	return nil
}

// Notify
func (l *defaultSingleFlight) notify(topicName string, no *notify) {
	l.rw.Lock()
	defer l.rw.Unlock()
	// 将这步操作从最后放在第一排
	v, ok := l.obs[topicName]
	if ok {
		v.publish(no)
	}
	delete(l.obs, topicName)
	delete(l.flag,topicName)
}

func (l *defaultSingleFlight)setNx(topicName string)  bool{
	l.rw.Lock()
	defer l.rw.Unlock()
	if _,ok :=l.flag[topicName];ok {
		return false
	}
	l.flag[topicName] = struct{}{}
	return true
}

func (l *defaultSingleFlight) Get(topicName string, slow func()(Value,error)) (Value, bool, error) {
	if topicName == ""  || slow == nil{
		return nil, false, ErrInValidParam
	}

	// 对标志位进行加锁操作，标志位设置成功，就进入slowWay 如果标志位设置失败就进入
	// waitWay, 这里需要注意临界值的判断，所以对资源颗粒度相对较小，要求比较高
	if l.setNx(topicName) {
		// 1 .创建topic ，此操作占用锁,占用排他锁进行Topic
		// 创建，此时读请求、写请求都将被阻塞，知道此操作释放
		slowWay, err := l.registerTopic(slow, topicName)
		if err != nil {
			return nil, false, err
		}

		// 2 .执行慢操作,此操作不占用锁
		v, err := slowWay.OnCall()
		if err != nil {
			return nil, false, err
		}
		niy := &notify{
			value: v,
			err:   err,
		}
		// 3 .发布成功内容，此操作占用写锁
		l.notify(topicName, niy)
		return v, true, err

	} else {
		// 设置锁失败发起订阅
		return l.subscribe(topicName)
	}
}
