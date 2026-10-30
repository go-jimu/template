package database

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/oops"
	"xorm.io/xorm/convert"
)

type (
	// Timestamp is a type that represents a timestamp in the database
	Timestamp struct {
		Time time.Time
	}

	// NullTimestamp represents a [Timestamp] that may be null.
	NullTimestamp struct {
		Time  time.Time
		Valid bool
	}
)

var (
	_ sql.Scanner        = (*Timestamp)(nil)
	_ driver.Valuer      = (*Timestamp)(nil)
	_ convert.Conversion = (*Timestamp)(nil)
	_ convert.Conversion = (*NullTimestamp)(nil)
	_ sql.Scanner        = (*NullTimestamp)(nil)
	_ driver.Valuer      = (*NullTimestamp)(nil)
)

var (
	ErrUnexpectedType = errors.New("unexpected data type")
	UnixEpoch         = time.UnixMilli(0)
)

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t}
}

func NewNullTimestamp(t time.Time) NullTimestamp {
	if t.IsZero() {
		return NullTimestamp{Valid: false}
	}
	return NullTimestamp{Time: t, Valid: true}
}

func (ts *Timestamp) Scan(src any) error {
	switch src.(type) {
	case int64:
		ts.Time = time.UnixMilli(src.(int64))
		return nil

	case nil:
		return oops.With("source_type", fmt.Sprintf("%T", src)).
			Wrap(ErrUnexpectedType)

	default:
		return oops.With("source_type", fmt.Sprintf("%T", src)).
			Wrap(ErrUnexpectedType)
	}
}

func (ts *Timestamp) Value() (driver.Value, error) {
	return ts.Time.UnixMilli(), nil
}

func (ts *NullTimestamp) Value() (driver.Value, error) {
	if ts.Valid {
		return ts.Time.UnixMilli(), nil
	}
	return nil, nil
}

func (ts *NullTimestamp) Scan(src any) error {
	switch src.(type) {
	case nil:
		ts.Valid = false
		return nil

	case int64:
		ts.Valid = true
		ts.Time = time.UnixMilli(src.(int64))
		return nil

	default:
		return oops.With("source_type", fmt.Sprintf("%T", src)).
			Wrap(ErrUnexpectedType)
	}
}

func (ts *Timestamp) ToDB() ([]byte, error) {
	data, err := json.Marshal(ts.Time.UnixMilli())
	return data, oops.Wrap(err)
}

func (ts *Timestamp) FromDB(data []byte) error {
	if data == nil {
		return oops.With("source", "database").
			Wrap(ErrUnexpectedType)
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return oops.Wrap(err)
	}
	ts.Time = time.UnixMilli(v)
	return nil
}

func (ts *NullTimestamp) ToDB() ([]byte, error) {
	if ts.Valid {
		data, err := json.Marshal(ts.Time.UnixMilli())
		return data, oops.Wrap(err)
	}
	return nil, nil
}

func (ts *NullTimestamp) FromDB(data []byte) error {
	if data == nil {
		ts.Valid = false
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return oops.Wrap(err)
	}
	ts.Time = time.UnixMilli(v)
	ts.Valid = true
	return nil
}
