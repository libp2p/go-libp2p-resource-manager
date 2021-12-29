package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
)

var (
	ErrResourceLimitExceeded = network.ErrResourceLimitExceeded
	ErrResourceScopeClosed   = network.ErrResourceScopeClosed
)
