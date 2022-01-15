package rcmgr

import (
	"github.com/libp2p/go-libp2p-core/network"
)

type trace struct{}

func (t *trace) Start(limits Limiter) error {
	if t == nil {
		return nil
	}

	// TODO
	return nil
}

func (t *trace) Close() error {
	if t == nil {
		return nil
	}

	// TODO
	return nil
}

func (t *trace) CreateScope(name string, limit Limit) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) DestroyScope(name string) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) ReserveMemory(scope string, prio uint8, size, mem int64) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) BlockReserveMemory(scope string, prio uint8, size, mem int64) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) ReleaseMemory(scope string, size, mem int64) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) AddStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) BlockAddStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) RemoveStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) AddStreams(scope string, rsvpIn, rsvpOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) BlockAddStreams(scope string, rsvpIn, rsvpOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) RemoveStreams(scope string, rsvpIn, rsvpOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) AddConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) BlockAddConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) RemoveConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) AddConns(scope string, rsvpIn, rsvpOut, rsvpFD, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) BlockAddConns(scope string, rsvpIn, rsvpOut, rsvpFD, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}

func (t *trace) RemoveConns(scope string, rsvpIn, rsvpOut, rsvpFD, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	// TODO
}
