// Package billing provides the official SDK for tracking usage-based billing
// events.
// Simple, lightweight, and reliable usage tracking for SaaS applications.
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Version is the SDK version.
const Version = "v0.1.0"

// Config holds the SDK configuration options.
type Config struct {
	// APIKey is your organization's API key (starts with sk_live_ or sk_test_)
	APIKey string `json:"api_key"`

	// APIUrl is the API base URL (default: https://api.fluxrate.co/api/v1)
	APIUrl string `json:"api_url"`

	// EnableBatching enables automatic batching of events (default: true)
	EnableBatching bool `json:"enable_batching"`

	// BatchSize is the batch size for automatic batching (default: 10)
	BatchSize int `json:"batch_size"`

	// BatchInterval is the batch interval (default: 5 seconds)
	BatchInterval time.Duration `json:"batch_interval"`

	// EnableRetry enables automatic retry on failure (default: true)
	EnableRetry bool `json:"enable_retry"`

	// MaxRetries is the maximum retry attempts (default: 3)
	MaxRetries int `json:"max_retries"`

	// AllowedCustomers is a list of customer IDs to allow requests for.
	// If empty, all customers are allowed.
	AllowedCustomers []string `json:"allowed_customers"`

	// Debug enables debug logging (default: false)
	Debug bool `json:"debug"`

	// HTTPClient allows customizing the HTTP client (optional)
	HTTPClient *http.Client `json:"-"`
}

