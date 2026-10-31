package subdistrict

import "errors"

const (
	CodeInvalidSubDistrictID = "INVALID_SUB_DISTRICT_ID"
	CodeSubDistrictNotFound  = "SUB_DISTRICT_NOT_FOUND"
	CodeSubDistrictExists    = "SUB_DISTRICT_EXISTS"
	CodeSubDistrictInUse     = "SUB_DISTRICT_IN_USE"
)

var (
	ErrInvalidName                   = errors.New("name must be 1-100 characters")
	ErrInvalidSlug                   = errors.New("slug must be 1-100 characters")
	ErrSubDistrictExists             = errors.New("sub-district with this name or slug already exists")
	ErrSubDistrictNotFound           = errors.New("sub-district not found")
	ErrSubDistrictInUse              = errors.New("sub-district is referenced by places")
	ErrInvalidEstimatedDailySpending = errors.New("estimated_daily_spending must be between 0 and 99999999.99")
)
