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

func NewSDS(key string, value Value) *sds {
	sd := sdsPool.Get().(*sds)
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

func (s *sds) ReUse() {
	s.st = SDSStatusDelete
}

//Calculation  这里 计算只计算sds的key值 + value值的大小
func (s *sds) Calculation() int {
	return len(s.key) + s.Value.Len()
}

type Value interface {
	Len() int
}

// ==========================================defaultValue String========================================
type defaultStringValue string

func (d defaultStringValue) Len() int {
	return len(d)
}

// ==========================================defaultValue String========================================
type defaultIntValue int

func (d defaultIntValue) Len() int {
	return int(d)
}
