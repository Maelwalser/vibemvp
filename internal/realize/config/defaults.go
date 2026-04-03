package config

// DefaultModel is the Claude model used when no override is specified.
const DefaultModel = "claude-opus-4-6"

// DefaultMaxTokens is the maximum output token budget per agent call.
const DefaultMaxTokens = int64(64000)

// MaxSkillBytes is the maximum number of characters stored per skill document.
const MaxSkillBytes = 2000

// MaxFileChars is the maximum characters included from a single dependency file.
const MaxFileChars = 1500

// MaxTotalChars is the total character budget across all dependency outputs.
const MaxTotalChars = 8000

// RateLimitBackoffBase is the per-attempt multiplier in seconds for rate-limit backoff.
// Wait = (attempt+1) * RateLimitBackoffBase seconds.
const RateLimitBackoffBase = 60
