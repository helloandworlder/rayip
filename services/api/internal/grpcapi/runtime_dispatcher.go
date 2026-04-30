package grpcapi

import (
	"context"
	"errors"
	"sync"
	"time"

	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

type RuntimeDispatcher struct {
	mu       sync.RWMutex
	sessions map[string]*runtimeSession
}

func NewRuntimeDispatcher() *RuntimeDispatcher {
	return &RuntimeDispatcher{sessions: map[string]*runtimeSession{}}
}

func (d *RuntimeDispatcher) Register(nodeID string, stream grpcServerStream) func() {
	session := &runtimeSession{
		nodeID:  nodeID,
		stream:  stream,
		results: map[string]chan runtimelab.ApplyResult{},
	}
	d.mu.Lock()
	d.sessions[nodeID] = session
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		if d.sessions[nodeID] == session {
			delete(d.sessions, nodeID)
		}
		d.mu.Unlock()
		session.close()
	}
}

func (d *RuntimeDispatcher) RegisterHTTP(nodeID string) func() {
	session := &runtimeSession{
		nodeID:  nodeID,
		kind:    sessionKindHTTP,
		pending: make(chan runtimelab.RuntimeApply, 16),
		results: map[string]chan runtimelab.ApplyResult{},
	}
	d.mu.Lock()
	d.sessions[nodeID] = session
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		if d.sessions[nodeID] == session {
			delete(d.sessions, nodeID)
		}
		d.mu.Unlock()
		session.close()
	}
}

func (d *RuntimeDispatcher) DispatchRuntimeApply(ctx context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error) {
	d.mu.RLock()
	session := d.sessions[apply.NodeID]
	d.mu.RUnlock()
	if session == nil {
		return runtimelab.ApplyResult{}, errors.New("node is not connected")
	}
	return session.dispatch(ctx, apply)
}

func (d *RuntimeDispatcher) HandleResult(result runtimelab.ApplyResult) {
	d.mu.RLock()
	sessions := make([]*runtimeSession, 0, len(d.sessions))
	for _, session := range d.sessions {
		sessions = append(sessions, session)
	}
	d.mu.RUnlock()
	for _, session := range sessions {
		if session.handleResult(result) {
			return
		}
	}
}

func (d *RuntimeDispatcher) PollHTTP(ctx context.Context, nodeID string) (runtimelab.RuntimeApply, bool, error) {
	d.mu.RLock()
	session := d.sessions[nodeID]
	d.mu.RUnlock()
	if session == nil || session.kind != sessionKindHTTP {
		return runtimelab.RuntimeApply{}, false, errors.New("http node session is not registered")
	}
	return session.poll(ctx)
}

type grpcServerStream interface {
	Send(*controlv1.ControlEnvelope) error
}

type sessionKind string

const (
	sessionKindGRPC sessionKind = "grpc"
	sessionKindHTTP sessionKind = "http"
)

type runtimeSession struct {
	nodeID  string
	kind    sessionKind
	stream  grpcServerStream
	pending chan runtimelab.RuntimeApply
	sendMu  sync.Mutex
	mu      sync.Mutex
	closed  bool
	results map[string]chan runtimelab.ApplyResult
}

func (s *runtimeSession) dispatch(ctx context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error) {
	ch := make(chan runtimelab.ApplyResult, 1)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return runtimelab.ApplyResult{}, errors.New("node session is closed")
	}
	s.results[apply.ApplyID] = ch
	s.mu.Unlock()

	if s.kind == sessionKindHTTP {
		select {
		case s.pending <- apply:
		case <-ctx.Done():
			s.remove(apply.ApplyID)
			return runtimelab.ApplyResult{}, ctx.Err()
		}
	} else {
		s.sendMu.Lock()
		err := s.stream.Send(&controlv1.ControlEnvelope{
			RequestId: apply.ApplyID,
			Payload: &controlv1.ControlEnvelope_RuntimeApply{
				RuntimeApply: runtimelab.ApplyToProto(apply),
			},
		})
		s.sendMu.Unlock()
		if err != nil {
			s.remove(apply.ApplyID)
			return runtimelab.ApplyResult{}, err
		}
	}

	deadline := 8 * time.Second
	if apply.DeadlineUnixMS > 0 {
		until := time.Until(time.UnixMilli(apply.DeadlineUnixMS))
		if until > 0 {
			deadline = until
		}
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case result, ok := <-ch:
		if !ok {
			return runtimelab.ApplyResult{}, errors.New("node session closed before runtime result")
		}
		return result, nil
	case <-timer.C:
		s.remove(apply.ApplyID)
		return runtimelab.ApplyResult{}, errors.New("runtime apply timed out")
	case <-ctx.Done():
		s.remove(apply.ApplyID)
		return runtimelab.ApplyResult{}, ctx.Err()
	}
}

func (s *runtimeSession) poll(ctx context.Context) (runtimelab.RuntimeApply, bool, error) {
	select {
	case apply, ok := <-s.pending:
		if !ok {
			return runtimelab.RuntimeApply{}, false, errors.New("node session is closed")
		}
		return apply, true, nil
	case <-ctx.Done():
		return runtimelab.RuntimeApply{}, false, ctx.Err()
	}
}

func (s *runtimeSession) handleResult(result runtimelab.ApplyResult) bool {
	s.mu.Lock()
	ch := s.results[result.ApplyID]
	if ch != nil {
		delete(s.results, result.ApplyID)
	}
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	ch <- result
	return true
}

func (s *runtimeSession) remove(commandID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.results, commandID)
}

func (s *runtimeSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.pending != nil {
		close(s.pending)
	}
	for commandID, ch := range s.results {
		delete(s.results, commandID)
		close(ch)
	}
}
