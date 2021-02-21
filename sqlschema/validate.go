package sqlschema

import (
	"fmt"
)

func Validate(scripts []SQLScript) error {
	validScripts := make(map[string]SQLScript)
	for _, s := range scripts {
		// check id is unique
		if _, exists := validScripts[s.ID]; exists {
			return fmt.Errorf("sql schema script %s contains a duplicate id", s.ID)
		}
		validScripts[s.ID] = s
		// TODO further sql validation
	}
	return nil
}
