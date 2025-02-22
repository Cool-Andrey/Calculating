package main

import "github.com/Cool-Andrey/Calculating/internal/agent/transport"

func main() {
	agent := transport.NewAgent()
	agent.Start()
}
