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
	"math/rand"
	"testing"
	"time"
)

var benchCall = func() (Value, error) {
	fmt.Println("走到这里来了")
	time.Sleep(1 * time.Second)
	return StringValue("steven"), nil
}

// goos: darwin
// goarch: amd64
// pkg: Scache
// cpu: Intel(R) Core(TM) i5-8500B CPU @ 3.00GHz
// BenchmarkDefaultSingleFlight_Get
// BenchmarkDefaultSingleFlight_Get-6   	  503554	      2070 ns/op
func BenchmarkDefaultSingleFlight_Get(b *testing.B) {
	ob := NewSingleFlight(10)
	in := atomic.Int32{}
	for i := 0; i < b.N; i++ {
		// 模拟100个请求，这些请求开始时间不一定，在10秒内的均匀分布，理想状况是每秒10个请求
		// 后台请求耗时在1秒左右，那么此时预计慢请求应该在10个以下，总请求数不变，
		go func() {
			in.Inc()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
			_, _, err := ob.Get("10", benchCall)
			if err != nil {
				b.Fatal(err)
				return
			}
		}()
	}
}
