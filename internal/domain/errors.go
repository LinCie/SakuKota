package domain

import (
	"errors"
	"fmt"
)

// Errors of users and authentication.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailTaken          = errors.New("email already registered")
	ErrUserNameRequired    = errors.New("name is required")
	ErrUserNameTooLong     = fmt.Errorf("name must be %d characters or fewer", MaxUserNameLength)
	ErrEmailInvalid        = errors.New("invalid email")
	ErrEmailTooLong        = fmt.Errorf("email must be %d characters or fewer", MaxEmailLength)
	ErrPasswordTooShort    = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// Errors of places and categories.
var (
	ErrPlaceNotFound                   = errors.New("place not found")
	ErrCategoryNotFound                = errors.New("category not found")
	ErrPlaceNameRequired               = errors.New("name is required")
	ErrPlaceNameTooLong                = fmt.Errorf("name must be %d characters or fewer", MaxPlaceNameLength)
	ErrPlaceAddressTooLong             = fmt.Errorf("address must be %d characters or fewer", MaxPlaceAddressLength)
	ErrPlaceOperatingHoursTooLong      = fmt.Errorf("operating_hours must be %d characters or fewer", MaxPlaceOperatingHoursLength)
	ErrPlaceEstimatedPriceRangeTooLong = fmt.Errorf("estimated_price_range must be %d characters or fewer", MaxPlaceEstimatedPriceRangeLength)
	ErrPlaceAttributesInvalid          = errors.New("invalid attributes")
	ErrPlaceLocationInvalid            = errors.New("location must be within Indonesia")
)

// IsPlaceValidationError reports whether err means the place's data was refused.
// An unknown sub-district or category is not included: reads answer those with 404.
func IsPlaceValidationError(err error) bool {
	return errors.Is(err, ErrPlaceNameRequired) ||
		errors.Is(err, ErrPlaceNameTooLong) ||
		errors.Is(err, ErrPlaceAddressTooLong) ||
		errors.Is(err, ErrPlaceOperatingHoursTooLong) ||
		errors.Is(err, ErrPlaceEstimatedPriceRangeTooLong) ||
		errors.Is(err, ErrPlaceAttributesInvalid) ||
		errors.Is(err, ErrPlaceLocationInvalid)
}

// Errors of the medias of places, sub-districts, and reviews.
var (
	ErrMediaNotFound     = errors.New("media not found")
	ErrMediaLimitReached = fmt.Errorf("at most %d medias are allowed", MaxMedias)
)

// Errors of reviews.
var (
	ErrReviewRatingInvalid  = fmt.Errorf("rating must be between %d and %d", MinRating, MaxRating)
	ErrReviewCommentTooLong = fmt.Errorf("comment must be %d characters or fewer", MaxReviewCommentLength)
	ErrPlaceAlreadyReviewed = errors.New("place already reviewed by this user")
	ErrReviewNotFound       = errors.New("review not found")
)

// Errors of sub-districts.
var (
	ErrSubDistrictNotFound        = errors.New("sub-district not found")
	ErrSubDistrictExists          = errors.New("sub-district with this name or slug already exists")
	ErrSubDistrictInUse           = errors.New("sub-district is referenced by places")
	ErrSubDistrictNameInvalid     = fmt.Errorf("name is required and must be %d characters or fewer", MaxSubDistrictNameLength)
	ErrSubDistrictSlugInvalid     = fmt.Errorf("slug is required and must be %d characters or fewer", MaxSubDistrictNameLength)
	ErrSubDistrictSpendingInvalid = fmt.Errorf("estimated_daily_spending must be between 0 and %.2f", MaxEstimatedDailySpending)
	ErrSubDistrictBoundaryInvalid = errors.New("boundary must lie in Indonesia and must not cross itself")
)

// Errors of suggestions. The place fields of a suggestion follow the rules of
// a place, so they fail with the errors of places.
var (
	ErrSuggestionNotFound          = errors.New("suggestion not found")
	ErrSuggestionLocationInvalid   = errors.New("location is required and must be within Indonesia")
	ErrSuggestionAttributesInvalid = errors.New("attributes must be a JSON object")
	ErrSuggestionNoteTooLong       = fmt.Errorf("note must be %d characters or fewer", MaxSuggestionNoteLength)
	ErrSuggestionReasonTooLong     = fmt.Errorf("reason must be %d characters or fewer", MaxSuggestionReasonLength)
	ErrSuggestionLimitReached      = errors.New("too many pending suggestions")
	ErrSuggestionAlreadyReviewed   = errors.New("suggestion already reviewed")
	ErrSuggestionStatusInvalid     = errors.New("status must be pending, approved, or rejected")
)
