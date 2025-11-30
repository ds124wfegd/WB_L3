package entity

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type CustomTime struct {
	time.Time
}

const customTimeLayout = "2006-01-02T15:04"

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := string(b[1 : len(b)-1]) // Remove quotes
	t, err := time.Parse(customTimeLayout, s)
	if err != nil {
		return err
	}
	ct.Time = t
	return nil
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.Format(customTimeLayout) + `"`), nil
}

func (ct CustomTime) Value() (driver.Value, error) {
	return ct.Time, nil
}

func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		ct.Time = v
	case []byte:
		t, err := time.Parse("2006-01-02 15:04:05", string(v))
		if err != nil {
			return err
		}
		ct.Time = t
	default:
		return fmt.Errorf("cannot scan type %T into CustomTime", value)
	}
	return nil
}
