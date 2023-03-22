package config

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Header represents the key-value pairs in an HTTP header.
type Header map[string][]string

func (h *Header) UnmarshalJSON(d []byte) error {
	values := map[string]any{}
	err := json.Unmarshal(d, &values)
	if err != nil {
		return err
	}
	return makeHeader(h, values)
}

func (h *Header) UnmarshalYAML(value *yaml.Node) error {
	values := map[string]any{}
	err := value.Decode(&values)
	if err != nil {
		return err
	}

	return makeHeader(h, values)
}

func makeHeader(h *Header, values map[string]any) error {
	head := *h
	if head == nil {
		head = map[string][]string{}
	}
	for k, v := range values {
		switch val := v.(type) {
		case string:
			head[k] = []string{val}
		case []any:
			strs := make([]string, len(val))
			for i := range val {
				strs[i] = val[i].(string)
			}
			head[k] = strs
		default:
			return fmt.Errorf("unexpected type in header: %T", val)
		}
	}
	*h = head
	return nil
}

var (
	_ json.Unmarshaler = (*Header)(nil)
	_ yaml.Unmarshaler = (*Header)(nil)
)
