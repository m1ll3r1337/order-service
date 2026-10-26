package broker

import "errors"

type notCriticalError struct {
	err error
}

func (e *notCriticalError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}

	return ""
}

func (e *notCriticalError) Unwrap() error {
	return e.err
}

func NotCriticalError(err error) error {
	if err == nil {
		return nil
	}

	return &notCriticalError{
		err: err,
	}
}

func IsNotCriticalError(err error) bool {
	var ncErr *notCriticalError
	return errors.As(err, &ncErr)
}
