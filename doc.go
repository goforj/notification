// Package notification sends plain-text messages through configured drivers.
// New(driver).Send(ctx, Message) is the common path. NewDispatcher adds optional
// recipient planning, templates, route policies and process-local deduplication.
// Sends are synchronous, perform no retries and start no background goroutines.
// Successful handling never proves recipient delivery.
package notification

//go:generate go -C docs run ./readme
