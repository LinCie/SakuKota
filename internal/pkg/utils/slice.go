// Package utils holds small generic helpers shared across modules.
package utils

// Map applies f to each element of in. The result is never nil, so it encodes as [].
func Map[T, R any](in []T, f func(T) R) []R {
	out := make([]R, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}

	return out
}
