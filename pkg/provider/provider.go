package provider

import (
	"errors"
	"sync"
)

var ErrNotEnabled = errors.New("provider not enabled")

/*
* Provider is an interface that represents backend
 */
type Provider interface {
	Get() (string, error)
	Enable()
	Disable()
	Check() bool
}

type TestProvider struct {
	Id      string
	enabled bool
	mux     sync.RWMutex
}

func NewTestProvider(id string) Provider {
	var mux sync.RWMutex
	return &TestProvider{
		Id:      id,
		mux:     mux,
		enabled: true,
	}
}

// Returns it's ID if it is enabled, otherwise returns error
func (t *TestProvider) Get() (string, error) {
	t.mux.RLock()
	defer t.mux.RUnlock()
	if !t.enabled {
		return "", ErrNotEnabled
	}
	return t.Id, nil
}

func (t *TestProvider) Enable() {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.enabled = true
}

func (t *TestProvider) Disable() {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.enabled = false
}

func (t *TestProvider) Check() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.enabled
}
