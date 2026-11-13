package httpx

// Error codes shared by every module and by middleware. They are the "code"
// field of error responses.
const (
	CodeInternalError   = "INTERNAL_ERROR"
	CodeInvalidBody     = "INVALID_BODY"
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"

	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeInvalidRefreshToken = "INVALID_REFRESH_TOKEN"
	CodeEmailTaken          = "EMAIL_TAKEN"

	CodeInvalidPlaceID       = "INVALID_PLACE_ID"
	CodePlaceNotFound        = "PLACE_NOT_FOUND"
	CodeInvalidSubDistrictID = "INVALID_SUB_DISTRICT_ID"
	CodeSubDistrictNotFound  = "SUB_DISTRICT_NOT_FOUND"
	CodeInvalidCategoryID    = "INVALID_CATEGORY_ID"
	CodeCategoryNotFound     = "CATEGORY_NOT_FOUND"
	CodeInvalidLimit         = "INVALID_LIMIT"
	CodeInvalidOffset        = "INVALID_OFFSET"
	CodeInvalidBBox          = "INVALID_BBOX"

	CodeInvalidMediaID    = "INVALID_MEDIA_ID"
	CodeMediaNotFound     = "MEDIA_NOT_FOUND"
	CodeMediaLimitReached = "MEDIA_LIMIT_REACHED"

	CodeSubDistrictExists = "SUB_DISTRICT_EXISTS"
	CodeSubDistrictInUse  = "SUB_DISTRICT_IN_USE"

	CodeAlreadyReviewed = "ALREADY_REVIEWED"

	CodeInvalidSuggestionID       = "INVALID_SUGGESTION_ID"
	CodeSuggestionNotFound        = "SUGGESTION_NOT_FOUND"
	CodeTooManyPendingSuggestions = "TOO_MANY_PENDING_SUGGESTIONS"
	CodeInvalidStatus             = "INVALID_STATUS"
)
