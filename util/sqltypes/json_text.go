package sqltypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON is a json.RawMessage, which is a []byte underneath.
// Value() validates the json format in the source, and returns an error if
// the json is not valid.  Scan does no validation.  JSON additionally
// implements `Unmarshal`, which unmarshals the json within to an interface{}
type JSON json.RawMessage

// Returns the *j as the JSON encoding of j.
func (j *JSON) MarshalJSON() ([]byte, error) {
	return *j, nil
}

// UnmarshalJSON sets *j to a copy of data
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSON: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil

}

// Value returns j as a value.  This does a validating unmarshal into another
// RawMessage.  If j is invalid json, it returns an error.
func (j JSON) Value() (driver.Value, error) {
	var m json.RawMessage
	var err = j.Unmarshal(&m)
	if err != nil {
		return []byte{}, err
	}
	return []byte(j), nil
}

// Scan stores the src in *j.  No validation is done.
func (j *JSON) Scan(src interface{}) error {
	var source []byte
	switch src.(type) {
	case string:
		source = []byte(src.(string))
	case []byte:
		source = src.([]byte)
	default:
		return errors.New("JSON: incompatible type")
	}
	*j = JSON(append((*j)[0:0], source...))
	return nil
}

// Unmarshal unmarshal's the json in j to v, as in json.Unmarshal.
func (j *JSON) Unmarshal(v interface{}) error {
	return json.Unmarshal([]byte(*j), v)
}
