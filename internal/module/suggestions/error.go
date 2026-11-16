package suggestions

import (
	"errors"
	"fmt"

	"rantau-go/internal/module/places"
)

// The place fields of a suggestion follow the rules of a place, so they fail
// with the place's errors.
var (
	ErrInvalidName                = places.ErrInvalidName
	ErrNameTooLong                = places.ErrNameTooLong
	ErrAddressTooLong             = places.ErrAddressTooLong
	ErrOperatingHoursTooLong      = places.ErrOperatingHoursTooLong
	ErrEstimatedPriceRangeTooLong = places.ErrEstimatedPriceRangeTooLong
	ErrSubDistrictNotFound        = places.ErrSubDistrictNotFound
	ErrCategoryNotFound           = places.ErrCategoryNotFound
	ErrPlaceInvalidLocation       = places.ErrInvalidLocation
	ErrPlaceInvalidAttributes     = places.ErrInvalidAttributes
)

var (
	ErrInvalidLocation    = errors.New("location is required and must be within Indonesia")
	ErrNoteTooLong        = fmt.Errorf("note must be %d characters or fewer", maxNoteLength)
	ErrInvalidAttributes  = errors.New("attributes must be a JSON object")
	ErrReasonTooLong      = fmt.Errorf("reason must be %d characters or fewer", maxReasonLength)
	ErrTooManyPending     = errors.New("too many pending suggestions")
	ErrSuggestionNotFound = errors.New("suggestion not found")
	ErrAlreadyReviewed    = errors.New("suggestion already reviewed")
	ErrInvalidStatus      = errors.New("status must be pending, approved, or rejected")
)
