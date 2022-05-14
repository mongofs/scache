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

type RegulationCall func() (Value, error)

type regular struct {
	call   RegulationCall
	expire int
}


type RegularManger interface {
	Register(regulation string, expire int, call RegulationCall)error
	Get(regulation string) (Value ,bool,error)
}


type defaultRegularManger struct {
	rw     *sync.RWMutex
	set    map[string]*regular
	singleFlight SingleFlight
}

func NewRegularManager ()RegularManger{
	return &defaultRegularManger{
		rw:     &sync.RWMutex{},
		set: map[string]*regular{},
		singleFlight: NewSingleFlight(20),
	}
}

func (r *defaultRegularManger) Register(regulation string, expire int, call RegulationCall)error {
	r.rw.Lock()
	defer r.rw.Unlock()
	if _,ok := r.set[regulation];ok {
		return ErrRegulationAlreadyExist
	}
	r.set[regulation] = &regular{
		call:   call,
		expire: expire,
	}
	return nil
}

func (r *defaultRegularManger) Get(regulation string) (Value,bool, error) {
	r.rw.RLock()
	v, ok := r.set[regulation]
	r.rw.RUnlock()
	if ok {
		return r.singleFlight.Get(regulation,v.call)
	}
	return nil,false,nil
}
