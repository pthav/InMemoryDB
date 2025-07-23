package main

import "InMemoryDB/cmd"

// Rate limiting on IP, add TTL to post and put commands, integration tests, fuzz tests, metric tests, authentication?
func main() {
	cmd.Execute()
}
