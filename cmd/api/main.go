package main

import (
	"log"
	"os"
)

func main() {
	// Verificar si se debe usar cola o modo directo
	useQueue := os.Getenv("USE_QUEUE")
	
	if useQueue == "true" {
		log.Println("🔄 Running in QUEUE mode (async)")
		mainWithQueue()
	} else {
		log.Println("⚡ Running in DIRECT mode (sync)")
		mainDirect()
	}
}
