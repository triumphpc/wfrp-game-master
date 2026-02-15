package main

import (
	"log"

	"wfrp-bot/config"
)

func main() {
	log.Println("WFRP Game Master Bot")
	log.Println("Starting bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded. Provider: %s, Group ID: %s", cfg.DefaultProvider, cfg.GroupID)
}
