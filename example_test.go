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

package Scache_test

import (
	"Scache"
	"fmt"
	"time"
)

func ExampleCacheImpl_Register() {
	cache := Scache.New(200 * 1204 * 1024 , 3* time.Second,func(key string, value Scache.Value) {
		fmt.Println("delete the ",key ,value)
	})

	cache.Register("testKey",0, func() (Scache.Value, error) {
		// do select db or some action slow
		time.Sleep(2 *time.Second)

		// store [] byte value
		return Scache.ByteValue("steven is handsome"),nil
	})

	if val ,err := cache.Get("testKey");err !=nil {
		panic(err)
	}else  {
		fmt .Println(string(val.(Scache.ByteValue).Value()))
	}
	// output : "steven is handsome"
}
