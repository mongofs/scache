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
	"testing"
	"time"
)

func TestCacheImpl_Set(t *testing.T) {
	var k1, k2, k3 string = "ky1", "ky2", "ky3"
	var v1, v2, v3  = "hah", "ddd", "ccc"
	ca := New(50, 10*time.Second, nil)
	ca.Set(k1, StringValue(v1))
	ca.Set(k2, StringValue(v2))
	ca.Set(k3, StringValue(v3))
	Convey("test set value ", t, func() {
		Convey("test for get key from cache ", func() {
			expect := []string{"hah", "ddd", "ccc"}
			give := []string{"ky1", "ky2", "ky3"}
			for k, v := range give {
				res, err := ca.Get(v)
				if err != nil {
					t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", v, expect[k])
				}
				So(res.(*DefaultStringValue).Value(), ShouldEqual, expect[k])
			}
		})

		Convey("test for set value repeat ", func() {
			ca.Set(k1, StringValue("cccc"))
			res, err := ca.Get(k1)
			if err != nil {
				t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", k1, "cccc")
			}
			So(res.(*DefaultStringValue).Value(), ShouldEqual, "cccc")
		})

		Convey("test for set nil value ", func() {
			ca.Set("k5", nil)
			ca.Set("", nil)
			_, err := ca.Get("k5")
			if err != nil {
				t.Fatal(err)
			}
			_, err = ca.Get("")
			if err != nil {
				t.Fatal(err)
			}
		})

		Convey("test for set a bigger than capacity value ", func() {
			err := ca.Set("haha", StringValue("i am a bigger data witch is bigger than maxBytes setting"))
			if err == nil {
				t.Fatalf("Call Set func failed, give a big-Value , expect err,but get nil  ")
			}
			So(err, ShouldEqual, ErrValueIsBiggerThanMaxByte)
		})
	})
}

func TestCacheImpl_Get(t *testing.T) {
	Convey("test get key from cache", t, func() {
		ca := New(5000, 10*time.Second, nil)
		ca.Set("key1", StringValue("im good man"))
		ca.Set("key2", StringValue("im good man1"))
		ca.Set("key3", StringValue("im good man2"))
		Convey("test get key witch existed ", func() {
			res, err := ca.Get("key1")
			if err != nil {
				t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", "key1", "im good man")
			}
			So(res.(*DefaultStringValue).Value(), ShouldEqual, "im good man")
		})
		Convey("test get a not exist key  ", func() {
			val, err := ca.Get("key4")
			if err !=nil {
				t.Fatal(err)
			}
			So(val, ShouldEqual, nil)
		})
	})
}

func TestCacheImpl_Del(t *testing.T) {
	Convey("test del key from cache", t, func() {
		var keys []string
		oncaller := func(key string, value Value) {
			keys = append(keys, key)
		}
		ca := New(5000, 10*time.Second, oncaller)
		ca.Set("key1", StringValue("im good man"))
		ca.Set("key2", StringValue("im good man1"))
		ca.Set("key3", StringValue("im good man2"))
		Convey("test del key1  ", func() {
			ca.Del("key1")
			So(keys, ShouldResemble,[]string{"key1"})
			val,err  := ca.Get("key1")
			if err != nil {
				t.Fatal(err)
			}
			So(val, ShouldBeNil)
		})
	})
}

func TestCacheImpl_Expire(t *testing.T) {
	Convey("test expire key from cache ", t, func() {
		cache := New(200*1204*1024, 10*time.Second, func(key string, value Value) {
			fmt.Println("delete the ", key, value)
		})
		cache.Set("userA", StringValue("你好"))
		cache.Expire("userA", 3)
		v, err := cache.Get("userA")
		if err !=nil {
			t.Fatal(err)
		}
		So(v, ShouldEqual,StringValue("你好"))
		time.Sleep(4 * time.Second)
		val, err := cache.Get("userA")
		if err !=nil {
			t.Fatal(err)
		}
		So(val, ShouldBeNil)
	})
}

func TestCacheImpl_Register(t *testing.T) {
	Convey("test expire key from cache ", t, func() {
		var in atomic.Int32
		cache := New(200*1204*1024, 10*time.Second, func(key string, value Value) {
			fmt.Println("delete the ", key, value)
		})
		cache.Register("testKey", 3, func() (Value, error) {
			in.Inc()
			return StringValue("steven"), nil
		})

		Convey("实际请求次数应该小于4次 ", func() {
			// 进行查询
			for i := 0; i < 10; i++ {
				_,err :=cache.Get("testKey")
				if err !=nil {
					t.Fatal(err)
				}
				time.Sleep(1 * time.Second)
			}
			So(in.Load(),ShouldBeLessThanOrEqualTo,4)
		})
	})
}

func TestCacheImpl_RegisterCron(t *testing.T) {
	Convey("测试注册定时器 ", t, func() {
		var in atomic.Int32
		cache := New(200*1204*1024, 10*time.Second, func(key string, value Value) {
			fmt.Println("delete the ", key, value)
		})
		cache.RegisterCron("testKey", 1, func() (Value, error) {
			in.Inc()
			t := time.Now().Unix()
			fmt.Printf("time : %v , 进入定时器\r\n",t)
			return StringValue(fmt.Sprintf("steven %v",t)), nil
		})

		Convey("实际请求次数应该小于4次 ", func() {
			// 进行查询
			for i := 0; i < 1000; i++ {
				time.Sleep(1 * time.Second)
				v ,err :=cache.Get("testKey")
				if err !=nil {
					t.Fatal(err)
				}
				if v == nil {
					fmt.Println(v)
					continue
				}
				fmt.Println(v.(*DefaultStringValue).Value())
			}
			So(in.Load(),ShouldBeLessThanOrEqualTo,4)
		})
	})
}