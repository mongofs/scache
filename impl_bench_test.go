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
	"go.uber.org/atomic"
	"testing"
	"time"
)


func BenchmarkCacheImpl_GetKeyWitchRegistered(b *testing.B) {
	ca := New(5000,time.Duration( 10)*time.Second, func(key string, value Value) {
		fmt.Println("删除 key",key,value)
	})
	var testKey = "steven"
	var counter = atomic.Int32{}

	ca.Register(testKey,3, func() (Value, error) {
		counter.Inc()
		time.Sleep(1 * time.Second)
		return defaultStringValue("steven is gooo guy"),nil
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			v,err := ca.Get(testKey)
			if err != nil {
				b.Fatal()
			}
			fmt.Println(v)
		}()
	}
}
