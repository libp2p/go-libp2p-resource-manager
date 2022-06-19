package rcmgr

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

type trace struct {
	path string

	ctx    context.Context
	cancel func()
	closed chan struct{}

	mx   sync.Mutex
	done bool
	pend []interface{}
}

func WithTrace(path string) Option {
	return func(r *resourceManager) error {
		r.trace = &trace{path: path}
		return nil
	}
}

const (
	traceStartEvt              = "start"
	traceCreateScopeEvt        = "create_scope"
	traceDestroyScopeEvt       = "destroy_scope"
	traceReserveMemoryEvt      = "reserve_memory"
	traceBlockReserveMemoryEvt = "block_reserve_memory"
	traceReleaseMemoryEvt      = "release_memory"
	traceAddStreamEvt          = "add_stream"
	traceBlockAddStreamEvt     = "block_add_stream"
	traceRemoveStreamEvt       = "remove_stream"
	traceAddConnEvt            = "add_conn"
	traceBlockAddConnEvt       = "block_add_conn"
	traceRemoveConnEvt         = "remove_conn"
)

type scopeClass struct {
	name string
}

func (s scopeClass) MarshalJSON() ([]byte, error) {
	name := s.name
	var span string
	if idx := strings.Index(name, "span:"); idx > -1 {
		name = name[:idx-1]
		span = name[idx+5:]
	}
	// System and Transient scope
	if name == "system" || name == "transient" {
		return json.Marshal(struct {
			Class string
			Span  string `json:",omitempty"`
		}{
			Class: name,
			Span:  span,
		})
	}
	// Connection scope
	if strings.HasPrefix(name, "conn-") {
		return json.Marshal(struct {
			Class string
			Conn  string
			Span  string `json:",omitempty"`
		}{
			Class: "conn",
			Conn:  name[5:],
			Span:  span,
		})
	}
	// Stream scope
	if strings.HasPrefix(name, "stream-") {
		return json.Marshal(struct {
			Class  string
			Stream string
			Span   string `json:",omitempty"`
		}{
			Class:  "stream",
			Stream: name[7:],
			Span:   span,
		})
	}
	// Peer scope
	if strings.HasPrefix(name, "peer:") {
		return json.Marshal(struct {
			Class string
			Peer  string
			Span  string `json:",omitempty"`
		}{
			Class: "peer",
			Peer:  name[5:],
			Span:  span,
		})
	}

	if strings.HasPrefix(name, "service:") {
		if idx := strings.Index(name, "peer:"); idx > 0 { // Peer-Service scope
			return json.Marshal(struct {
				Class   string
				Service string
				Peer    string
				Span    string `json:",omitempty"`
			}{
				Class:   "service-peer",
				Service: name[8 : idx-1],
				Peer:    name[idx+5:],
				Span:    span,
			})
		} else { // Service scope
			return json.Marshal(struct {
				Class   string
				Service string
				Span    string `json:",omitempty"`
			}{
				Class:   "service",
				Service: name[8:],
				Span:    span,
			})
		}
	}

	if strings.HasPrefix(name, "protocol:") {
		if idx := strings.Index(name, "peer:"); idx > -1 { // Peer-Protocol scope
			return json.Marshal(struct {
				Class    string
				Protocol string
				Peer     string
				Span     string `json:",omitempty"`
			}{
				Class:    "protocol-peer",
				Protocol: name[9 : idx-1],
				Peer:     name[idx+5:],
				Span:     span,
			})
		} else { // Protocol scope
			return json.Marshal(struct {
				Class    string
				Protocol string
				Span     string `json:",omitempty"`
			}{
				Class:    "protocol",
				Protocol: name[9:],
				Span:     span,
			})
		}
	}

	return nil, fmt.Errorf("unrecognized scope: %s", name)
}

type TraceEvt struct {
	Time string
	Type string

	Scope *scopeClass `json:",omitempty"`
	Name  string      `json:",omitempty"`

	Limit interface{} `json:",omitempty"`

	Priority uint8 `json:",omitempty"`

	Delta    int64 `json:",omitempty"`
	DeltaIn  int   `json:",omitempty"`
	DeltaOut int   `json:",omitempty"`

	Memory int64 `json:",omitempty"`

	StreamsIn  int `json:",omitempty"`
	StreamsOut int `json:",omitempty"`

	ConnsIn  int `json:",omitempty"`
	ConnsOut int `json:",omitempty"`

	FD int `json:",omitempty"`
}

