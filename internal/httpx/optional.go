package httpx

import "encoding/json"

// Optional is a field of a partial update. Set reports whether the client sent
// the key, and Value is nil when the client sent null.
type Optional[T any] struct {
	Set   bool
	Value *T
}

// UnmarshalJSON marks the field as set. A JSON null leaves Value nil.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
