package place

import "errors"

var (
	ErrInvalidName         = errors.New("name is required")
	ErrPlaceNameTooLong    = errors.New("name must be 150 characters or fewer")
	ErrPlaceNotFound       = errors.New("place not found")
	ErrSubDistrictNotFound = errors.New("sub-district not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrInvalidAttributes   = errors.New("invalid attributes")
	ErrInvalidLocation     = errors.New("location must be within Indonesia")
)
