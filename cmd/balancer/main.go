package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/igor-karpukhin/load-balancer-example/pkg/loadbalancer"
	"github.com/igor-karpukhin/load-balancer-example/pkg/provider"
)

const maxProvidersAllowed = 10

func main() {
	// Application context with an abitility to cancel it
	appContext, cancelFunc := context.WithCancel(context.Background())

	// Create new LoadBalancer with health-check interval 1 sec and 2 successfull health-check passes
	lb := loadbalancer.NewTestLoadBalancer(appContext, maxProvidersAllowed, true, 1*time.Second, 2)

	// Create 10 Providers and add them to the LoadBalancer and to the providers map where key is provider ID
	providers := map[string]provider.Provider{}
	for i := 0; i < 10; i++ {
		ID := fmt.Sprintf("p-%d", i)
		p := provider.NewTestProvider(ID)
		providers[ID] = p
		fmt.Println("Adding provider:", ID)
		err := lb.Register(p)
		if err != nil {
			panic(err)
		}
		fmt.Println("\tprovider added")
	}

	// Try to add 11th Provider, it shouldn't be added
	fmt.Println("Trying to add one more provider")
	if err := lb.Register(provider.NewTestProvider("p-11")); err != nil {
		fmt.Println("\tProvider can't be added. Error:", err.Error())
	}

	// Try to call LoadBalacer 10 times and expect responses from all 10 providers (0-9)
	fmt.Println("Verifying if round robin works by calling Get() 10 times")
	for i := 0; i < 10; i++ {
		resp := lb.Get()
		ID := fmt.Sprintf("p-%d", i)
		if resp != ID {
			fmt.Println("RESP:", resp)
			//panic(fmt.Sprintf("\tIncorrect provider response. Expected: %s. Got: %s\r\n", ID, resp))
		}
		fmt.Printf("\tCorrect provider response. Expected: %s. Got %s\r\n", ID, resp)
	}

	// Unregister providers #2 and #4
	lb.Unregister("p-2")
	lb.Unregister("p-4")

	// Try the round robin again
	fmt.Println("Verifying if round robin works by calling Get() 8 times")
	for _, i := range []int{0, 1, 3, 5, 6, 7, 8, 9} {
		resp := lb.Get()
		ID := fmt.Sprintf("p-%d", i)
		if resp != ID {
			panic(fmt.Sprintf("\tIncorrect provider response. Expected: %s. Got: %s\r\n", ID, resp))
		}
		fmt.Printf("\tCorrect provider response. Expected: %s. Got %s\r\n", ID, resp)
	}

	// Register p-2 and p-2 again and expect them to be added in to the end if the providers list
	lb.Register(providers["p-2"])
	lb.Register(providers["p-4"])

	fmt.Println("Registering back providers 2 and 4")
	fmt.Println("Verifying if round robin works by calling Get() 10 times")
	for _, i := range []int{2, 4, 0, 1, 3, 5, 6, 7, 8, 9} {
		resp := lb.Get()
		ID := fmt.Sprintf("p-%d", i)
		if resp != ID {
			panic(fmt.Sprintf("\tIncorrect provider response. Expected: %s. Got: %s\r\n", ID, resp))
		}
		fmt.Printf("\tCorrect provider response. Expected: %s. Got %s\r\n", ID, resp)
	}

	// Unregister unknown provider
	fmt.Println("Try to unregister unknown provider (p-20)")
	if err := lb.Unregister("p-20"); err != nil {
		fmt.Printf("Unable to unregister provider. Error: %s\r\n", err.Error())
	}

	// Disable P-5 so that health-check exclude this provider from load balancing
	fmt.Println("Disable health check on provider 5")
	providers["p-5"].Disable()
	time.Sleep(2 * time.Second)

	fmt.Println("Verifying if round robin works by calling Get() 9 times")
	for _, i := range []int{2, 4, 0, 1, 3, 6, 7, 8, 9} {
		resp := lb.Get()
		ID := fmt.Sprintf("p-%d", i)
		if resp != ID {
			panic(fmt.Sprintf("\tIncorrect provider response. Expected: %s. Got: %s\r\n", ID, resp))
		}
		fmt.Printf("\tCorrect provider response. Expected: %s. Got %s\r\n", ID, resp)
	}

	// Enable P-5 so that health-check will include this provider to the load banancing
	fmt.Println("Enable P-5")
	providers["p-5"].Enable()
	time.Sleep(4 * time.Second)

	fmt.Println("Verifying if round robin works by calling Get() 10 times")
	for _, i := range []int{2, 4, 0, 1, 3, 5, 6, 7, 8, 9} {
		resp := lb.Get()
		ID := fmt.Sprintf("p-%d", i)
		if resp != ID {
			panic(fmt.Sprintf("\tIncorrect provider response. Expected: %s. Got: %s\r\n", ID, resp))
		}
		fmt.Printf("\tCorrect provider response. Expected: %s. Got %s\r\n", ID, resp)
	}

	fmt.Println("An infinite LoadBalancer requests loop begins. Hit CTRL+C to exit")
	// Handle CTRL+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)

	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("LB response from provider #%s\r\n", lb.Get())
		case <-sigCh:
			cancelFunc()
		}
	}
}
