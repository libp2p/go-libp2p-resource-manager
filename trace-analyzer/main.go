package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
)

type ScopeClass string

const (
	ClassSystem       ScopeClass = "system"
	ClassTransient    ScopeClass = "transient"
	ClassService      ScopeClass = "service"
	ClassServicePeer  ScopeClass = "service-peer"
	ClassProtocol     ScopeClass = "protocol"
	ClassProtocolPeer ScopeClass = "protocol-peer"
	ClassPeer         ScopeClass = "peer"
	ClassConn         ScopeClass = "conn"
	ClassStream       ScopeClass = "stream"
)

func classify(str string) ScopeClass {
	switch {
	case str == "system":
		return ClassSystem
	case str == "transient":
		return ClassTransient
	case strings.HasPrefix(str, "peer:"):
		return ClassPeer
	case strings.HasPrefix(str, "stream-"):
		return ClassStream
	case strings.HasPrefix(str, "conn-"):
		return ClassConn
	case strings.HasPrefix(str, "service:") && strings.Contains(str, "peer:"):
		return ClassServicePeer
	case strings.HasPrefix(str, "service:"):
		return ClassService
	case strings.HasPrefix(str, "protocol:") && strings.Contains(str, "peer:"):
		return ClassProtocolPeer
	case strings.HasPrefix(str, "protocol:"):
		return ClassProtocol
	default:
		panic(fmt.Sprintf("cannot classify scope: %s", str))
	}
}

func extract(str string, prefix string) string {
	val := str[len(prefix):]
	idx := strings.Index(val, ".peer:")
	if idx != -1 {
		val = val[:idx]
	}
	return val
}

func extractService(str string) string {
	return extract(str, "service:")
}

func extractProtocol(str string) string {
	return extract(str, "protocol:")
}

func extractPeer(str string) string {
	const prefix = "peer:"
	idx := strings.Index(str, prefix)
	if idx == -1 {
		panic("prefix not found")
	}
	return str[idx+len(prefix):]
}

type Stat struct {
	StreamsIn  int
	StreamsOut int

	ConnsIn  int
	ConnsOut int
	FD       int

	Memory int64
}

type Evt struct {
	Time     string
	Class    ScopeClass
	Protocol string `json:",omitempty"`
	Service  string `json:",omitempty"`
	Peer     string `json:",omitempty"`
	Scope    string `json:"-"`
	Stat     Stat
}

func main() {
	a := &analyzer{
		current: make(map[string]*Stat),
	}
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s /path/to/rcmgr.json.gz /path/to/events.json\n", os.Args[0])
		os.Exit(1)
	}
	if err := a.Run(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

type analyzer struct {
	current map[string] /*scope*/ *Stat
	conf    rcmgr.DefaultLimitConfig
}

func (a *analyzer) Run(inFile, outFile string) error {
	in, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer in.Close()
	r, err := gzip.NewReader(in)
	if err != nil {
		return err
	}

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()

	w := bufio.NewWriter(out)

	w.Write([]byte("[\n"))
	defer func() {
		w.Write([]byte("\n]"))
		w.Flush()
	}()

	dec := json.NewDecoder(r)
	var wroteFirst bool
	for {
		var evt rcmgr.TraceEvt
		if err := dec.Decode(&evt); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if evt.Scope == "" {
			continue
		}
		ev := a.processEvent(&evt)
		data, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if wroteFirst {
			w.Write([]byte(",\n"))
		}
		w.Write(data)
		wroteFirst = true

		a.maybeUpdateConf(ev)
	}

	conf, err := json.Marshal(a.conf)
	if err != nil {
		return err
	}
	fmt.Println(string(conf))

	return nil
}

func (a *analyzer) processEvent(evt *rcmgr.TraceEvt) Evt {
	name := evt.Scope
	s, ok := a.current[name]
	if !ok {
		s = &Stat{}
		a.current[name] = s
	}

	switch evt.Type {
	case "add_conn", "remove_conn":
		s.FD = evt.FD
		s.ConnsOut = evt.ConnsOut
		s.ConnsIn = evt.ConnsIn
	case "add_stream", "remove_stream":
		s.StreamsIn = evt.StreamsIn
		s.StreamsOut = evt.StreamsOut
	case "reserve_memory", "release_memory":
		s.Memory = evt.Memory
	default:
	}

	ev := Evt{
		Time:  evt.Time,
		Class: classify(name),
		Scope: name,
		Stat:  *s,
	}
	if ev.Class == ClassProtocol || ev.Class == ClassProtocolPeer {
		ev.Protocol = extractProtocol(name)
	}
	if ev.Class == ClassService || ev.Class == ClassServicePeer {
		ev.Service = extractService(name)
	}
	if ev.Class == ClassPeer || ev.Class == ClassProtocolPeer || ev.Class == ClassServicePeer {
		ev.Peer = extractPeer(name)
	}
	return ev
}

func (a *analyzer) maybeUpdateConf(ev Evt) {
	switch ev.Class {
	case ClassSystem:
		a.maybeUpdateBaseLimit(&a.conf.SystemBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.SystemMemory, ev.Stat)
	case ClassTransient:
		a.maybeUpdateBaseLimit(&a.conf.TransientBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.TransientMemory, ev.Stat)
	case ClassService:
		a.maybeUpdateBaseLimit(&a.conf.ServiceBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.ServiceMemory, ev.Stat)
	case ClassServicePeer:
		a.maybeUpdateBaseLimit(&a.conf.ServicePeerBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.ServicePeerMemory, ev.Stat)
	case ClassProtocol:
		a.maybeUpdateBaseLimit(&a.conf.ProtocolBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.ProtocolMemory, ev.Stat)
	case ClassProtocolPeer:
		a.maybeUpdateBaseLimit(&a.conf.ProtocolPeerBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.ProtocolPeerMemory, ev.Stat)
	case ClassPeer:
		a.maybeUpdateBaseLimit(&a.conf.PeerBaseLimit, ev.Stat)
		a.maybeUpdateMemory(&a.conf.PeerMemory, ev.Stat)
	case ClassConn:
		a.maybeUpdateBaseLimit(&a.conf.ConnBaseLimit, ev.Stat)
		if ev.Stat.Memory > a.conf.ConnMemory {
			a.conf.ConnMemory = ev.Stat.Memory
		}
	case ClassStream:
		a.maybeUpdateBaseLimit(&a.conf.StreamBaseLimit, ev.Stat)
		if ev.Stat.Memory > a.conf.StreamMemory {
			a.conf.StreamMemory = ev.Stat.Memory
		}
	}
}

func (a *analyzer) maybeUpdateBaseLimit(l *rcmgr.BaseLimit, stat Stat) {
	if stat.FD > l.FD {
		l.FD = stat.FD
	}
	if stat.StreamsOut > l.StreamsOutbound {
		l.StreamsOutbound = stat.StreamsOut
	}
	if stat.StreamsIn > l.StreamsInbound {
		l.StreamsInbound = stat.StreamsIn
	}
	if stat.ConnsOut > l.ConnsOutbound {
		l.ConnsOutbound = stat.ConnsOut
	}
	if stat.ConnsIn > l.ConnsInbound {
		l.ConnsInbound = stat.ConnsIn
	}
	l.Streams = l.StreamsInbound + l.StreamsOutbound
	l.Conns = l.ConnsInbound + l.ConnsOutbound
}

func (a *analyzer) maybeUpdateMemory(l *rcmgr.MemoryLimit, stat Stat) {
	l.MemoryFraction = 1
	if stat.Memory > l.MaxMemory {
		l.MaxMemory = stat.Memory
	}
}
