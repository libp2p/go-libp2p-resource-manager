package rcmgr

import (
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.TraceReporter instead
type TraceReporter = rcmgr.TraceReporter

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.WithTrace instead
func WithTrace(path string) Option {
	return rcmgr.WithTrace(path)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.WithTraceReporter instead
func WithTraceReporter(reporter TraceReporter) Option {
	return rcmgr.WithTraceReporter(reporter)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.TraceEvtTyp instead
type TraceEvtTyp = rcmgr.TraceEvtTyp

const (
	TraceStartEvt              = rcmgr.TraceStartEvt
	TraceCreateScopeEvt        = rcmgr.TraceCreateScopeEvt
	TraceDestroyScopeEvt       = rcmgr.TraceDestroyScopeEvt
	TraceReserveMemoryEvt      = rcmgr.TraceReserveMemoryEvt
	TraceBlockReserveMemoryEvt = rcmgr.TraceBlockReserveMemoryEvt
	TraceReleaseMemoryEvt      = rcmgr.TraceReleaseMemoryEvt
	TraceAddStreamEvt          = rcmgr.TraceAddStreamEvt
	TraceBlockAddStreamEvt     = rcmgr.TraceBlockAddStreamEvt
	TraceRemoveStreamEvt       = rcmgr.TraceRemoveStreamEvt
	TraceAddConnEvt            = rcmgr.TraceAddConnEvt
	TraceBlockAddConnEvt       = rcmgr.TraceBlockAddConnEvt
	TraceRemoveConnEvt         = rcmgr.TraceRemoveConnEvt
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/host/resource-manager.TraceEvt instead
type TraceEvt = rcmgr.TraceEvt
