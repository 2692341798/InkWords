package contracts

import "errors"

// ErrSeriesNotFound marks a missing series export target across blog-related
// handlers and adapters without forcing package-level cycles.
var ErrSeriesNotFound = errors.New("series not found")

