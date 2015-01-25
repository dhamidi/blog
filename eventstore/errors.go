package eventstore

import "errors"

// StorageError records an error and the operation that caused it when
// storing/retrieving events.
type StorageError struct {
	// Op is the operation that caused the error.
	Op string

	// Stream is the id of the event stream operated on.
	Stream string

	// Err is the semantic error, e.g. ErrNotFound or ErrInternal
	Err error

	// InternalErr is any error produced by the underlying storage,
	// e.g. when writing to the file system failed.
	InternalErr error
}

func (err *StorageError) Error() string {
	result := err.Op + " " + err.Stream + ": " + err.Err.Error()

	if err.Err == ErrInternal {
		result = result + ": " + err.InternalErr.Error()
	}

	return result
}

var (
	ErrNotFound = errors.New("not found")
	ErrInternal = errors.New("internal")
)

func IsNotFound(err error) bool {
	if serr, ok := err.(*StorageError); ok {
		return serr.Err == ErrNotFound
	} else {
		return false
	}
}

func IsInternal(err error) bool {
	if serr, ok := err.(*StorageError); ok {
		return serr.Err == ErrInternal
	} else {
		return false
	}
}
