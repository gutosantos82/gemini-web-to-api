package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gemini-web-to-api/internal/config"
	"gemini-web-to-api/internal/providers/gemini"
	"go.uber.org/zap"
)

func main() {
	cfg := &config.Config{}
	cfg.Gemini.Secure1PSID = os.Getenv("GEMINI_1PSID")
	cfg.Gemini.Secure1PSIDTS = os.Getenv("GEMINI_1PSIDTS")
	cfg.Gemini.Secure1PSIDCC = os.Getenv("GEMINI_1PSIDCC")

	logger, _ := zap.NewDevelopment()
	client := gemini.NewClient(cfg, logger)

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}

	// YOU NEED A VALID CONVERSATION ID HERE FROM A RECENT DEEP RESEARCH
	// Example: conversationID := "c_..."
	conversationID := os.Getenv("TEST_CONVERSATION_ID")
	if conversationID == "" {
		log.Println("Skipping manual retrieval test: TEST_CONVERSATION_ID not set")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resp, err := client.RetrieveDeepResearch(ctx, conversationID)
	if err != nil {
		log.Fatalf("Failed to retrieve research: %v", err)
	}

	fmt.Printf("Research Title/Summary:\n%s\n\n", resp.Text)
	fmt.Printf("References Found: %d\n", len(resp.References))
	for i, ref := range resp.References {
		fmt.Printf("[%d] %s\n    URL: %s\n", i+1, ref.Title, ref.URL)
	}
}
