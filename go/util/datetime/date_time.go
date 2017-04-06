// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package datetime implements marshalling of Go DateTime values into Noms structs
// with type DateTimeType.
package datetime

import (
	"math"
	"time"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

// DateTime is an alias for time.Time that allows us to marshal date time to
// Noms.
type DateTime time.Time

// DateTimeType is the Noms type used to represent date time objects in Noms.
// The field secSinceEpoch may contain fractions in cases where seconds are
// not sufficient.
var DateTimeType = types.MakeStructTypeFromFields("DateTime", types.FieldMap{
	"secSinceEpoch": types.NumberType,
})

// MarshalNoms makes DateTime implement marshal.Marshaler and it makes
// DateTime marshal into a Noms struct with type DateTimeType.
func (dt DateTime) MarshalNoms() (types.Value, error) {
	t := time.Time(dt)
	return types.NewStructWithType(DateTimeType, types.ValueSlice{
		types.Number(float64(t.UnixNano()) * 1e-9),
	}), nil
}

// MarshalNomsType makes DateTime implement marshal.TypeMarshaler and it
// allows marshal.MarshalType to work with DateTime.
func (dt DateTime) MarshalNomsType() (*types.Type, error) {
	return DateTimeType, nil
}

// UnmarshalNoms makes DateTime implement marshal.Unmarshaler and it allows
// Noms struct with type DateTimeType able to be unmarshaled onto a DateTime
// Go struct
func (dt *DateTime) UnmarshalNoms(v types.Value) error {
	strct := struct {
		SecSinceEpoch float64
	}{}
	err := marshal.Unmarshal(v, &strct)
	if err != nil {
		return err
	}

	s, frac := math.Modf(strct.SecSinceEpoch)
	*dt = DateTime(time.Unix(int64(s), int64(frac*1e9)))
	return nil
}

// DateTimeNow returns a DateTime struct representing the current time. This
// struct is created using seconds only. Since nanoseconds aren't used, it's
// value is unchanged when Unmarshalling/Marshalling:
//   dt1 := DateTimeNow()
//   dt2 := DateTime{}
//   marshal.MustUnmarshal(marshal.MustMarshal(dt1), &dt2)
//   dt1 == dt2 // Will be true
func DateTimeNow() DateTime {
	return DateTime(time.Unix(time.Now().Unix(), 0))
}

// IsZero returns true if the dt is a true zero value or if it's a zero value
// that has been marshalled into Noms and unmarshalled again.
func (dt DateTime) IsZero() bool {
	if time.Time(dt).IsZero() {
		return true
	}

	dt1 := DateTime{}
	marshal.MustUnmarshal(marshal.MustMarshal(DateTime{}), &dt1)
	return dt == dt1
}

// String() causes DateTime structs to be printed in the same way as time.Time
// structs.
func (dt DateTime) String() string {
	return time.Time(dt).String()
}
