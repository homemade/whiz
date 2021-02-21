package sqlschema

import (
	"testing"
)

func TestValidate(t *testing.T) {
	scripts := SQLScripts
	err := Validate(scripts)
	if err != nil {
		t.Error(err)
	}
	// add an invalid script
	scripts = append(scripts, SQLScript{ID: "eventz#0"})
	err = Validate(scripts)
	expectedError := "sql schema script eventz#0 contains a duplicate id"
	if err == nil || err.Error() != expectedError {
		t.Errorf("expected valdation to fail with error `%s` not `%v`", expectedError, err)
	}
}
