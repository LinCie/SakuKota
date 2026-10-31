package place

import "errors"

const (
	CodeInvalidPlaceID       = "INVALID_PLACE_ID"
	CodePlaceNotFound        = "PLACE_NOT_FOUND"
	CodeInvalidSubDistrictID = "INVALID_SUB_DISTRICT_ID"
	CodeSubDistrictNotFound  = "SUB_DISTRICT_NOT_FOUND"
	CodeInvalidCategoryID    = "INVALID_CATEGORY_ID"
	CodeCategoryNotFound     = "CATEGORY_NOT_FOUND"
	CodeInvalidLimit         = "INVALID_LIMIT"
	CodeInvalidBBox          = "INVALID_BBOX"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrPlaceNameTooLong    = errors.New("name must be 150 characters or fewer")
	ErrPlaceNotFound       = errors.New("place not found")
	ErrSubDistrictNotFound = errors.New("sub-district not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrInvalidAttributes   = errors.New("invalid attributes")
	ErrInvalidLocation     = errors.New("location must be within Indonesia")
)
