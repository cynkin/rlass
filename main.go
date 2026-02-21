package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run . <port> <redis-addr>")
		fmt.Println("Example: go run . 8080 localhost:6379")
		return
	}
	startServer(os.Args[1], os.Args[2])
}