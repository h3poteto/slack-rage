package event

import (
	"encoding/json"
	"io"
)

type Event map[string]interface{}

func DecodeJSON(r io.Reader) (Event, error) {
	data := make(map[string]interface{})
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (p Event) String(key string) string {
	if v, ok := p[key]; !ok {
		return ""
	} else if vv, ok := v.(string); !ok {
		return ""
	} else {
		return vv
	}
}

func (p Event) Type() string {
	return p.String("type")
}
