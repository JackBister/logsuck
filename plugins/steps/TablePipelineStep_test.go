package steps

import "testing"

func TestTablePipelineStep_NoFields_Error(t *testing.T) {
	_, err := compileTableStep("", map[string]string{})
	if err == nil {
		t.Error("Did not get an error when compiling table step without any given fields. This is not allowed because the table pipeline wouldn't do anything if not given any fields.")
	}
}
