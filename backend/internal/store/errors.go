package store

import "errors"

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrNotDispatchable = errors.New("order cannot be dispatched")
)
