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

var mockerCaller OnCall = func() (Value,error){
	time.Sleep(3 * time.Second)
	return defaultStringValue("steven "),nil
}

func  TestLocker_Get(t *testing.T) {
	Convey("test 10 req  parallel",t, func() {
		ob :=NewLocker(10)
		in := atomic.Int32{}
		for i :=0 ;i<50 ;i ++ {
			go func() {
				value ,err := ob.Get("10", mockerCaller)
				in.Inc()
				fmt.Println(value,err)
			}()
		}
		time.Sleep(5 * time.Second)
		ob.Get("11", mockerCaller)
		fmt.Println(in.Load(),"finish")
	})
}

