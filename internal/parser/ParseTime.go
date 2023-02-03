// Copyright 2023 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTime(layout string, value string) (time.Time, error) {
	if layout == "UNIX" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Now(), fmt.Errorf("failed to parse time: failed to parse value='%s' as int64: %w", value, err)
		}
		return time.Unix(i, 0), nil
	} else if layout == "UNIX_MILLIS" {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Now(), fmt.Errorf("failed to parse time: failed to parse value='%s' as int64: %w", value, err)
		}
		return time.UnixMilli(i), nil
	} else if layout == "UNIX_DECIMAL_NANOS" {
		split := strings.Split(value, ".")
		if len(split) != 2 {
			return time.Now(), fmt.Errorf("failed to parse time: failed to parse value='%s' as UNIX_DECIMAL_NANOS: unexpected length after splitting on '.'. Got length=%v", value, len(split))
		}
		i0, err := strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			return time.Now(), fmt.Errorf("failed to parse time: failed to parse split[0]='%s' as int64: %w", split[0], err)
		}
		i1, err := strconv.ParseInt(split[1], 10, 64)
		if err != nil {
			return time.Now(), fmt.Errorf("failed to parse time: failed to parse split[1]='%s' as int64: %w", split[1], err)
		}
		return time.Unix(i0, i1), nil
	} else {
		return time.Parse(layout, value)
	}
}
