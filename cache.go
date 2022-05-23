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

type Cache interface {
	// 获取到一个值，key值，当key不存在的时候返回为空，所以在get一个不存在的值的时候除了判断
	// err !=nil  && value !=nil ,此时才算真正获取到值
	Get(key string) (value Value, err error)

	// 设置一个值到缓存当中，这里值得注意的是，如果设置一个值，它的优先级是高于register 的方法
	Set(key string, value Value) error

	// 防止覆盖，建议内存中优先使用这个函数，防止key值之间的覆盖
	SetNX(key string, value Value) error

	// 覆盖已经存在的某个key，如果key不存在，就返回，否则就覆盖
	SetEX(key string, value Value) error

	// 设置一个值，并为这个值设置一个过期时间
	SetWithTTL(key string, content Value, ttl int) error

	// 删除一个key值，并未真正删除，只是将key值进行标注为不可访问，get获取是获取不到的，但是
	// 会回调删除方法，真正删除的时候是在内存占用超过80% 的时候
	Del(key string)

	// 过期某个值
	Expire(key string, ttl int)

	// 提前将规则注册到cache中，regulation 之间是不能覆盖，否则就会报错
	Register(regulation string, expire int, f /* slow way func */ func() (Value, error))

	// 注册一个CornJob
	RegisterCron(regulation string,flushInterval int ,f /* slow way func */ func() (Value, error))
}
