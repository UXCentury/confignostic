package domain

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Counter is encoded as a canonical decimal string, preserving uint64 precision
// in browser clients. Each owning contract decides whether zero is permitted.
type Counter uint64

var ErrInvalidCounter = errors.New("counter must be a canonical unsigned decimal string")

func (n Counter) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatUint(uint64(n), 10))
}
func (n *Counter) UnmarshalJSON(data []byte) error {
	var value string
	if json.Unmarshal(data, &value) != nil {
		return ErrInvalidCounter
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || strconv.FormatUint(parsed, 10) != value {
		return ErrInvalidCounter
	}
	*n = Counter(parsed)
	return nil
}
