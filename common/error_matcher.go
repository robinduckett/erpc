package common

import (
	"regexp"
	"strings"
)

// ErrorMatcher provides functionality to match errors against ignore patterns
type ErrorMatcher struct {
	patterns []*IgnoreErrorConfig
}

// NewErrorMatcher creates a new ErrorMatcher with the given patterns
func NewErrorMatcher(patterns []*IgnoreErrorConfig) *ErrorMatcher {
	if patterns == nil {
		patterns = []*IgnoreErrorConfig{}
	}
	return &ErrorMatcher{
		patterns: patterns,
	}
}

// ShouldIgnoreError checks if the given error should be ignored based on configured patterns
func (m *ErrorMatcher) ShouldIgnoreError(err error) bool {
	if err == nil || m == nil || len(m.patterns) == 0 {
		return false
	}

	// Extract error message and code
	errMsg := err.Error()
	var errCode string

	// Try to extract error code if it's a StandardError
	if stdErr, ok := err.(StandardError); ok {
		base := stdErr.Base()
		if base != nil && base.Code != "" {
			errCode = string(base.Code)
		}
	}

	// Check each pattern
	for _, pattern := range m.patterns {
		if m.matchesPattern(errMsg, errCode, pattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if the error matches a specific ignore pattern
func (m *ErrorMatcher) matchesPattern(errMsg, errCode string, pattern *IgnoreErrorConfig) bool {
	// Determine match type (default to substring if not specified)
	matchType := pattern.MatchType
	if matchType == "" {
		matchType = ErrorMatchTypeSubstring
	}

	// Check code match (if code pattern is provided)
	if pattern.Code != "" {
		if !m.matchString(errCode, pattern.Code, matchType) {
			// Code doesn't match, so this pattern doesn't match
			return false
		}
	}

	// Check message match (if message pattern is provided)
	if pattern.Message != "" {
		if !m.matchString(errMsg, pattern.Message, matchType) {
			// Message doesn't match, so this pattern doesn't match
			return false
		}
	}

	// If we get here and at least one of code or message was specified, we have a match
	return pattern.Code != "" || pattern.Message != ""
}

// matchString performs the actual string matching based on match type
func (m *ErrorMatcher) matchString(target, pattern string, matchType ErrorMatchType) bool {
	if target == "" || pattern == "" {
		return false
	}

	switch matchType {
	case ErrorMatchTypeExact:
		return target == pattern

	case ErrorMatchTypeSubstring:
		return strings.Contains(target, pattern)

	case ErrorMatchTypeWildcard:
		matched, err := WildcardMatch(pattern, target)
		if err != nil {
			// If wildcard matching fails, fallback to substring
			return strings.Contains(target, pattern)
		}
		return matched

	case ErrorMatchTypeRegex:
		re, err := regexp.Compile(pattern)
		if err != nil {
			// If regex compilation fails, fallback to substring
			return strings.Contains(target, pattern)
		}
		return re.MatchString(target)

	default:
		// Default to substring matching
		return strings.Contains(target, pattern)
	}
}

// MergePatterns combines multiple pattern lists into a single ErrorMatcher
// Priority order: upstream-specific patterns > network patterns > project-wide patterns
func MergePatterns(projectPatterns, networkPatterns, upstreamPatterns []*IgnoreErrorConfig) *ErrorMatcher {
	// Combine all patterns with upstream patterns taking highest priority
	allPatterns := make([]*IgnoreErrorConfig, 0, len(projectPatterns)+len(networkPatterns)+len(upstreamPatterns))

	// Add in order of priority (upstream first, so they're checked first)
	allPatterns = append(allPatterns, upstreamPatterns...)
	allPatterns = append(allPatterns, networkPatterns...)
	allPatterns = append(allPatterns, projectPatterns...)

	return NewErrorMatcher(allPatterns)
}
