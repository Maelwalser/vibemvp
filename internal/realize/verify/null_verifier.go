package verify

import "context"

// NullVerifier always returns Passed:true. Used for languages/tasks without a
// suitable local checker (Java, Kotlin, docs, etc.).
type NullVerifier struct{}

func (n *NullVerifier) Language() string { return "null" }

func (n *NullVerifier) Verify(_ context.Context, _ string, _ []string) (*Result, error) {
	return &Result{Passed: true, Output: "(no verifier for this task kind)"}, nil
}
