package tests

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Fluxratehq/fluxrate-golang-sdk/billing"
)

// NOTE: These benchmarks use REAL HTTP requests to the Fluxrate API
// Set BILLING_API_KEY and BILLING_METER_TOKEN environment variables to run them
// Without credentials, benchmarks will be skipped

func getTestCredentials(b *testing.B) (string, string) {
	apiKey := os.Getenv("BILLING_API_KEY")
	meterToken := os.Getenv("BILLING_METER_TOKEN")

	if apiKey == "" || meterToken == "" {
		b.Skip("Skipping benchmark: set BILLING_API_KEY and BILLING_METER_TOKEN to run real-world benchmarks")
	}

	return apiKey, meterToken
}

// BenchmarkTrackImmediate tests the performance of immediate (non-batched) tracking
func BenchmarkTrackImmediate(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: false,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()
	params := billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "bench_user_immediate",
		Quantity:           1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sdk.TrackImmediate(ctx, params)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkTrackWithBatching tests performance with batching enabled
func BenchmarkTrackWithBatching(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: true,
		BatchSize:      100,
		BatchInterval:  1 * time.Second,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()
	params := billing.TrackEventParams{
		MeterToken:         meterToken,
		CustomerExternalID: "bench_user_batching",
		Quantity:           1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sdk.Track(ctx, params)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkTrackWithMetadata tests performance when including metadata
func BenchmarkTrackWithMetadata(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: false,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()
	metadata := map[string]interface{}{
		"endpoint":    "/api/data",
		"method":      "POST",
		"duration_ms": 123,
		"status":      200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sdk.Track(ctx, billing.TrackEventParams{
			MeterToken:         meterToken,
			CustomerExternalID: "bench_user_metadata",
			Quantity:           1,
			Metadata:           metadata,
		})
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkTrackConcurrent tests concurrent tracking performance
func BenchmarkTrackConcurrent(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: false,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, err := sdk.Track(ctx, billing.TrackEventParams{
				MeterToken:         meterToken,
				CustomerExternalID: fmt.Sprintf("bench_user_%d", i),
				Quantity:           1,
			})
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkTrackConcurrentWithBatching tests concurrent tracking with batching
func BenchmarkTrackConcurrentWithBatching(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: true,
		BatchSize:      100,
		BatchInterval:  100 * time.Millisecond,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, err := sdk.Track(ctx, billing.TrackEventParams{
				MeterToken:         meterToken,
				CustomerExternalID: fmt.Sprintf("bench_user_%d", i),
				Quantity:           1,
			})
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkCustomerFiltering tests performance impact of customer filtering
func BenchmarkCustomerFiltering(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	allowedCustomers := make([]string, 100)
	for i := 0; i < 100; i++ {
		allowedCustomers[i] = fmt.Sprintf("bench_user_%d", i)
	}

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:           apiKey,
		EnableBatching:   false,
		AllowedCustomers: allowedCustomers,
		Debug:            false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()

	b.Run("AllowedCustomer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sdk.Track(ctx, billing.TrackEventParams{
				MeterToken:         meterToken,
				CustomerExternalID: "bench_user_0", // In allowed list
				Quantity:           1,
			})
		}
	})

	b.Run("DisallowedCustomer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sdk.Track(ctx, billing.TrackEventParams{
				MeterToken:         meterToken,
				CustomerExternalID: "bench_user_999", // Not in allowed list
				Quantity:           1,
			})
		}
	})
}

// BenchmarkFlush tests the performance of manual batch flushing
func BenchmarkFlush(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: true,
		BatchSize:      1000, // Large batch to prevent auto-flush
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()

	// Pre-fill the batch
	for i := 0; i < 10; i++ {
		sdk.Track(ctx, billing.TrackEventParams{
			MeterToken:         meterToken,
			CustomerExternalID: fmt.Sprintf("bench_user_%d", i),
			Quantity:           1,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sdk.Flush(ctx)
	}
}

// BenchmarkSDKInitialization tests the cost of creating new SDK instances
func BenchmarkSDKInitialization(b *testing.B) {
	apiKey, _ := getTestCredentials(b)

	config := billing.Config{
		APIKey:         apiKey,
		EnableBatching: true,
		Debug:          false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sdk, err := billing.NewSDK(config)
		if err != nil {
			b.Fatalf("NewSDK failed: %v", err)
		}
		sdk.Shutdown(context.Background())
	}
}

// BenchmarkHighThroughput simulates a high-throughput scenario
func BenchmarkHighThroughput(b *testing.B) {
	apiKey, meterToken := getTestCredentials(b)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         apiKey,
		EnableBatching: true,
		BatchSize:      100,
		BatchInterval:  10 * time.Millisecond,
		Debug:          false,
	})
	defer sdk.Shutdown(context.Background())

	ctx := context.Background()
	var counter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := atomic.AddInt64(&counter, 1)
			_, err := sdk.Track(ctx, billing.TrackEventParams{
				MeterToken:         meterToken,
				CustomerExternalID: fmt.Sprintf("bench_user_%d", id%1000), // 1000 unique users
				Quantity:           1,
				Metadata: map[string]interface{}{
					"request_id": id,
					"timestamp":  time.Now().Unix(),
				},
			})
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
		}
	})
}
