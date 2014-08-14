package lighthouse

import (
	"bytes"
	"encoding/json"
)

func (this *UserProfile) toJSON() []byte {
	if buff, err := json.Marshal(this); err == nil {
		return buff
	}
	return nil
}

func (this *UserProfile) fromJSON(s []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(s))
	return dec.Decode(this)
}
