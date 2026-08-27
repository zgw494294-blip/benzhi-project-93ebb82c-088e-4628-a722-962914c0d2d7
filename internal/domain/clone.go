package domain

import "encoding/json"

func CloneAggregate(in Aggregate) Aggregate {
	b, err := json.Marshal(in)
	if err != nil {
		return Aggregate{}
	}
	var out Aggregate
	if json.Unmarshal(b, &out) != nil {
		return Aggregate{}
	}
	return out
}
