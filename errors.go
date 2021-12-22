package rcmgr

import (
	"errors"
)

var (
	ErrResourceLimitExceeded = errors.New("resource limit exceeded")
	ErrResourceScopeClosed   = errors.New("resource scope closed")
)
