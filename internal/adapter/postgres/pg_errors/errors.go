package pg_errors

import "errors"

var ErrNotFound = errors.New("not found")
var ErrNilScanValue = errors.New("nil scan value")
var ErrInvalidBatchSize = errors.New("invalid batch size")
var ErrUpdateRowsAffectedCount = errors.New("update rows affected count")
var ErrDeleteRowsAffectedCount = errors.New("delete rows affected count")
