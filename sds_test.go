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
	"unsafe"
)

func TestSds_Calculation(t *testing.T) {
	Convey("test sds size ", t, func() {
		s := &sds{
			key:    "123",                     // 16字节
			expire: 123241,                    // 8字节
			st:     1,                         // 1个字节
			Value:  StringValue("you"), // 16字节
		}
		// key :
		// expire : int64 8字节
		// st : uint8 ,1个字节
		// value :
		fmt.Println(s.Calculation())
		fmt.Println(len(s.key))
		fmt.Println(unsafe.Sizeof(s.expire))
		fmt.Println(unsafe.Sizeof(s.st))
		fmt.Println(len("abc"))
		fmt.Println(unsafe.Sizeof([]string{"1", "2"}))
	})
}
