package places

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidName                = errors.New("name is required")
	ErrNameTooLong                = fmt.Errorf("name must be %d characters or fewer", MaxNameLength)
	ErrPlaceNotFound              = errors.New("place not found")
	ErrSubDistrictNotFound        = errors.New("sub-district not found")
	ErrCategoryNotFound           = errors.New("category not found")
	ErrInvalidAttributes          = errors.New("invalid attributes")
	ErrInvalidLocation            = errors.New("location must be within Indonesia")
	ErrAddressTooLong             = fmt.Errorf("address must be %d characters or fewer", MaxAddressLength)
	ErrOperatingHoursTooLong      = fmt.Errorf("operating_hours must be %d characters or fewer", MaxOperatingHoursLength)
	ErrEstimatedPriceRangeTooLong = fmt.Errorf("estimated_price_range must be %d characters or fewer", MaxEstimatedPriceRangeLength)
)

// IsValidationError reports whether err means the place's data was refused.
// An unknown sub-district or category is not included: reads answer those with 404.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidName) ||
		errors.Is(err, ErrNameTooLong) ||
		errors.Is(err, ErrAddressTooLong) ||
		errors.Is(err, ErrOperatingHoursTooLong) ||
		errors.Is(err, ErrEstimatedPriceRangeTooLong) ||
		errors.Is(err, ErrInvalidAttributes) ||
		errors.Is(err, ErrInvalidLocation)
}
