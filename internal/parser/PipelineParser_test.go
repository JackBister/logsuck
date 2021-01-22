// Copyright 2020 The Logsuck Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import "testing"

func TestImplicitSearch(t *testing.T) {
	const input = "source=*my-log.txt* hello world"
	res, err := ParsePipeline(input)
	if err != nil {
		t.Fatalf("ImplicitSearch parse returned error: %v", err)
	}
	if len(res.Steps) != 1 {
		t.Fatalf("ImplicitSearch expected 1 step, got %v", len(res.Steps))
	}
	if res.Steps[0].StepType != "search" {
		t.Fatalf("ImplicitSearch expected step 0 to be search, got %v", res.Steps[0].StepType)
	}
	if input != res.Steps[0].Value {
		t.Fatalf("ImplicitSearch expected value='%v', got '%v'", input, res.Steps[0].Value)
	}
}

func TestExplicitSearch(t *testing.T) {
	// TODO: This test should be extended with startTime and endTime when the parser supports options
	const input = "| search \"source=*my-log.txt* hello world\""
	res, err := ParsePipeline(input)
	if err != nil {
		t.Fatalf("ExplicitSearch parse returned error: %v", err)
	}
	if len(res.Steps) != 1 {
		t.Fatalf("ExplicitSearch expected 1 step, got %v", len(res.Steps))
	}
	if res.Steps[0].StepType != "search" {
		t.Fatalf("ExplicitSearch expected step 0 to be search, got %v", res.Steps[0].StepType)
	}
	const expected = "source=*my-log.txt* hello world"
	if expected != res.Steps[0].Value {
		t.Fatalf("ExplicitSearch expected value='%v', got '%v'", expected, res.Steps[0].Value)
	}
}

func TestIncompletePipe_Fails(t *testing.T) {
	const input = "hello world |"
	_, err := ParsePipeline(input)
	if err == nil {
		t.Fatalf("Expected an error but got nil when parsing '%v'", input)
	}

	const input2 = "hello world | rex"
	_, err = ParsePipeline(input2)
	if err == nil {
		t.Fatalf("Expected an error but got nil when parsing '%v'", input2)
	}
}

func TestPipe(t *testing.T) {
	const input = "hello world | rex \"(?P<field>world)\""
	res, err := ParsePipeline(input)
	if err != nil {
		t.Fatalf("TestPipe parse returned error: %v", err)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("TestPipe expected 2 steps, got %v", len(res.Steps))
	}
	step0 := res.Steps[0]
	if step0.StepType != "search" {
		t.Fatalf("TestPipe expected step 0 to be search, got %v", step0.StepType)
	}
	const step0exp = "hello world "
	if step0.Value != step0exp {
		t.Fatalf("TestPipe expected step 0 to have value='%v', got '%v'", step0exp, step0.Value)
	}
	step1 := res.Steps[1]
	if step1.StepType != "rex" {
		t.Fatalf("TestPipe expected step 1 to be rex, got %v", step1.StepType)
	}
	const step1exp = "(?P<field>world)"
	if step1.Value != step1exp {
		t.Fatalf("TestPipe expected step 1 to have value='%v', got '%v'", step1exp, step1.Value)
	}
}

func TestPipeWithOptions(t *testing.T) {
	const input = "hello world | rex field=source \"log-(?P<logid>\\d+).txt\""
	res, err := ParsePipeline(input)
	if err != nil {
		t.Fatalf("TestPipeWithOptions parse returned error: %v", err)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("TestPipeWithOptions expected 2 steps, got %v", len(res.Steps))
	}
	step0 := res.Steps[0]
	if step0.StepType != "search" {
		t.Fatalf("TestPipeWithOptions expected step 0 to be search, got %v", step0.StepType)
	}
	const step0exp = "hello world "
	if step0.Value != step0exp {
		t.Fatalf("TestPipeWithOptions expected step 0 to have value='%v', got '%v'", step0exp, step0.Value)
	}
	step1 := res.Steps[1]
	if step1.StepType != "rex" {
		t.Fatalf("TestPipeWithOptions expected step 1 to be rex, got %v", step1.StepType)
	}
	fieldOption, ok := step1.Args["field"]
	if !ok {
		t.Fatalf("TestPipeWithOptions got unexpected !ok when getting field option")
	}
	if fieldOption != "source" {
		t.Fatalf("TestPipeWithOptions expected step 1 field option to be '%v', got '%v'", "source", fieldOption)
	}
	const step1exp = "log-(?P<logid>\\d+).txt"
	if step1.Value != step1exp {
		t.Fatalf("TestPipeWithOptions expected step 1 to have value='%v', got '%v'", step1exp, step1.Value)
	}
}
