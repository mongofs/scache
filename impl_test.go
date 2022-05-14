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
	"testing"
	"time"
)

func TestCacheImpl_Set(t *testing.T) {
	Convey("test set value ", t, func() {
		var k1, k2, k3 string = "ky1", "ky2", "ky3"
		var v1, v2, v3 defaultStringValue = "hah", "ddd", "ccc"
		ca := New(50, 10*time.Second, nil)
		ca.Set(k1, v1)
		ca.Set(k2, v2)
		ca.Set(k3, v3)

		Convey("test for get key from cache ", func() {
			expect := []string{"hah", "ddd", "ccc"}
			give := []string{"ky1", "ky2", "ky3"}
			for k, v := range give {
				res, err := ca.Get(v)
				if err !=nil  {
					t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", v, expect[k])
				}
				So(string(res.(defaultStringValue)), ShouldEqual, expect[k])
			}
		})

		Convey("test for set value repeat ", func() {
			ca.Set(k1, defaultStringValue("cccc"))
			res, err := ca.Get(k1)
			if err !=nil  {
				t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", k1, "cccc")
			}
			So(string(res.(defaultStringValue)), ShouldEqual, "cccc")
		})

		Convey("test for set nil value ", func() {
			ca.Set("k5", nil)
			ca.Set("", nil)
			_, ok := ca.Get("k5")
			_, ok1 := ca.Get("")
			So(ok, ShouldEqual, false)
			So(ok1, ShouldEqual, false)
		})

		Convey("test for set a bigger than capacity value ", func() {
			err := ca.Set("haha", defaultStringValue("i am a bigger data witch is bigger than maxBytes setting"))
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
		ca.Set("key1", defaultStringValue("im good man"))
		ca.Set("key2", defaultStringValue("im good man1"))
		ca.Set("key3", defaultStringValue("im good man2"))
		Convey("test get key witch existed ", func() {
			res, err := ca.Get("key1")
			if err !=nil  {
				t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", "key1", "im good man")
			}
			So(string(res.(defaultStringValue)), ShouldEqual, "im good man")
		})
		Convey("test get a not exist key  ", func() {
			_, ok := ca.Get("key4")
			So(ok, ShouldEqual, false)
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
		ca.Set("key1", defaultStringValue("im good man"))
		ca.Set("key2", defaultStringValue("im good man1"))
		ca.Set("key3", defaultStringValue("im good man2"))
		Convey("test del key1  ", func() {
			ca.Del("key1")
			So(keys, ShouldBeEmpty)
			_, ok := ca.Get("key1")
			So(ok, ShouldBeFalse)
		})
	})
}

func TestCacheImpl_Expire(t *testing.T) {
	Convey("test expire key from cache ", t, func() {
		cache := New(200*1204*1024, 10*time.Second, func(key string, value Value) {
			fmt.Println("delete the ", key, value)
		})
		cache.Set("userA", defaultStringValue("你好"))
		cache.Expire("userA", 5)
		v, ok := cache.Get("userA")
		So(ok, ShouldBeTrue)
		So(cache.nBytes,ShouldEqual,v.Len()+len("userA"))
		time.Sleep(6 * time.Second)
		_, ok1 := cache.Get("userA")
		So(ok1, ShouldBeFalse)
		So(cache.nBytes, ShouldEqual, 0)
	})
}

