package storage

import "errors"

var ErrObjectNotFound = errors.New("object not found")
var ErrBadObject = errors.New("bad object")
var InternalStorageError = errors.New("internal storage error")
