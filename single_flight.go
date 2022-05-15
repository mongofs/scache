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
	"fmt"
	"sync"
	"time"
)

var (
	ErrTopicNotExist     = errors.New("sCache : topic not exist ")
	ErrSlowCallIsTimeOut = errors.New("sCache : slow call timeout")
)

type Element struct {
	value Value
	err   error
}

type Topic struct {
	// 保护notify的用户切片集合
	rw sync.RWMutex

	// notify 是订阅了这个主题的所有请求的集合
	notify []chan *Element

	// OnCall 观察者慢函数，所有订阅对象其实就是为了获取到这个方法的返回
	OnCall func() (Value, error)
}

func (o *Topic) publish(value *Element) {
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

func (o *Topic) subscribe(notify chan *Element) {
	o.rw.Lock()
	o.notify = append(o.notify, notify)
	o.rw.Unlock()
}

type SingleFlight interface {
	Get(regulation string, slowWay func() (Value, error)) (Value, bool, error)
}

// 这里是一个分布式锁的快捷实现 ， short for distributed lock
type defaultSingleFlight struct {
	rw *sync.RWMutex

	// 这里是一个观察者，用户可以订阅，内部调用push 方法,用户请求到内部大概率会被
	// 阻塞到observer 当中，最好使
	obs map[string]*Topic

	// 超时时间，如果存在update方法特别慢，超过了expireTime的最大等待时间，那么
	// 就会返回超时错误
	MaxWaitTime int
}

func NewSingleFlight(maxWaitTime int) SingleFlight {
	l := &defaultSingleFlight{
		rw:          &sync.RWMutex{},
		obs:         map[string]*Topic{},
		MaxWaitTime: maxWaitTime,
	}
	return l
}

func (l *defaultSingleFlight) notify(topicName string, no *Element) {
	l.rw.Lock()
	defer l.rw.Unlock()
	// 将这步操作从最后放在第一排
	v, ok := l.obs[topicName]
	if ok {
		v.publish(no)
	}
	delete(l.obs, topicName)
}

func (l *defaultSingleFlight) setNx(topicName string, slow func() (Value, error)) (chan *Element, error) {
	l.rw.Lock()
	defer l.rw.Unlock()
	if val, ok := l.obs[topicName]; ok {
		ch := make(chan *Element, 1)
		val.subscribe(ch)
		return ch, nil
	} else {
		// 将内容注册到topic管理中心中
		ob := &Topic{
			notify: []chan *Element{},
			OnCall: slow,
		}
		l.obs[topicName] = ob
		return nil, nil
	}
}

func (l *defaultSingleFlight) slowWay(topicName string, slow func() (Value, error)) (Value, bool, error) {
	v, err := l.call(slow)
	if err != nil {
		return nil, false, err
	}

	niy := &Element{
		value: v,
		err:   err,
	}
	l.notify(topicName, niy)
	return v, true, err
}

func (l *defaultSingleFlight) call(slow func() (Value, error)) (Value, error) {
	t := time.NewTicker(time.Duration(l.MaxWaitTime) * time.Second)
	ch := make(chan *Element)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				// todo 记录好err
				fmt.Println(err)
			}
		}()
		// 2 .执行慢操作,此操作不占用锁
		val, err := slow()
		niy := &Element{
			value: val,
			err:   err,
		}
		ch <- niy
	}()
	select {
	case <-t.C:
		return nil, ErrSlowCallIsTimeOut
	case data := <-ch:
		return data.value, data.err
	}
}

func (l *defaultSingleFlight) Get(topicName string, slow func() (Value, error)) (Value, bool /* is slow path*/, error) {
	if topicName == "" || slow == nil {
		return nil, false, ErrInValidParam
	}

	// 对标志位进行加锁操作，标志位设置成功，就进入slowWay 如果标志位设置失败就进入
	// waitWay, 这里需要注意临界值的判断，所以对资源颗粒度相对较小，要求比较高
	ch, err := l.setNx(topicName, slow)
	if err != nil {
		return nil, false, err
	}
	if ch != nil {
		data := <-ch
		return data.value, false, data.err
	}
	return l.slowWay(topicName, slow)
}
