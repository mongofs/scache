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

import "sync"

// regulation 是用于管理注册用户的狗子函数，本想起名字为hook，但是感觉regulation比较不错
type regular struct {
	call   func() (Value, error)
	expire int
}

type RegularManger interface {
	Register(regulation string, expire int, call func() (Value, error)) error

	// 第二个参数理解起来有点困难，因为RegularManager 是并发安全的，所以同一时间打入Get方法
	// 的流量势必会非常多，假设100个流量都进入了Get，在上层调用的时候不应该将所有的请求拿到的
	// 值都去存储起来，那么在这么多流量选择一个的时候，就可以选择slowPath 返回的路径作为存储
	Get(regulation string) (Value, bool/* is slow way */, int /* expire time */, error)
}

type defaultRegularManger struct {
	rw           *sync.RWMutex
	set          map[string]*regular
	singleFlight SingleFlight
}

func NewRegularManager() RegularManger {
	return &defaultRegularManger{
		rw:           &sync.RWMutex{},
		set:          map[string]*regular{},
		singleFlight: NewSingleFlight(20),
	}
}

func (r *defaultRegularManger) Register(regulation string, expire int, call func() (Value, error)) error {
	r.rw.Lock()
	defer r.rw.Unlock()
	if _, ok := r.set[regulation]; ok {
		return ErrRegulationAlreadyExist
	}
	r.set[regulation] = &regular{
		call:   call,
		expire: expire,
	}
	return nil
}

func (r *defaultRegularManger) Get(regulation string) (Value, bool,int , error) {
	r.rw.RLock()
	v, ok := r.set[regulation]
	r.rw.RUnlock()
	if ok {
		val ,slow ,err :=  r.singleFlight.Get(regulation, v.call)
		if err != nil {
			return nil, false, 0, err
		}
		return val,slow,v.expire,nil
	}
	return nil, false, 0, nil
}
