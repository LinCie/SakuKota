package place

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
