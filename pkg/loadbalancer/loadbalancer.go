package loadbalancer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/igor-karpukhin/load-balancer-example/pkg/provider"
)

type LoadBalancer interface {
	Register(providers ...provider.Provider) error
	Get() string
	Unregister(providerID string) error
}

// Small wrapper with some helper fields
type ProviderWrapper struct {
	Healthy           bool
	HealthCheckPassed uint32
	Provider          provider.Provider
}

type TestLoadBalancer struct {
	ctx                        context.Context
	maxProvidersAllowed        uint32
	healthInterval             time.Duration
	currentProvider            uint32
	fullness                   uint32
	providers                  []ProviderWrapper
	mtx                        sync.RWMutex
	HealthCheckPassesThreshold uint32
}

func NewTestLoadBalancer(ctx context.Context, MaxProvidersAllowed uint32, enabledHealthCheck bool, healthCheckInterval time.Duration, healthCheckSuccessThreshold uint32) LoadBalancer {
	t := &TestLoadBalancer{
		ctx:                        ctx,
		maxProvidersAllowed:        MaxProvidersAllowed,
		healthInterval:             healthCheckInterval,
		providers:                  []ProviderWrapper{},
		HealthCheckPassesThreshold: healthCheckSuccessThreshold,
		currentProvider:            0,
	}
	if enabledHealthCheck {
		go t.healthCheckRoutine(ctx)
	}
	return t
}

func (tlb *TestLoadBalancer) performHealthCheck(ctx context.Context) {
	tlb.mtx.Lock()
	tlb.mtx.Unlock()
	for i := 0; i < len(tlb.providers); i++ {
		p := &tlb.providers[i]

		if !p.Provider.Check() {
			p.HealthCheckPassed = 0
			p.Healthy = false
			continue
		}
		p.HealthCheckPassed += 1

		if p.HealthCheckPassed >= tlb.HealthCheckPassesThreshold {
			p.Healthy = true
		}
	}
}

func (tlb *TestLoadBalancer) healthCheckRoutine(ctx context.Context) {
	for {
		select {
		case <-time.After(tlb.healthInterval):
			tlb.performHealthCheck(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (tlb *TestLoadBalancer) nextProvider() (provider.Provider, error) {
	tlb.mtx.RLock()
	defer tlb.mtx.RUnlock()

	if tlb.fullness == 0 {
		return nil, fmt.Errorf("no providers added\r\n")
	}

	count := 0
	for count < len(tlb.providers) {
		if tlb.currentProvider >= tlb.fullness {
			tlb.currentProvider = 0
		}
		prv := tlb.providers[tlb.currentProvider]
		if prv.Healthy {
			tlb.currentProvider += 1
			return prv.Provider, nil
		}
		tlb.currentProvider += 1
		count += 1
	}
	return nil, fmt.Errorf("no healthy providers\r\n")
}

func (tlb *TestLoadBalancer) Get() string {
	provider, err := tlb.nextProvider()
	if err != nil {
		return fmt.Sprintf("LoadBanalcer error: %s\r\n", err.Error())
	}

	result, err := provider.Get()
	if err != nil {
		return fmt.Sprintf("Provider error: %s\r\n", err.Error())

	}
	return result
}

func (tlb *TestLoadBalancer) Register(providers ...provider.Provider) error {
	tlb.mtx.Lock()
	defer tlb.mtx.Unlock()

	if uint32(len(tlb.providers)+len(providers)) > tlb.maxProvidersAllowed {
		return fmt.Errorf("LB Registration: providers limit reached. Allowed: %d", len(tlb.providers))
	}
	pw := []ProviderWrapper{}
	for _, p := range providers {
		pw = append(pw, ProviderWrapper{
			Provider: p,
			Healthy:  true,
		})
	}
	tlb.providers = append(tlb.providers, pw...)
	tlb.fullness += uint32(len(providers))
	return nil
}

func (tlb *TestLoadBalancer) Unregister(providerID string) error {
	tlb.mtx.Lock()
	defer tlb.mtx.Unlock()

	for i := 0; i < len(tlb.providers); i++ {
		id, err := tlb.providers[i].Provider.Get()
		if err != nil {
			fmt.Errorf("LB Unregistration: provider error: %v\r\n", err)
		}
		if id == providerID {
			tlb.providers = append(tlb.providers[:i], tlb.providers[i+1:]...)
			tlb.fullness -= 1
			return nil
		}
	}
	return fmt.Errorf("LB Unregistration: provider with ID '%s' not found\r\n", providerID)
}
