// Example Go application using the Billing SDK
//
// This demonstrates how to integrate usage tracking into your application.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Fluxratehq/fluxrate-golang-sdk/billing"
)

func main() {
	fmt.Println("🎯 Billing SDK Example Application")
	fmt.Println("===================================\n")

	// Get configuration from environment
	apiKey := os.Getenv("BILLING_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ BILLING_API_KEY environment variable is required")
		fmt.Println("   Get your API key from: https://app.fluxrate.co/settings/api-keys")
		os.Exit(1)
	}

	meterToken := os.Getenv("BILLING_METER_TOKEN")
	if meterToken == "" {
		fmt.Println("❌ BILLING_METER_TOKEN environment variable is required")
		fmt.Println("   Get your meter token from: https://app.fluxrate.co/meter")
		os.Exit(1)
	}

	// Initialize the SDK
	config := billing.DefaultConfig(apiKey)
	config.Debug = true // Enable debug logging
	config.BatchInterval = 5 * time.Second

	sdk, err := billing.NewSDK(config)
	if err != nil {
		fmt.Printf("💥 Failed to initialize SDK: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if err := simulateAPIUsage(ctx, sdk, meterToken); err != nil {
		fmt.Printf("💥 Fatal error: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down...")
	if err := sdk.Shutdown(ctx); err != nil {
		fmt.Printf("❌ Shutdown error: %v\n", err)
	}
	fmt.Println("✅ Shutdown complete")
}

func simulateAPIUsage(ctx context.Context, sdk *billing.SDK, meterToken string) error {
	fmt.Println("\n🚀 Simulating API usage...\n")

	// Example 1: Track simple API call
	fmt.Println("📊 Example 1: Track API call")
	result, err := sdk.Track(ctx, billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "user_123",
		Quantity:           1,
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Event tracked: %s\n", result.ID)
	}

	// Example 2: Track with metadata
	fmt.Println("\n📊 Example 2: Track with metadata")
	_, err = sdk.Track(ctx, billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "user_456",
		Quantity:           1,
		Metadata: map[string]interface{}{
			"endpoint":    "/api/data",
			"method":      "POST",
			"duration_ms": 123,
		},
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Event with metadata tracked")
	}

	// Example 3: Track with idempotency
	fmt.Println("\n📊 Example 3: Track with idempotency")
	idempotencyKey := fmt.Sprintf("req_%d", time.Now().UnixMilli())

	result1, err := sdk.Track(ctx, billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "user_789",
		Quantity:           1,
		IdempotencyKey:     idempotencyKey,
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Event with idempotency tracked")
	}

	// Try to send the same event again
	result2, err := sdk.Track(ctx, billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "user_789",
		Quantity:           1,
		IdempotencyKey:     idempotencyKey, // Same key
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Duplicate prevented (same event returned)")
		if result1 != nil && result2 != nil {
			fmt.Printf("   Event IDs match: %v\n", result1.ID == result2.ID)
		}
	}

	// Example 4: Immediate tracking (no batching)
	fmt.Println("\n📊 Example 4: Immediate tracking")
	result, err = sdk.TrackImmediate(ctx, billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "user_premium",
		Quantity:           1,
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Event sent immediately: %s\n", result.ID)
	}

	// Wait a moment for batching
	fmt.Println("\n⏳ Waiting for batch to flush...")
	time.Sleep(2 * time.Second)

	// Manual flush
	fmt.Println("\n📊 Example 5: Manual flush")
	flushResult, err := sdk.Flush(ctx)
	if err != nil {
		fmt.Printf("❌ Flush error: %v\n", err)
	} else {
		fmt.Printf("✅ Flushed %d events, %d failed\n", flushResult.Successful, flushResult.Failed)
	}

	return nil
}

