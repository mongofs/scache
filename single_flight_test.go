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
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/atomic"
	"math/rand"
	"testing"
	"time"
)

var mockerCaller = func() (Value, error) {
	incr.Inc()
	time.Sleep(2 * time.Second)
	// select * from list
	return defaultStringValue("steven "), nil
}

var incr atomic.Int32

func TestDefaultSingleFlight_Get(t *testing.T) {
	Convey("测试是否存在流量漏洞", t, func() {
		ob := NewSingleFlight(10)
		Convey("测试多个并发请求，是否会存在穿透", func() {
			var parallel = 5
			in := atomic.Int32{}
			for i := 0; i < parallel; i++ {
				go func() {
					_, _, err := ob.Get("10", mockerCaller)
					if err != nil {
						t.Fatal(err)
					}
					in.Inc()
				}()
			}
			time.Sleep(3 * time.Second)
			So(in.Load(), ShouldEqual, parallel)
			So(incr.Load(), ShouldEqual, 1)
		})
	})
}

var qukCaller = func() (Value, error) {
	incr.Inc()
	fmt.Println("进入了注册函数找中")
	defer func() {
		fmt.Println("出去了注册函数找中")
	}()
	time.Sleep(1 * time.Second)
	return defaultStringValue("steven "), nil
}

func TestDefaultSingleFlight_GetWithManyReq(t *testing.T) {
	Convey("测试是否存在流量漏洞", t, func() {
		ob := NewSingleFlight(10)
		var parallel = 1000
		in := atomic.Int32{}
		for i := 0; i < parallel; i++ {
			// 模拟100个请求，这些请求开始时间不一定，在10秒内的均匀分布，理想状况是每秒10个请求
			// 后台请求耗时在1秒左右，那么此时预计慢请求应该在10个以下，总请求数不变，
			go func() {
				in.Inc()
				time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
				fmt.Println("调用Caller")
				_, _, err := ob.Get("10", qukCaller)
				if err != nil {
					t.Fatal(err)
				}
			}()
		}
		time.Sleep(11 * time.Second)
		So(in.Load(), ShouldEqual, parallel)
		So(incr.Load(), ShouldBeLessThanOrEqualTo, 10)
	})
}

var timeOutCaller = func() (Value, error) {
	incr.Inc()
	time.Sleep(3 * time.Second)
	// select * from list
	return defaultStringValue("steven "), nil
}

func TestDefaultSingleFlight_GetTimeOut(t *testing.T) {
	Convey("测试超时是否正确的错误处理", t, func() {
		ob := NewSingleFlight(2)
		var parallel = 5
		in := atomic.Int32{}
		for i := 0; i < parallel; i++ {
			go func() {
				in.Inc()
				_, _, err := ob.Get("10", timeOutCaller)
				if err != ErrSlowCallIsTimeOut {
					t.Fatal("出现错误", err)
				}
			}()
		}
		time.Sleep(3 * time.Second)
		So(incr.Load(), ShouldEqual, 1)
		So(in.Load(), ShouldEqual, parallel)
	})
}
