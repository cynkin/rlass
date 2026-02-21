//go:build ignore

package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	target := "http://localhost:8080/check?client_id=race-client"
	if len(os.Args) > 1 {
		target = fmt.Sprintf("http://%s:8080/check?client_id=race-client", os.Args[1])
	}

	// Wait until the next exact 10-second mark on the clock
	// Both machines will fire at the same Unix timestamp
	now := time.Now()
	next := now.Truncate(10 * time.Second).Add(10 * time.Second)
	waitDuration := time.Until(next)

	fmt.Printf("Waiting %v until %v to fire...\n", waitDuration.Round(time.Millisecond), next.Format("15:04:05"))
	time.Sleep(waitDuration)

	// Now fire all requests simultaneously
	totalAllowed := 0
	totalBlocked := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	requests := 200
	fmt.Printf("FIRING %d concurrent requests at %v\n", requests, time.Now().Format("15:04:05.000"))

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(target)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			mu.Lock()
			if resp.StatusCode == http.StatusOK {
				totalAllowed++
			} else {
				totalBlocked++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	fmt.Printf("Allowed: %d | Blocked: %d | Total: %d\n", totalAllowed, totalBlocked, totalAllowed+totalBlocked)
	fmt.Printf("Expected allowed: 10\n")
	if totalAllowed > 10 {
		fmt.Printf("🚨 RACE CONDITION — %d extra requests slipped through\n", totalAllowed-10)
	} else {
		fmt.Printf("✓ Clean this run\n")
	}
}