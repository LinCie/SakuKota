package suggestions

import (
	"errors"
	"fmt"
)

// The place fields of a suggestion follow the rules of a place, so they fail
// with the errors of package places. These are the suggestion's own.
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
