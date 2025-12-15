# Fluxrate Golang SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/Fluxratehq/fluxrate-golang-sdk.svg)](https://pkg.go.dev/github.com/Fluxratehq/fluxrate-golang-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/Fluxratehq/fluxrate-golang-sdk)](https://goreportcard.com/report/github.com/Fluxratehq/fluxrate-golang-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Official Go SDK for [Fluxrate](https://fluxrate.co) - Usage-based billing for SaaS.

## Installation

```bash
go get github.com/Fluxratehq/fluxrate-golang-sdk@v0.1.1
```

Or to get the latest version:

```bash
go get github.com/Fluxratehq/fluxrate-golang-sdk@latest
```

## Example

Please check the [examples](examples/README.md) directory for a complete runnable example and usage instructions.

## Configuration

**Configuration example**

The `Config` struct configures the SDK. Only `APIKey` is required; all other fields have sensible defaults.

```go
sdk, err := billing.NewSDK(billing.Config{
    APIKey:           "sk_live_abc123", // Required
    APIUrl:           "https://api.fluxrate.co/api/v1", // Optional, default: https://api.fluxrate.co/api/v1
    EnableBatching:   true, // Optional, default: true
    BatchSize:        200, // Optional, default: 100
    BatchInterval:    5 * time.Second, // Optional, default: 5s
    EnableRetry:      true, // Optional, default: true
    MaxRetries:       20, // Optional, default: 10
    Debug:            true, // Optional, default: false
    AllowedCustomers: []string{"customer_123", "customer_456"}, // Optional, default: [] (track all customers)
    HTTPClient:       nil, // Optional, default: nil (uses default HTTP client)
})
```

**Note on customer filtering**

When `AllowedCustomers` is set to a non-empty list, the SDK will only send tracking requests for customers in that list. For customers not in the list:
- The request is skipped (not sent to the backend)
- `Track()` returns `(nil, nil)`
- A debug message is logged if `Debug` is enabled

This is useful for:
- Testing in production with a subset of customers
- Gradual rollout of usage-based billing
- Excluding certain customer tiers from billing

## Integration

### HTTP Server Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"

    "github.com/Fluxratehq/fluxrate-golang-sdk/billing"
)

var sdk *billing.SDK

func main() {
    // Initialize SDK
    sdk, err := billing.NewSDK(billing.Config{
        APIKey: "YOUR_BILLING_API_KEY",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sdk.Shutdown(context.Background())

    http.HandleFunc("/api/data", handleData)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleData(w http.ResponseWriter, r *http.Request) {
    // Your business logic
    data := getData()

    // Track usage
    userID := r.Header.Get("X-User-ID")
    _, err := sdk.Track(r.Context(), billing.TrackEventParams{
        MeterToken:         "YOUR_BILLING_METER_TOKEN",
        CustomerExternalID: userID,
        Quantity:           1,
    })
    if err != nil {
        log.Printf("Failed to track usage: %v", err)
    }

    json.NewEncoder(w).Encode(data)
}

func getData() map[string]string {
    return map[string]string{"message": "Hello, World!"}
}
```

### Gin Framework Example

```go
package main

import (
    "context"
    "log"

    "github.com/Fluxratehq/fluxrate-golang-sdk/billing"
    "github.com/gin-gonic/gin"
)

var sdk *billing.SDK

func main() {
    // Initialize SDK
    var err error
    sdk, err = billing.NewSDK(billing.Config{
        APIKey: "YOUR_BILLING_API_KEY",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sdk.Shutdown(context.Background())

    r := gin.Default()

    // Middleware to track API usage
    r.Use(func(c *gin.Context) {
        c.Next()

        // Track after request completes
        userID := c.GetHeader("X-User-ID")
        if userID != "" {
            go func() {
                sdk.Track(context.Background(), billing.TrackEventParams{
                    MeterToken:         "YOUR_BILLING_METER_TOKEN",
                    CustomerExternalID: userID,
                    Quantity:           1,
                })
            }()
        }
    })

    r.GET("/api/data", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello, World!"})
    })

    r.Run(":8080")
}
```

## Troubleshooting

- Ensure server is healthy by visting `https://api.fluxrate.co/health`
- Check if API key and meter token are valid
- Check if meter exists, create a new meter if not created yet


## License

MIT
