package utils

import "errors"

var (
	ErrNotInitialized = errors.New("docx: not initialized")
	ErrInvalidConfig  = errors.New("docx: invalid config")
)
