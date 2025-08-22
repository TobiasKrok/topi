package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	log.Println("Topi Builder starting...")

	// TODO: Implement builder logic
	// - Connect to RabbitMQ to receive build jobs
	// - Clone git repository
	// - Execute build based on .topi workflow file
	// - Store artifacts to persistent storage
	// - Report build status back

	fmt.Println("Builder component is not yet implemented")
	os.Exit(0)
}
