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
	"time"
)

// sds is a short of simple dynamic string

type SDSStatus uint8

const (
	SDSStatusDelete SDSStatus = iota + 1 // the key is not exist
	SDSStatusNormal
)

var sdsPool = sync.Pool{New: func() interface{} {
	return &sds{
		key:    "",
		expire: 0,
		st:     SDSStatusNormal,
		Value:  nil,
	}
}}

type sds struct {
	key    string // key
	expire int64  // 过期时间

	st    SDSStatus // 当前的key的状态
	Value Value
}

func NewSDS(key string, value Value, expire int) *sds {
	sd := sdsPool.Get().(*sds)
	if expire > 0 {
		sd.expire = time.Now().Unix() + int64(expire)
	}
	sd.key = key
	sd.Value = value
	return sd
}
func (s *sds) Status() SDSStatus {
	return s.st
}

func (s *sds) Delete() {
	s.st = SDSStatusDelete
}

func (s *sds) Destroy() {
	s.key = ""
	s.expire = 0
	s.st = SDSStatusNormal
	s.Value = nil

	sdsPool.Put(s)
}

func (s *sds) ReUse() {
	s.st = SDSStatusNormal
}

//Calculation  这里 计算只计算sds的key值 + value值的大小
func (s *sds) Calculation() int {
	return len(s.key) + s.Value.Len()
}

type Value interface {
	Len() int
}

// ==========================================defaultValue byte========================================

type DefaultByteValue struct {
	cot []byte
}

func ByteValue(cot []byte)  Value{
	return  &DefaultByteValue{cot:cot}
}

func (d *DefaultByteValue) Len() int {
	return len(d.cot)
}

func (d *DefaultByteValue) Value() []byte {
	return d.cot
}

// ==========================================defaultValue String========================================

type DefaultStringValue struct {
	cot string
}

func StringValue(cot string) Value{
	return  &DefaultStringValue{cot:cot}
}

func (d *DefaultStringValue) Len() int {
	return len(d.cot)
}

func (d *DefaultStringValue) Value() string {
	return  d.cot
}
