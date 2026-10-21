package httpx

import "strconv"

// ParseID parses s as a positive int64 identifier.
func ParseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}
