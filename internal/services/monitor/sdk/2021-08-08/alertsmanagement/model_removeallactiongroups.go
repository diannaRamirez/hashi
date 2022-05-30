package alertsmanagement

import (
	"encoding/json"
	"fmt"
)

var _ Action = RemoveAllActionGroups{}

type RemoveAllActionGroups struct {

	// Fields inherited from Action
}

var _ json.Marshaler = RemoveAllActionGroups{}

func (s RemoveAllActionGroups) MarshalJSON() ([]byte, error) {
	type wrapper RemoveAllActionGroups
	wrapped := wrapper(s)
	encoded, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshaling RemoveAllActionGroups: %+v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshaling RemoveAllActionGroups: %+v", err)
	}
	decoded["actionType"] = "RemoveAllActionGroups"

	encoded, err = json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling RemoveAllActionGroups: %+v", err)
	}

	return encoded, nil
}
