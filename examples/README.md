# Fluxrate Golang SDK Examples

This directory contains example applications demonstrating how to use the Fluxrate SDK.

## Example Application

### Prerequisites

1. **Generate an API key** in your dashboard:
   - Navigate to Settings → API Keys.
   - Click "Generate New Key".
   - Copy the key.

2. **Create a meter and get meter token** from the dashboard, e.g.
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
# From the root of the repo
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

📊 Example 4: Immediate tracking
✅ Event sent immediately: <uuid>

⏳ Waiting for batch to flush...

📊 Example 5: Manual flush
✅ Flushed 3 events, 0 failed

📊 Example 6: Disallowed customer (filtered out)
   Note: SDK is configured to only track specific customers
[BillingSDK] [DEBUG] Customer not in allowed list: user_not_allowed
✅ Event skipped (customer not in allowed list)
   Check debug logs above for: 'Customer not in allowed list'

🛑 Shutting down...
✅ Shutdown complete
```

### What This Example Demonstrates

- ✅ Basic event tracking
- ✅ Tracking with custom metadata
- ✅ Idempotency protection
- ✅ Immediate vs batched tracking
- ✅ Manual batch flushing
- ✅ Customer filtering (AllowedCustomers)
- ✅ Graceful shutdown
