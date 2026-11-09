// Package notification delivers semantic messages through explicit recipient
// plans, typed renderers, and channel drivers. Send is synchronous and never
// retries or starts background goroutines. Provider acceptance is distinct from
// recipient delivery. See docs/contracts.md for ownership and retry guarantees.
package notification

//go:generate python3 scripts/generate-docs.py
