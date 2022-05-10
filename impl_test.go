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
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCache_Set(t *testing.T) {
	Convey("test set value ", t, func() {
		var k1, k2, k3 string = "ky1", "ky2", "ky3"
		var v1, v2, v3 defaultStringValue = "hah", "ddd", "ccc"
		ca := New(12, nil)
		ca.Set(k1,v1)
		ca.Set(k2,v2)
		ca.Set(k3,v3)

		Convey("test for get key from cache ", func() {
			expect := []string {"hah", "ddd", "ccc"}
			give := []string {"ky1", "ky2", "ky3"}
			for k,v := range give {
				res ,ok := ca.Get(v)
				if !ok {
					t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", v, expect[k])
				}
				So(string(res.(defaultStringValue)) ,ShouldEqual,expect[k])
			}
		})

		Convey("test for set value repeat ", func() {
			ca.Set(k1,defaultStringValue("cccc"))
			res ,ok := ca.Get(k1)
			if !ok {
				t.Fatalf("Call Set func failed, give %v , expect %v,but get nil  ", k1, "cccc")
			}
			So(string(res.(defaultStringValue)) ,ShouldEqual,"cccc")
		})

		Convey("test for set nil value ", func() {
			ca.Set("k5",nil)
			ca.Set("",nil)
			_ ,ok := ca.Get("k5")
			_ ,ok1 := ca.Get("")
			So(ok ,ShouldEqual,false)
			So(ok1 ,ShouldEqual,false)
		})
	})
}




