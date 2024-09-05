package util

import "errors"

func Find[A any](s []A, f func(A) bool) (A, error) {
	var r A
	for _, v := range s {
		if f(v) {
			return v, nil
		}
	}
	return r, errors.New("no match in slice")
}

func Filter[A any](s []A, f func(A) bool) []A {
	filtered := []A{}
	for _, i := range s {
		if f(i) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

func InSlice[C comparable](n C, h []C) bool {
	for _, i := range h {
		if i == n {
			return true
		}
	}
	return false
}

func InSliceFunc[C comparable](n C, h []C, f func(i, p C) bool) bool {
	for _, item := range h {
		if f(item, n) {
			return true
		}
	}
	return false
}