func (t *trace) push(evt TraceEvt) {
	t.mx.Lock()
	defer t.mx.Unlock()

	if t.done {
		return
	}
	evt.Time = time.Now().Format(time.RFC3339Nano)
	if evt.Name != "" {
		evt.Scope = &scopeClass{name: evt.Name}
	}

	t.pend = append(t.pend, evt)
}

func (t *trace) background(out io.WriteCloser) {
	defer close(t.closed)
	defer out.Close()

	gzOut := gzip.NewWriter(out)
	defer gzOut.Close()

	jsonOut := json.NewEncoder(gzOut)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var pend []interface{}

	getEvents := func() {
		t.mx.Lock()
		tmp := t.pend
		t.pend = pend[:0]
		pend = tmp
		t.mx.Unlock()
	}

	for {
		select {
		case <-ticker.C:
			getEvents()

			if len(pend) == 0 {
				continue
			}

			if err := t.writeEvents(pend, jsonOut); err != nil {
				log.Warnf("error writing rcmgr trace: %s", err)
				t.mx.Lock()
				t.done = true
				t.mx.Unlock()
				return
			}

			if err := gzOut.Flush(); err != nil {
				log.Warnf("error flushing rcmgr trace: %s", err)
				t.mx.Lock()
				t.done = true
				t.mx.Unlock()
				return
			}

		case <-t.ctx.Done():
			getEvents()

			if len(pend) == 0 {
				return
			}

			if err := t.writeEvents(pend, jsonOut); err != nil {
				log.Warnf("error writing rcmgr trace: %s", err)
				return
			}

			if err := gzOut.Flush(); err != nil {
				log.Warnf("error flushing rcmgr trace: %s", err)
			}

			return
		}
	}
}

func (t *trace) writeEvents(pend []interface{}, jout *json.Encoder) error {
	for _, e := range pend {
		if err := jout.Encode(e); err != nil {
			return err
		}
	}

	return nil
}

