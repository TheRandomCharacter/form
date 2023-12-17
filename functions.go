package form

import (
	"errors"
	"reflect"
	"strconv"
	"time"
)

var ErrEmptyInput = errors.New("empty input provided")

type SliceDecodeFunc func([]string) (any, error)
type DecodeFunc func(string) (any, error)

func predefinedDecodeFuncs() map[reflect.Type]DecodeFunc {
	return map[reflect.Type]DecodeFunc{
		reflect.TypeOf(int(42)):     DecodeFuncInt,
		reflect.TypeOf(string("")):  DecodeFuncString,
		reflect.TypeOf(bool(false)): DecodeFuncBool,
		reflect.TypeOf(time.Now()):  DecodeFuncTime,
	}
}

func DecodeFuncInt(value string) (any, error) {
	return CheckEmpty(value, strconv.Atoi)
}

func DecodeFuncString(value string) (any, error) {
	return CheckEmpty(value, func(v string) (string, error) { return v, nil })
}

func DecodeFuncBool(value string) (any, error) {
	return CheckEmpty(value, strconv.ParseBool)
}

func DecodeFuncTime(value string) (any, error) {
	return CheckEmpty(value, func(v string) (time.Time, error) {
		return time.Parse(time.RFC3339, v)
	})
}

func CheckEmpty[T any](value string, f func(string) (T, error)) (t T, e error) {
	if len(value) != 0 {
		return f(value)
	} else {
		e = ErrEmptyInput
		return
	}
}
