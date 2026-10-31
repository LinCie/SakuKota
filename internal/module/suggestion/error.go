package suggestion

import (
	"errors"
	"fmt"

	"rantau-go/internal/module/place"
)

var (
	ErrInvalidName                = errors.New("name is required")
	ErrNameTooLong                = fmt.Errorf("name must be %d characters or fewer", place.MaxNameLength)
	ErrAddressTooLong             = fmt.Errorf("address must be %d characters or fewer", place.MaxAddressLength)
	ErrOperatingHoursTooLong      = fmt.Errorf("operating_hours must be %d characters or fewer", place.MaxOperatingHoursLength)
	ErrEstimatedPriceRangeTooLong = fmt.Errorf("estimated_price_range must be %d characters or fewer", place.MaxEstimatedPriceRangeLength)
	ErrInvalidLocation            = errors.New("location is required and must be within Indonesia")
	ErrNoteTooLong                = fmt.Errorf("note must be %d characters or fewer", maxNoteLength)
	ErrInvalidAttributes          = errors.New("attributes must be a JSON object")
	ErrReasonTooLong              = fmt.Errorf("reason must be %d characters or fewer", maxReasonLength)
	ErrTooManyPending             = errors.New("too many pending suggestions")
	ErrSubDistrictNotFound        = errors.New("sub-district not found")
	ErrCategoryNotFound           = errors.New("category not found")
	ErrSuggestionNotFound         = errors.New("suggestion not found")
	ErrAlreadyReviewed            = errors.New("suggestion already reviewed")
	ErrInvalidStatus              = errors.New("status must be pending, approved, or rejected")
)
