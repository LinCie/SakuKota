package httpx

// Error codes shared by every module and by middleware. They are the "code"
// field of error responses. Codes that belong to one resource are declared by
// the module that responds with them.
const (
	CodeInternalError   = "INTERNAL_ERROR"
	CodeInvalidBody     = "INVALID_BODY"
	CodeValidationError = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
)
