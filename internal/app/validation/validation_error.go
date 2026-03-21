package validation

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func NewValidationError(msg string) *ValidationError { return &ValidationError{Msg: msg} }
