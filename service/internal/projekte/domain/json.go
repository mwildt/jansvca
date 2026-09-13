package domain

import "encoding/json"

func jsonUnmarshal(b []byte, e any) error {
	return json.Unmarshal(b, e)
}
