package common

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMatcher_ShouldIgnoreError_ExactMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "transaction underpriced",
			MatchType: ErrorMatchTypeExact,
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "exact match",
			err:      errors.New("transaction underpriced"),
			expected: true,
		},
		{
			name:     "no match - different message",
			err:      errors.New("nonce too low"),
			expected: false,
		},
		{
			name:     "no match - contains but not exact",
			err:      errors.New("error: transaction underpriced for account"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_SubstringMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "transaction underpriced",
			MatchType: ErrorMatchTypeSubstring,
		},
		{
			Message: "nonce too low", // default is substring
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "exact match",
			err:      errors.New("transaction underpriced"),
			expected: true,
		},
		{
			name:     "substring match",
			err:      errors.New("error: transaction underpriced for account"),
			expected: true,
		},
		{
			name:     "second pattern match",
			err:      errors.New("the nonce too low for this tx"),
			expected: true,
		},
		{
			name:     "no match",
			err:      errors.New("insufficient funds"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_WildcardMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "transaction*underpriced",
			MatchType: ErrorMatchTypeWildcard,
		},
		{
			Message:   "nonce*low",
			MatchType: ErrorMatchTypeWildcard,
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "wildcard match with space",
			err:      errors.New("transaction is underpriced"),
			expected: true,
		},
		{
			name:     "wildcard match direct",
			err:      errors.New("transaction underpriced"),
			expected: true,
		},
		{
			name:     "second pattern match",
			err:      errors.New("nonce too low"),
			expected: true,
		},
		{
			name:     "no match",
			err:      errors.New("insufficient funds"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_RegexMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "transaction.+underpriced",
			MatchType: ErrorMatchTypeRegex,
		},
		{
			Message:   "nonce (is )?too low",
			MatchType: ErrorMatchTypeRegex,
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "regex match",
			err:      errors.New("transaction is underpriced"),
			expected: true,
		},
		{
			name:     "regex match with space",
			err:      errors.New("transaction underpriced"),
			expected: true, // space counts as a character for .+
		},
		{
			name:     "regex no match without space",
			err:      errors.New("transactionunderpriced"),
			expected: false, // .+ requires at least one character between
		},
		{
			name:     "second pattern match with optional text",
			err:      errors.New("nonce is too low"),
			expected: true,
		},
		{
			name:     "second pattern match without optional text",
			err:      errors.New("nonce too low"),
			expected: true,
		},
		{
			name:     "no match",
			err:      errors.New("insufficient funds"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_CodeMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Code:      "ErrEndpointCapacityExceeded",
			MatchType: ErrorMatchTypeExact,
		},
		{
			Code:      "ErrEndpoint*",
			MatchType: ErrorMatchTypeWildcard,
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "exact code match",
			err: &BaseError{
				Code:    "ErrEndpointCapacityExceeded",
				Message: "capacity exceeded",
			},
			expected: true,
		},
		{
			name: "wildcard code match",
			err: &BaseError{
				Code:    "ErrEndpointTimeout",
				Message: "timeout",
			},
			expected: true,
		},
		{
			name: "no match - different code",
			err: &BaseError{
				Code:    "ErrInvalidRequest",
				Message: "invalid",
			},
			expected: false,
		},
		{
			name:     "no match - not a StandardError",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_CodeAndMessageMatch(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Code:      "ErrEndpointCapacityExceeded",
			Message:   "rate limit",
			MatchType: ErrorMatchTypeSubstring,
		},
	}

	matcher := NewErrorMatcher(patterns)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "both code and message match",
			err: &BaseError{
				Code:    "ErrEndpointCapacityExceeded",
				Message: "rate limit exceeded",
			},
			expected: true,
		},
		{
			name: "code matches but message doesn't",
			err: &BaseError{
				Code:    "ErrEndpointCapacityExceeded",
				Message: "capacity full",
			},
			expected: false,
		},
		{
			name: "message matches but code doesn't",
			err: &BaseError{
				Code:    "ErrSomeOtherError",
				Message: "rate limit exceeded",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_ShouldIgnoreError_EmptyMatcher(t *testing.T) {
	matcher := NewErrorMatcher(nil)

	err := errors.New("some error")
	result := matcher.ShouldIgnoreError(err)

	if result {
		t.Errorf("Empty matcher should not ignore any errors, got true")
	}
}

func TestErrorMatcher_ShouldIgnoreError_NilError(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "test",
			MatchType: ErrorMatchTypeSubstring,
		},
	}

	matcher := NewErrorMatcher(patterns)
	result := matcher.ShouldIgnoreError(nil)

	if result {
		t.Errorf("Should not ignore nil error, got true")
	}
}

func TestMergePatterns(t *testing.T) {
	projectPatterns := []*IgnoreErrorConfig{
		{Message: "project error"},
	}

	networkPatterns := []*IgnoreErrorConfig{
		{Message: "network error"},
	}

	upstreamPatterns := []*IgnoreErrorConfig{
		{Message: "upstream error"},
	}

	matcher := MergePatterns(projectPatterns, networkPatterns, upstreamPatterns)

	// Test that all patterns are included
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "upstream pattern",
			err:      errors.New("upstream error"),
			expected: true,
		},
		{
			name:     "network pattern",
			err:      errors.New("network error"),
			expected: true,
		},
		{
			name:     "project pattern",
			err:      errors.New("project error"),
			expected: true,
		},
		{
			name:     "no match",
			err:      errors.New("other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorMatcher_InvalidRegex(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Message:   "[invalid(regex",
			MatchType: ErrorMatchTypeRegex,
		},
	}

	matcher := NewErrorMatcher(patterns)

	// Should fallback to substring matching
	err := errors.New("this contains [invalid(regex pattern")
	result := matcher.ShouldIgnoreError(err)

	if !result {
		t.Errorf("Should fallback to substring match for invalid regex, got false")
	}
}

func TestErrorMatcher_NestedStandardError(t *testing.T) {
	patterns := []*IgnoreErrorConfig{
		{
			Code:      "ErrUpstreamRequest",
			MatchType: ErrorMatchTypeExact,
		},
	}

	matcher := NewErrorMatcher(patterns)

	// Create a nested error structure
	innerErr := &BaseError{
		Code:    "ErrUpstreamRequest",
		Message: "upstream failed",
	}

	wrappedErr := fmt.Errorf("wrapped: %w", innerErr)

	// The error matcher checks the error itself, not wrapped errors
	// So this should NOT match since the outer error is not a StandardError
	result := matcher.ShouldIgnoreError(wrappedErr)

	if result {
		t.Errorf("Wrapped errors should not match, got true")
	}

	// But the inner error should match
	result = matcher.ShouldIgnoreError(innerErr)
	if !result {
		t.Errorf("Inner error should match, got false")
	}
}
