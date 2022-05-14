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
	"fmt"
	"testing"
	"time"
)

// 这里为singleFlight 进行压力测试

func  BenchmarkDefaultSingleFlight_Get(b *testing.B) {
	ca := New(500,30*time.Second, func(key string, value Value) {
		fmt.Println(key,value)
	})
	ca.Register("test",5, func() (Value, error) {
		fmt.Println("进入了慢方法")
		return defaultStringValue("steven"),nil
	})

	for i := 0;i < b.N;i ++ {
		_ ,err := ca.Get("test")
		if err !=nil {
			b.Fatal(err)
		}
	}
}