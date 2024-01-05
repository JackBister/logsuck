// Copyright 2024 Jack Bister
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

package steps

import "testing"

func TestTablePipelineStep_NoFields_Error(t *testing.T) {
	_, err := compileTableStep("", map[string]string{})
	if err == nil {
		t.Error("Did not get an error when compiling table step without any given fields. This is not allowed because the table pipeline wouldn't do anything if not given any fields.")
	}
}
