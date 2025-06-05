package util

import (
	"errors"
)

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

func InSliceFunc[A any](n A, h []A, f func(i, p A) bool) bool {
	for _, item := range h {
		if f(item, n) {
			return true
		}
	}
	return false
}

func SlicePop[T any](s []T, i int) ([]T, T) {
	elem := s[i]
	s = append(s[:i], s[i+1:]...)
	return s, elem
}

func SliceTake[T any](s []T, amount int) (picked, remaining []T) {
	if amount > len(s) {
		amount = len(s)
	}
	res := make([]T, 0, amount)
	remaining = s
	for i := 0; i < amount; i++ {
		var elem T
		remaining, elem = SlicePop(remaining, 0)
		res = append(res, elem)
	}
	return res, remaining
}

func SmallestGroupIndex[T any](groups [][]T) int {
	var res, smallest int
	for i, g := range groups {
		count := len(g)
		if smallest == 0 {
			smallest = count
		}
		if count < smallest {
			smallest = i
			res = i
		}
	}
	return res
}
