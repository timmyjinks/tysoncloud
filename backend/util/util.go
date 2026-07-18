package util

import "iter"

func Unwrap[T comparable](ok T, err error) T {
	if err != nil {
		panic(err)
	}

	return ok
}

func Enumerate[T any](cookieIndex *int, iterator iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for val := range iterator {
			if !yield(*cookieIndex, val) {
				return
			}
			*cookieIndex += 1
		}
	}
}
