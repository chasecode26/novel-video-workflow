package providers

type ProviderError struct {
	Category ProviderErrorCategory
	Message  string
	Cause    error
}

func NewProviderError(category ProviderErrorCategory, message string, cause error) ProviderError {
	return ProviderError{
		Category: category,
		Message:  message,
		Cause:    cause,
	}
}

func (e ProviderError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e ProviderError) Unwrap() error {
	return e.Cause
}
