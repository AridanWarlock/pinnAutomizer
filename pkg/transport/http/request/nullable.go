package request

import (
	"encoding/json"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type Nullable[T any] struct {
	core.Nullable[T]
}

func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	n.Set = true
	var val T
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}

	n.Value = &val
	return nil
}
