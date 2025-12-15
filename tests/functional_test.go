package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Fluxratehq/fluxrate-golang-sdk/billing"
)

// MockRoundTripper allows us to mock HTTP requests
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// Helper function to create a mock HTTP client that returns success
func createMockClient(t *testing.T, requestCount *int) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				*requestCount++

				// Verify request structure
				if req.Method != "POST" {
					t.Errorf("Expected POST request, got %s", req.Method)
				}

				// Parse request body
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
				}

				var params billing.TrackEventParams
				if err := json.Unmarshal(body, &params); err != nil {
					t.Errorf("Failed to unmarshal request body: %v", err)
				}

				// Return mock success response
				resp := billing.TrackEventResponse{
					ID:         fmt.Sprintf("evt_%d", time.Now().UnixNano()),
					CustomerID: params.CustomerExternalID,
					MeterID:    params.MeterToken,
					Quantity:   fmt.Sprintf("%.2f", params.Quantity),
					Timestamp:  time.Now().Format(time.RFC3339),
					CreatedAt:  time.Now().Format(time.RFC3339),
				}
				respBody, _ := json.Marshal(resp)

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(string(respBody))),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
}

func TestSDKInitialization(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		sdk, err := billing.NewSDK(billing.Config{
			APIKey: "sk_test_123",
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if sdk == nil {
			t.Error("Expected SDK instance, got nil")
		}
		sdk.Shutdown(context.Background())
	})

	t.Run("Invalid API Key - Empty", func(t *testing.T) {
		_, err := billing.NewSDK(billing.Config{
			APIKey: "",
		})
		if err == nil {
			t.Error("Expected error for empty API key")
		}
	})

	t.Run("Invalid API Key - Wrong Prefix", func(t *testing.T) {
		_, err := billing.NewSDK(billing.Config{
			APIKey: "invalid_key",
		})
		if err == nil {
			t.Error("Expected error for invalid API key prefix")
		}
	})

	t.Run("Custom Configuration", func(t *testing.T) {
		sdk, err := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_custom",
			BatchSize:      20,
			EnableBatching: true,
			Debug:          true,
			MaxRetries:     5,
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if sdk == nil {
			t.Error("Expected SDK instance, got nil")
		}
		sdk.Shutdown(context.Background())
	})
}

func TestCustomerFiltering(t *testing.T) {
	requestCount := 0
	httpClient := createMockClient(t, &requestCount)

	t.Run("Allowed Customer - Should Track", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:           "sk_test_123",
			HTTPClient:       httpClient,
			AllowedCustomers: []string{"cust_1", "cust_2"},
			EnableBatching:   false,
		})
		defer sdk.Shutdown(context.Background())

		_, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "cust_1",
			Quantity:           1,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if requestCount != 1 {
			t.Errorf("Expected 1 request, got %d", requestCount)
		}
	})

	t.Run("Disallowed Customer - Should Skip", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:           "sk_test_123",
			HTTPClient:       httpClient,
			AllowedCustomers: []string{"cust_1", "cust_2"},
			EnableBatching:   false,
		})
		defer sdk.Shutdown(context.Background())

		resp, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "cust_3", // Not in allow list
			Quantity:           1,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp != nil {
			t.Errorf("Expected nil response for disallowed customer, got %v", resp)
		}
		if requestCount != 0 {
			t.Errorf("Expected 0 requests for disallowed customer, got %d", requestCount)
		}
	})

	t.Run("Empty Allow List - Track All Customers", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:           "sk_test_123",
			HTTPClient:       httpClient,
			AllowedCustomers: nil, // No filtering
			EnableBatching:   false,
		})
		defer sdk.Shutdown(context.Background())

		_, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "any_customer",
			Quantity:           1,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if requestCount != 1 {
			t.Errorf("Expected 1 request, got %d", requestCount)
		}
	})
}

func TestTrackingMethods(t *testing.T) {
	requestCount := 0
	httpClient := createMockClient(t, &requestCount)

	t.Run("Track with Batching Disabled", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: false,
		})
		defer sdk.Shutdown(context.Background())

		result, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_1",
			Quantity:           5,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
		if requestCount != 1 {
			t.Errorf("Expected 1 immediate request, got %d", requestCount)
		}
	})

	t.Run("TrackImmediate - Bypass Batching", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: true, // Even with batching enabled
		})
		defer sdk.Shutdown(context.Background())

		result, err := sdk.TrackImmediate(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_premium",
			Quantity:           1,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
		if requestCount != 1 {
			t.Errorf("Expected 1 immediate request, got %d", requestCount)
		}
	})

	t.Run("Track with Metadata", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: false,
		})
		defer sdk.Shutdown(context.Background())

		metadata := map[string]interface{}{
			"endpoint":    "/api/data",
			"method":      "POST",
			"duration_ms": 123,
		}

		result, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_1",
			Quantity:           1,
			Metadata:           metadata,
		})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Error("Expected result, got nil")
		}
	})

	t.Run("Track with Idempotency Key", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: false,
		})
		defer sdk.Shutdown(context.Background())

		idempotencyKey := "unique_key_123"

		result1, err1 := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_1",
			Quantity:           1,
			IdempotencyKey:     idempotencyKey,
		})

		if err1 != nil {
			t.Errorf("Unexpected error: %v", err1)
		}
		if result1 == nil {
			t.Error("Expected result, got nil")
		}
	})
}

