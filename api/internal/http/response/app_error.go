package response

// AppError ??????
type AppError struct {
	Code    int
	Message string
	Err     error `json:"-"`
}

// WrapError ????
func WrapError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
