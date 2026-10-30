package place

import "errors"

var (
	ErrInvalidName            = errors.New("name is required")
	ErrPlaceNameTooLong       = errors.New("name must be 150 characters or fewer")
	ErrPlaceNotFound          = errors.New("place not found")
	ErrInvalidCategoryName    = errors.New("name must be 1-100 characters")
	ErrInvalidSlug            = errors.New("slug must be 1-100 lowercase letters or digits, with single hyphens between words")
	ErrSubDistrictNotFound    = errors.New("sub-district not found")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrCategoryInUse          = errors.New("category is referenced by places")
	ErrCategoryExists         = errors.New("category with this name or slug already exists")
	ErrInvalidFieldKey        = errors.New("key must be 1-64 lowercase letters, digits, or underscores, starting with a letter")
	ErrInvalidFieldLabel      = errors.New("label must be 1-100 characters")
	ErrInvalidFieldDataType   = errors.New("data_type must be one of text, number, boolean, select, range, date")
	ErrFieldOptionsRequired   = errors.New("select fields require unique, non-blank options")
	ErrFieldOptionsNotAllowed = errors.New("options are only allowed for select fields")
	ErrInvalidFieldSortOrder  = errors.New("sort_order must be a positive position, or omitted to add the field last")
	ErrCategoryFieldExists    = errors.New("field with this key already exists in the category")
	ErrInvalidAttributes      = errors.New("invalid attributes")
	ErrInvalidLocation        = errors.New("location must be within Indonesia")
)