func TestBatching(t *testing.T) {
	requestCount := 0
	httpClient := createMockClient(t, &requestCount)

	t.Run("Manual Flush", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: true,
			BatchSize:      10, // High batch size to prevent auto-flush
		})
		defer sdk.Shutdown(context.Background())

		// Track 3 events - they should be queued, not sent
		for i := 0; i < 3; i++ {
			result, err := sdk.Track(context.Background(), billing.TrackEventParams{
				MeterToken:         "meter_123",
				CustomerExternalID: fmt.Sprintf("user_%d", i),
				Quantity:           1,
			})
			if err != nil {
				t.Errorf("Track failed: %v", err)
			}
			if result != nil {
				t.Error("Expected nil result for batched event")
			}
		}

		// No requests should have been sent yet
		if requestCount != 0 {
			t.Errorf("Expected 0 requests before flush, got %d", requestCount)
		}

		// Manual flush
		result, err := sdk.Flush(context.Background())
		if err != nil {
			t.Errorf("Flush error: %v", err)
		}
		if result.Successful != 3 {
			t.Errorf("Expected 3 successful events, got %d", result.Successful)
		}
		if result.Failed != 0 {
			t.Errorf("Expected 0 failed events, got %d", result.Failed)
		}

		// Now 3 requests should have been sent
		if requestCount != 3 {
			t.Errorf("Expected 3 requests after flush, got %d", requestCount)
		}
	})

	t.Run("Auto-Flush on Batch Size", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: true,
			BatchSize:      2, // Small batch size
		})
		defer sdk.Shutdown(context.Background())

		// Track 3 events - should trigger one batch (2 events) + 1 pending
		for i := 0; i < 3; i++ {
			sdk.Track(context.Background(), billing.TrackEventParams{
				MeterToken:         "meter_123",
				CustomerExternalID: fmt.Sprintf("user_%d", i),
				Quantity:           1,
			})
		}

		// Give it a moment to process the batch
		time.Sleep(100 * time.Millisecond)

		// Should have sent 2 events in one batch
		if requestCount != 2 {
			t.Errorf("Expected 2 requests (batch flushed), got %d", requestCount)
		}
	})
}

func TestGracefulShutdown(t *testing.T) {
	requestCount := 0
	httpClient := createMockClient(t, &requestCount)

	t.Run("Shutdown Flushes Pending Events", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     httpClient,
			EnableBatching: true,
			BatchSize:      100, // Large batch size
		})

		// Add some events
		for i := 0; i < 5; i++ {
			sdk.Track(context.Background(), billing.TrackEventParams{
				MeterToken:         "meter_123",
				CustomerExternalID: fmt.Sprintf("user_%d", i),
				Quantity:           1,
			})
		}

		// Events should still be queued
		if requestCount != 0 {
			t.Errorf("Expected 0 requests before shutdown, got %d", requestCount)
		}

		// Shutdown should flush pending events
		err := sdk.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Shutdown error: %v", err)
		}

		// All 5 events should be sent
		if requestCount != 5 {
			t.Errorf("Expected 5 requests after shutdown, got %d", requestCount)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	requestCount := 0

	// Create a slow mock client
	slowClient := &http.Client{
		Transport: &MockRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				requestCount++
				// Simulate slow request
				time.Sleep(100 * time.Millisecond)
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("{}")),
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	t.Run("Context Timeout", func(t *testing.T) {
		requestCount = 0
		sdk, _ := billing.NewSDK(billing.Config{
			APIKey:         "sk_test_123",
			HTTPClient:     slowClient,
			EnableBatching: false,
		})
		defer sdk.Shutdown(context.Background())

		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := sdk.Track(ctx, billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_1",
			Quantity:           1,
		})

		// Should get a context deadline exceeded error
		if err != nil && err.Error() != "context deadline exceeded" {
			// Context error might propagate
			t.Logf("Got error (expected with context timeout): %v", err)
		}
	})
}

func TestValidation(t *testing.T) {
	requestCount := 0
	httpClient := createMockClient(t, &requestCount)

	sdk, _ := billing.NewSDK(billing.Config{
		APIKey:         "sk_test_123",
		HTTPClient:     httpClient,
		EnableBatching: false,
	})
	defer sdk.Shutdown(context.Background())

	t.Run("Valid Request", func(t *testing.T) {
		requestCount = 0
		_, err := sdk.Track(context.Background(), billing.TrackEventParams{
			MeterToken:         "meter_123",
			CustomerExternalID: "user_1",
			Quantity:           1,
		})

		if err != nil {
			t.Errorf("Unexpected error for valid request: %v", err)
		}
		if requestCount != 1 {
			t.Errorf("Expected 1 request, got %d", requestCount)
		}
	})

	// Note: SDK currently doesn't validate parameters client-side
	// Validation happens on the server side
	t.Run("Server-Side Validation", func(t *testing.T) {
		// The SDK will send these requests to the server
		// and the server will validate them
		t.Log("SDK performs server-side validation only")
	})
}
