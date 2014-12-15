package sqltypes

import "testing"

func TestJSON(t *testing.T) {
	j := JSON(`{"foo": 1, "bar": 2}`)
	v, err := j.Value()
	if err != nil {
		t.Errorf("Was not expecting an error")
	}
	err = (&j).Scan(v)
	if err != nil {
		t.Errorf("Was not expecting an error")
	}
	m := map[string]interface{}{}
	j.Unmarshal(&m)

	if m["foo"].(float64) != 1 || m["bar"].(float64) != 2 {
		t.Errorf("Expected valid json but got some garbage instead? %#v", m)
	}

	j = JSON(`{"foo": 1, invalid, false}`)
	v, err = j.Value()
	if err == nil {
		t.Errorf("Was expecting invalid json to fail!")
	}
}
