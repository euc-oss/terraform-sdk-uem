package client

import (
	"encoding/json"
	"fmt"
)

// IntOrString holds a value that the UEM API may return as either a JSON number
// or a JSON string depending on whether the field is set. Int value 0 means
// "not set"; named string values (e.g. "LEGACY") represent set states.
//
// This type is emitted by codegen for fields annotated x-flex-enum: true in the
// swagger overlay. It is not a permanent solution — retire each field here when
// upstream UEM swagger declares a consistent wire type.
type IntOrString struct {
	// IsStr is true when the value arrived as a JSON string.
	IsStr  bool
	IntVal int
	StrVal string
}

func (v *IntOrString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("IntOrString: unmarshal string: %w", err)
		}
		v.StrVal = s
		v.IsStr = true
		return nil
	}
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("IntOrString: unmarshal int: %w", err)
	}
	v.IntVal = i
	v.IsStr = false
	return nil
}

func (v IntOrString) MarshalJSON() ([]byte, error) {
	if v.IsStr {
		return json.Marshal(v.StrVal)
	}
	return json.Marshal(v.IntVal)
}

// String returns a human-readable representation.
func (v IntOrString) String() string {
	if v.IsStr {
		return v.StrVal
	}
	return fmt.Sprintf("%d", v.IntVal)
}