func (t *trace) Start(limits Limiter) error {
	if t == nil {
		return nil
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.closed = make(chan struct{})

	out, err := os.OpenFile(t.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil
	}

	go t.background(out)

	t.push(TraceEvt{
		Type:  traceStartEvt,
		Limit: limits,
	})

	return nil
}

func (t *trace) Close() error {
	if t == nil {
		return nil
	}

	t.mx.Lock()

	if t.done {
		t.mx.Unlock()
		return nil
	}

	t.cancel()
	t.done = true
	t.mx.Unlock()

	<-t.closed
	return nil
}

func (t *trace) CreateScope(scope string, limit Limit) {
	if t == nil {
		return
	}

	t.push(TraceEvt{
		Type:  traceCreateScopeEvt,
		Name:  scope,
		Limit: limit,
	})
}

func (t *trace) DestroyScope(scope string) {
	if t == nil {
		return
	}

	t.push(TraceEvt{
		Type: traceDestroyScopeEvt,
		Name: scope,
	})
}

func (t *trace) ReserveMemory(scope string, prio uint8, size, mem int64) {
	if t == nil {
		return
	}

	if size == 0 {
		return
	}

	t.push(TraceEvt{
		Type:     traceReserveMemoryEvt,
		Name:     scope,
		Priority: prio,
		Delta:    size,
		Memory:   mem,
	})
}

func (t *trace) BlockReserveMemory(scope string, prio uint8, size, mem int64) {
	if t == nil {
		return
	}

	if size == 0 {
		return
	}

	t.push(TraceEvt{
		Type:     traceBlockReserveMemoryEvt,
		Name:     scope,
		Priority: prio,
		Delta:    size,
		Memory:   mem,
	})
}

func (t *trace) ReleaseMemory(scope string, size, mem int64) {
	if t == nil {
		return
	}

	if size == 0 {
		return
	}

	t.push(TraceEvt{
		Type:   traceReleaseMemoryEvt,
		Name:   scope,
		Delta:  size,
		Memory: mem,
	})
}

func (t *trace) AddStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut int
	if dir == network.DirInbound {
		deltaIn = 1
	} else {
		deltaOut = 1
	}

	t.push(TraceEvt{
		Type:       traceAddStreamEvt,
		Name:       scope,
		DeltaIn:    deltaIn,
		DeltaOut:   deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) BlockAddStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut int
	if dir == network.DirInbound {
		deltaIn = 1
	} else {
		deltaOut = 1
	}

	t.push(TraceEvt{
		Type:       traceBlockAddStreamEvt,
		Name:       scope,
		DeltaIn:    deltaIn,
		DeltaOut:   deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) RemoveStream(scope string, dir network.Direction, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut int
	if dir == network.DirInbound {
		deltaIn = -1
	} else {
		deltaOut = -1
	}

	t.push(TraceEvt{
		Type:       traceRemoveStreamEvt,
		Name:       scope,
		DeltaIn:    deltaIn,
		DeltaOut:   deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) AddStreams(scope string, deltaIn, deltaOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 {
		return
	}

	t.push(TraceEvt{
		Type:       traceAddStreamEvt,
		Name:       scope,
		DeltaIn:    deltaIn,
		DeltaOut:   deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) BlockAddStreams(scope string, deltaIn, deltaOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 {
		return
	}

	t.push(TraceEvt{
		Type:       traceBlockAddStreamEvt,
		Name:       scope,
		DeltaIn:    deltaIn,
		DeltaOut:   deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) RemoveStreams(scope string, deltaIn, deltaOut, nstreamsIn, nstreamsOut int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 {
		return
	}

	t.push(TraceEvt{
		Type:       traceRemoveStreamEvt,
		Name:       scope,
		DeltaIn:    -deltaIn,
		DeltaOut:   -deltaOut,
		StreamsIn:  nstreamsIn,
		StreamsOut: nstreamsOut,
	})
}

func (t *trace) AddConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut, deltafd int
	if dir == network.DirInbound {
		deltaIn = 1
	} else {
		deltaOut = 1
	}
	if usefd {
		deltafd = 1
	}

	t.push(TraceEvt{
		Type:     traceAddConnEvt,
		Name:     scope,
		DeltaIn:  deltaIn,
		DeltaOut: deltaOut,
		Delta:    int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}

func (t *trace) BlockAddConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut, deltafd int
	if dir == network.DirInbound {
		deltaIn = 1
	} else {
		deltaOut = 1
	}
	if usefd {
		deltafd = 1
	}

	t.push(TraceEvt{
		Type:     traceBlockAddConnEvt,
		Name:     scope,
		DeltaIn:  deltaIn,
		DeltaOut: deltaOut,
		Delta:    int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}

func (t *trace) RemoveConn(scope string, dir network.Direction, usefd bool, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	var deltaIn, deltaOut, deltafd int
	if dir == network.DirInbound {
		deltaIn = -1
	} else {
		deltaOut = -1
	}
	if usefd {
		deltafd = -1
	}

	t.push(TraceEvt{
		Type:     traceRemoveConnEvt,
		Name:     scope,
		DeltaIn:  deltaIn,
		DeltaOut: deltaOut,
		Delta:    int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}

func (t *trace) AddConns(scope string, deltaIn, deltaOut, deltafd, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 && deltafd == 0 {
		return
	}

	t.push(TraceEvt{
		Type:     traceAddConnEvt,
		Name:     scope,
		DeltaIn:  deltaIn,
		DeltaOut: deltaOut,
		Delta:    int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}

func (t *trace) BlockAddConns(scope string, deltaIn, deltaOut, deltafd, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 && deltafd == 0 {
		return
	}

	t.push(TraceEvt{
		Type:     traceBlockAddConnEvt,
		Name:     scope,
		DeltaIn:  deltaIn,
		DeltaOut: deltaOut,
		Delta:    int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}

func (t *trace) RemoveConns(scope string, deltaIn, deltaOut, deltafd, nconnsIn, nconnsOut, nfd int) {
	if t == nil {
		return
	}

	if deltaIn == 0 && deltaOut == 0 && deltafd == 0 {
		return
	}

	t.push(TraceEvt{
		Type:     traceRemoveConnEvt,
		Name:     scope,
		DeltaIn:  -deltaIn,
		DeltaOut: -deltaOut,
		Delta:    -int64(deltafd),
		ConnsIn:  nconnsIn,
		ConnsOut: nconnsOut,
		FD:       nfd,
	})
}