// TrackEventParams contains the parameters for tracking a usage event.
type TrackEventParams struct {
	// MeterToken is the unique token of the meter to track
	MeterToken string `json:"meter_token"`

	// CustomerExternalID is your own customer ID (not the internal UUID)
	CustomerExternalID string `json:"customer_external_id"`

	// Quantity is the usage quantity to track
	Quantity float64 `json:"quantity"`

	// Timestamp is an optional timestamp (default: now)
	Timestamp *time.Time `json:"timestamp,omitempty"`

	// IdempotencyKey is an optional key to prevent duplicates
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// Metadata is optional additional data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TrackEventResponse is the response from tracking an event.
type TrackEventResponse struct {
	ID         string                 `json:"id"`
	CustomerID string                 `json:"customer_id"`
	MeterID    string                 `json:"meter_id"`
	Quantity   string                 `json:"quantity"`
	Timestamp  string                 `json:"timestamp"`
	CreatedAt  string                 `json:"created_at"`
	MetaData   map[string]interface{} `json:"meta_data,omitempty"`
}

// BatchResult contains the results of a batch flush.
type BatchResult struct {
	Successful int
	Failed     int
	Errors     []BatchError
}

// BatchError represents an error that occurred while sending an event.
type BatchError struct {
	Event TrackEventParams
	Error error
}

// SDK is the main billing SDK client.
type SDK struct {
	config           Config
	httpClient       *http.Client
	batchQueue       []TrackEventParams
	batchMu          sync.Mutex
	stopChan         chan struct{}
	wg               sync.WaitGroup
	allowedCustomers map[string]bool
}

// NewSDK creates a new billing SDK instance.
func NewSDK(config Config) (*SDK, error) {
	// Validate API key
	if config.APIKey == "" || !strings.HasPrefix(config.APIKey, "sk_") {
		return nil, fmt.Errorf("Invalid API key: must start with 'sk_live_' or 'sk_test_'")
	}

	// Set defaults
	if config.APIUrl == "" {
		config.APIUrl = "https://api.fluxrate.co/api/v1"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.BatchInterval == 0 {
		config.BatchInterval = 5 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 10
	}

	allowedCustomers := make(map[string]bool)
	for _, id := range config.AllowedCustomers {
		allowedCustomers[id] = true
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	sdk := &SDK{
		config:           config,
		httpClient:       httpClient,
		batchQueue:       make([]TrackEventParams, 0),
		stopChan:         make(chan struct{}),
		allowedCustomers: allowedCustomers,
	}

	sdk.log("SDK initialized: version=%s, apiUrl=%s, batching=%v, batchSize=%d",
		Version, config.APIUrl, config.EnableBatching, config.BatchSize)

	// Start batch processing if enabled
	if config.EnableBatching {
		sdk.startBatchTimer()
	}

	return sdk, nil
}

// Track tracks a single usage event.
// If batching is enabled, the event will be queued and sent in a batch.
func (s *SDK) Track(ctx context.Context, params TrackEventParams) (*TrackEventResponse, error) {
	// Check allowed customers
	if len(s.allowedCustomers) > 0 && !s.allowedCustomers[params.CustomerExternalID] {
		s.log("Skipping event for disallowed customer: %s", params.CustomerExternalID)
		return nil, nil
	}

	if s.config.EnableBatching {
		s.batchMu.Lock()
		s.batchQueue = append(s.batchQueue, params)
		queueLen := len(s.batchQueue)
		s.batchMu.Unlock()

		s.log("Event queued for batching (%d/%d)", queueLen, s.config.BatchSize)

		// Flush if batch is full
		if queueLen >= s.config.BatchSize {
			_, err := s.flushBatch(ctx)
			if err != nil {
				s.log("Batch flush error: %v", err)
			}
		}

		// Return nil since event is queued and will be sent in batch
		// Events are sent when batch is full, interval expires, or Flush() is called
		return nil, nil
	}

	return s.TrackImmediate(ctx, params)
}

// TrackImmediate tracks an event immediately without batching.
func (s *SDK) TrackImmediate(ctx context.Context, params TrackEventParams) (*TrackEventResponse, error) {
	return s.sendEventWithRetry(ctx, params)
}

// Flush manually flushes the current batch.
func (s *SDK) Flush(ctx context.Context) (*BatchResult, error) {
	return s.flushBatch(ctx)
}

// Shutdown gracefully shuts down the SDK (flushes pending events).
func (s *SDK) Shutdown(ctx context.Context) error {
	s.log("Shutting down SDK...")

	// Signal stop to batch timer
	close(s.stopChan)

	// Wait for batch timer to stop
	s.wg.Wait()

	// Flush remaining events
	s.batchMu.Lock()
	queueLen := len(s.batchQueue)
	s.batchMu.Unlock()

	if queueLen > 0 {
		_, err := s.flushBatch(ctx)
		if err != nil {
			return fmt.Errorf("Failed to flush remaining events: %w", err)
		}
	}

	s.log("SDK shutdown complete")
	return nil
}

// Private methods

func (s *SDK) log(format string, args ...interface{}) {
	if s.config.Debug {
		log.Printf("[BillingSDK] "+format, args...)
	}
}

func (s *SDK) startBatchTimer() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.config.BatchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.batchMu.Lock()
				queueLen := len(s.batchQueue)
				s.batchMu.Unlock()

				if queueLen > 0 {
					_, err := s.flushBatch(context.Background())
					if err != nil {
						s.log("Batch flush error: %v", err)
					}
				}
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *SDK) flushBatch(ctx context.Context) (*BatchResult, error) {
	s.batchMu.Lock()
	if len(s.batchQueue) == 0 {
		s.batchMu.Unlock()
		return &BatchResult{Successful: 0, Failed: 0, Errors: nil}, nil
	}

	batch := make([]TrackEventParams, len(s.batchQueue))
	copy(batch, s.batchQueue)
	s.batchQueue = s.batchQueue[:0]
	s.batchMu.Unlock()

	s.log("Flushing batch of %d events", len(batch))

	result := &BatchResult{
		Successful: 0,
		Failed:     0,
		Errors:     make([]BatchError, 0),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, event := range batch {
		wg.Add(1)
		go func(e TrackEventParams) {
			defer wg.Done()
			_, err := s.sendEventWithRetry(ctx, e)
			mu.Lock()
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, BatchError{Event: e, Error: err})
			} else {
				result.Successful++
			}
			mu.Unlock()
		}(event)
	}

	wg.Wait()

	s.log("Batch complete: %d successful, %d failed", result.Successful, result.Failed)

	return result, nil
}

func (s *SDK) sendEventWithRetry(ctx context.Context, params TrackEventParams) (*TrackEventResponse, error) {
	var lastErr error
	maxAttempts := 1
	if s.config.EnableRetry {
		maxAttempts = s.config.MaxRetries
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := s.sendEvent(ctx, params)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		s.log("Attempt %d/%d failed: %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			// Exponential backoff
			delay := time.Duration(min(1000*pow(2, attempt-1), 10000)) * time.Millisecond
			s.log("Retrying in %v...", delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, lastErr
}

func (s *SDK) sendEvent(ctx context.Context, params TrackEventParams) (*TrackEventResponse, error) {
	url := s.config.APIUrl + "/sdk/track"

	// Build request body
	body := map[string]interface{}{
		"meter_token":          params.MeterToken,
		"customer_external_id": params.CustomerExternalID,
		"quantity":             params.Quantity,
	}

	if params.Timestamp != nil {
		body["timestamp"] = params.Timestamp.Format(time.RFC3339)
	}
	if params.IdempotencyKey != "" {
		body["idempotency_key"] = params.IdempotencyKey
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	s.log("Sending event: %+v", body)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.config.APIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		json.Unmarshal(respBody, &errResp)
		errMsg := errResp.Detail
		if errMsg == "" {
			errMsg = errResp.Message
		}
		if errMsg == "" {
			errMsg = "Unknown error"
		}
		return nil, fmt.Errorf("Failed to track event: %d %s - %s", resp.StatusCode, resp.Status, errMsg)
	}

	var result TrackEventResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %w", err)
	}

	return &result, nil
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
