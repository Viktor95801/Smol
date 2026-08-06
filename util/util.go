package util

import "reflect"

func IsAlpha(c uint8) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_')
}

func IsDigit(c uint8) bool {
	return (c >= '0' && c <= '9')
}

func IsAlnum(c uint8) bool {
	return IsAlpha(c) || IsDigit(c)
}

func IsSpace(c uint8) bool {
	return (c == ' ') || (c == '\t') || (c == '\n') || (c == '\r') || (c == '\v') || (c == '\f')
}

func Is[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

func IsEqual[T any](v1 any, v2 T) bool {
	return reflect.DeepEqual(v1, v2)
}

func IsOneOf(v any, targets ...any) bool {
	for _, target := range targets {
		return reflect.DeepEqual(v, target)
	}
	return false
}

func As[T any](v any) T {
	return v.(T)
}
