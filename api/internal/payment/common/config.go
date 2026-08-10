package common

import (
	"encoding/json"
	"fmt"
)

// ConfigNormalizer ???????,????????? Normalize()?
type ConfigNormalizer[T any] interface {
	*T
	Normalize()
}

// ParseConfig ??????:JSON marshal/unmarshal + Normalize?
func ParseConfig[T any, PT ConfigNormalizer[T]](raw map[string]interface{}, errConfigInvalid error) (*T, error) {
	if raw == nil {
		return nil, fmt.Errorf("%w: empty config", errConfigInvalid)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal config failed: %v", errConfigInvalid, err)
	}
	var cfg T
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: unmarshal config failed: %v, data: %s", errConfigInvalid, err, string(data))
	}
	PT(&cfg).Normalize()
	return &cfg, nil
}
