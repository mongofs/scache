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
	"time"
)

type Cache interface {
	// 获取到一个值，key值，当key不存在的时候返回错误。存在就返回具体内容
	Get(key string) (value Value, ok bool)

	// 设置一个值到缓存当中
	Set(key string, value Value)

	// 删除一个值
	Del(key string) error

	// 设置一个值，并为这个值设置一个过期时间
	SetWithTTL(key string, content []byte, ttl time.Duration) error

	// 获取到targetKey值，如果说这个值存在则从内存中获取，如果从内存中获取不到，则
	// 调用f 闭包方法，并将返回值存入Cache中。这里还存在一个问题，时间问题，如果f 执行时间过长
	// 那么需要一个超时返回，此时就会报错，所有调用这个方法的请求都将收到这个错误返回
	GetTargetKeyLockerWithTimeOut(targetKey string, f func() (Value, error)) (Value, error)
}

type sds struct {
	key    string // key
	expire int64  // 过期时间
 	Value  Value
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

