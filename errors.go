package main

import (
	"bytes"
	"errors"
	"fmt"
)

type ValidationError map[string][]error

func (verr ValidationError) Error() string {
	out := bytes.NewBufferString("")

	fmt.Fprintf(out, "ValidationError:\n")
	for field, errors := range verr {
		fmt.Fprintf(out, "  %s: %v\n", field, errors)
	}

	return out.String()
}

func (verr ValidationError) Add(key string, err error) ValidationError {
	verr[key] = append(verr[key], err)
	return verr
}

func (verr ValidationError) Len() int {
	return len(verr)
}

func (verr ValidationError) Return() error {
	if verr.Len() == 0 {
		return nil
	} else {
		return verr
	}
}

var (
	ErrNotUnique = errors.New("not unique")
	ErrEmpty     = errors.New("empty")
)
