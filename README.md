# Fluxrate Golang SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/Fluxratehq/fluxrate-golang-sdk.svg)](https://pkg.go.dev/github.com/Fluxratehq/fluxrate-golang-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/Fluxratehq/fluxrate-golang-sdk)](https://goreportcard.com/report/github.com/Fluxratehq/fluxrate-golang-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Official Go SDK for [Fluxrate](https://fluxrate.co) - Usage-based billing for SaaS.

## Installation

```bash
go get github.com/Fluxratehq/fluxrate-golang-sdk@v0.1.0
```

Or to get the latest version:

```bash
go get github.com/Fluxratehq/fluxrate-golang-sdk@latest
```

## Example

### Prerequisites

1. **Generate an API key** in your dashboard:
   - Navigate to Settings → API Keys.
   - Click "Generate New Key".
   - Copy the key.

1. **Create a meter and get meter token** from the dashboard, e.g.
   - Create a meter, e.g.
    **API Calls Meter**: Name: `api_calls`, Unit: `calls`, Aggregation: `sum`
   - Go to meter details page and copy meter token.

3. **Set environment variables**:
   ```bash
   export BILLING_API_KEY="your-api-key-here"
   export BILLING_METER_TOKEN="your-meter-token"
   ```

### Running the Example

```bash
cd sdk/golang/examples
go run main.go
```

### Expected Output

```
🎯 Billing SDK Example Application
===================================

[BillingSDK] SDK initialized: version=v0.1.0, apiUrl=..., batching=true, batchSize=10

🚀 Simulating API usage...

📊 Example 1: Track API call
[BillingSDK] Event queued for batching (1/10)
✅ Event tracked: <uuid>

📊 Example 2: Track with metadata
[BillingSDK] Event queued for batching (2/10)
✅ Event with metadata tracked

📊 Example 3: Track with idempotency
[BillingSDK] Event queued for batching (3/10)
✅ Event with idempotency tracked
✅ Duplicate prevented (same event returned)

...
```

## Integration

### HTTP Server Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"

    "github.com/Fluxratehq/fluxrate-golang-sdk/billing"
)

var billingSDK *billing.SDK

func main() {
    // Initialize SDK
    config := billing.DefaultConfig(os.Getenv("BILLING_API_KEY"))

    var err error
    billingSDK, err = billing.NewSDK(config)
    if err != nil {
        log.Fatal(err)
    }
    defer billingSDK.Shutdown(context.Background())

    http.HandleFunc("/api/data", handleData)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleData(w http.ResponseWriter, r *http.Request) {
    // Your business logic
    data := getData()

    // Track usage
    userID := r.Header.Get("X-User-ID")
    _, err := billingSDK.Track(r.Context(), billing.TrackEventParams{
        MeterToken:         os.Getenv("BILLING_METER_TOKEN"),
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
    "os"

    "github.com/Fluxratehq/fluxrate-golang-sdk/billing"
    "github.com/gin-gonic/gin"
)

var billingSDK *billing.SDK

func main() {
    // Initialize SDK
    config := billing.DefaultConfig(os.Getenv("BILLING_API_KEY"))
    billingSDK, _ = billing.NewSDK(config)
    defer billingSDK.Shutdown(context.Background())

    r := gin.Default()

    // Middleware to track API usage
    r.Use(func(c *gin.Context) {
        c.Next()

        // Track after request completes
        userID := c.GetHeader("X-User-ID")
        if userID != "" {
            go func() {
                billingSDK.Track(context.Background(), billing.TrackEventParams{
                    MeterToken:         os.Getenv("BILLING_METER_TOKEN"),
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
