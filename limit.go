package rcmgr

type Limit interface {
	GetMemoryLimit() int64
	GetStreamLimit() int
	GetConnLimit() int
}
