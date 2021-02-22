package pipeline

import (
	"reflect"
	"testing"
)

func TestIgnoresPreviousStepsOptimization(t *testing.T) {
	p, err := CompilePipeline("| search \"abc\" | search \"def\"", nil, nil)
	if err != nil {
		t.Fatalf("got error when compiling pipeline: %v", err)
		return
	}
	if len(p.steps) != 1 {
		t.Fatalf("unexpected number of steps in pipeline, expected=%v, got=%v", 1, len(p.steps))
		return
	}
	sps, ok := p.steps[0].(*searchPipelineStep)
	if !ok {
		t.Fatalf("unexpected step type for step 0 in pipeline, expected=searchPipelineStep, got=%v", reflect.TypeOf(p.steps[0]).Name())
		return
	}
	_, ok = sps.srch.Fragments["def"]
	if !ok {
		t.Fatalf("expected step 0 to contain the fragment \"def\", have=%v", sps.srch.Fragments)
		return
	}
}
