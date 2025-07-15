package main

import "InMemoryDB/cmd"

// Add logs to database package, maybe add more CLI commands, add tests for CLI, metrics endpoint, rate limiting on IP
func main() {
	cmd.Execute()
}
