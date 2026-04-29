package runtime

import "sync"

type Endpoint struct {
	mu       sync.RWMutex
	grpcAddr string
}

func NewEndpoint(initialGRPCAddr string) *Endpoint {
	return &Endpoint{grpcAddr: initialGRPCAddr}
}

func (e *Endpoint) SetGRPCAddr(addr string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.grpcAddr = addr
}

func (e *Endpoint) GRPCAddr() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.grpcAddr
}
