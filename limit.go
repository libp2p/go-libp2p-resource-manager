package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
)

type Limit interface {
	GetMemoryLimit() int64
	GetStreamLimit(network.Direction) int
	GetConnLimit(network.Direction) int
	GetFDLimit() int
}
