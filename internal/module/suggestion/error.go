package suggestion

import "errors"

const (
	CodeInvalidSuggestionID       = "INVALID_SUGGESTION_ID"
	CodeSuggestionNotFound        = "SUGGESTION_NOT_FOUND"
	CodeAlreadyReviewed           = "ALREADY_REVIEWED"
	CodeTooManyPendingSuggestions = "TOO_MANY_PENDING_SUGGESTIONS"
	CodeInvalidStatus             = "INVALID_STATUS"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrNameTooLong         = errors.New("name must be 150 characters or fewer")
	ErrInvalidLocation     = errors.New("location is required and must be within Indonesia")
	ErrNoteTooLong         = errors.New("note must be 1000 characters or fewer")
	ErrInvalidAttributes   = errors.New("attributes must be a JSON object")
	ErrReasonTooLong       = errors.New("reason must be 1000 characters or fewer")
	ErrTooManyPending      = errors.New("too many pending suggestions")
	ErrSubDistrictNotFound = errors.New("sub-district not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrSuggestionNotFound  = errors.New("suggestion not found")
	ErrAlreadyReviewed     = errors.New("suggestion already reviewed")
	ErrInvalidStatus       = errors.New("status must be pending, approved, or rejected")
)
