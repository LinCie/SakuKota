package subdistricts

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidName                   = fmt.Errorf("name is required and must be %d characters or fewer", maxNameLength)
	ErrInvalidSlug                   = fmt.Errorf("slug is required and must be %d characters or fewer", maxNameLength)
	ErrSubDistrictExists             = errors.New("sub-district with this name or slug already exists")
	ErrSubDistrictNotFound           = errors.New("sub-district not found")
	ErrSubDistrictInUse              = errors.New("sub-district is referenced by places")
	ErrInvalidEstimatedDailySpending = fmt.Errorf("estimated_daily_spending must be between 0 and %.2f", maxEstimatedDailySpending)
	ErrInvalidBoundary               = errors.New("boundary must lie in Indonesia and must not cross itself")
)
