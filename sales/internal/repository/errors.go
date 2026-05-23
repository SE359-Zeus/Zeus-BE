package repository

import "errors"

var ErrNotFound = errors.New("not found")
var ErrInsufficientInventory = errors.New("insufficient inventory")
