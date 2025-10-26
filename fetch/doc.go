// Package fetch provides a comprehensive HTTP client with advanced features
// including retry mechanisms, circuit breakers, logging, and metrics.
//
// Basic Usage:
//
//   // Simple GET request
//   data, err := fetch.Get("https://api.example.com/data")
//   if err != nil {
//       log.Fatal(err)
//   }
//
//   // GET request with context and headers
//   data, err := fetch.CtxGet(ctx, "https://api.example.com/data",
//       fetch.WithContentTypeJSON(),
//       fetch.WithAuthToken("Bearer token"),
//   )
//
//   // POST request with JSON body
//   body := strings.NewReader(`{"key": "value"}`)
//   data, err := fetch.Post("https://api.example.com/data", body,
//       fetch.WithContentTypeJSON(),
//   )
//
// Advanced Features:
//
// Retry Mechanism:
//
//   // Configure retry behavior
//   config := fetch.NewRetryConfig(
//       fetch.WithMaxAttempts(5),
//       fetch.WithBaseDelay(100*time.Millisecond),
//       fetch.WithMaxDelay(5*time.Second),
//   )
//
//   // Use retry with custom function
//   status, data, headers, err := fetch.WithRetry(config, func() (int, []byte, http.Header, error) {
//       return fetch.DoRequestWithOptions("GET", url, opts, nil)
//   })
//
// Circuit Breaker:
//
//   // Create circuit breaker
//   cb := fetch.NewCircuitBreaker(fetch.DefaultCircuitBreakerConfig)
//
//   // Execute with circuit breaker protection
//   err := cb.Execute(func() error {
//       _, err := fetch.Get(url)
//       return err
//   })
//
// Error Handling:
//
//   // Check for specific error types
//   if httpErr, ok := err.(*fetch.HTTPError); ok {
//       fmt.Printf("HTTP %d: %s\n", httpErr.StatusCode, string(httpErr.Body))
//   }
//
//   if retryErr, ok := err.(*fetch.RetryableError); ok {
//       fmt.Printf("Failed after %d attempts: %v\n", retryErr.Attempts, retryErr.Err)
//   }
//
//   if cbErr, ok := err.(*fetch.CircuitBreakerError); ok {
//       fmt.Printf("Circuit breaker open: %v\n", cbErr.Err)
//   }
//
// Security:
//
//   The package enforces TLS 1.2+ by default. For testing purposes,
//   you can create an insecure client:
//
//   insecureClient := fetch.NewInsecureClient()
//   fetch.SetDefaultClient(insecureClient)
//
//   WARNING: Only use insecure client in development environments.
package fetch