package test

import (
	"fmt"
	"testing"

	"github.com/erpc/erpc/common"
	"github.com/erpc/erpc/util"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func init() {
	util.ConfigureTestLogger()
}

func TestIgnoreErrors_EndToEnd_ConfigParsing(t *testing.T) {
	yamlConfig := `
logLevel: DEBUG
projects:
  - id: test-project
    upstreamDefaults:
      ignoreErrors:
        - message: "default ignored error"
          matchType: substring
    networkDefaults:
      ignoreErrors:
        - message: "network ignored error"
          matchType: substring
    upstreams:
      - id: upstream-with-ignore
        endpoint: http://rpc1.example.com
        ignoreErrors:
          - message: "transaction underpriced"
            matchType: substring
          - message: "nonce too low"
            matchType: substring
          - code: "ErrEndpointCapacityExceeded"
            matchType: exact
      - id: upstream-without-ignore
        endpoint: http://rpc2.example.com
    networks:
      - architecture: evm
        evm:
          chainId: 1
`

	// Use in-memory filesystem for tests (following project conventions)
	fs := afero.NewMemMapFs()
	cfgFile, err := afero.TempFile(fs, "", "erpc.yaml")
	assert.NoError(t, err)
	_, err = cfgFile.WriteString(yamlConfig)
	assert.NoError(t, err)

	// Load config
	cfg, err := common.LoadConfig(fs, cfgFile.Name(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify project-level ignoreErrors
	project := cfg.Projects[0]
	assert.NotNil(t, project.UpstreamDefaults)
	assert.Len(t, project.UpstreamDefaults.IgnoreErrors, 1)
	assert.Equal(t, "default ignored error", project.UpstreamDefaults.IgnoreErrors[0].Message)
	assert.Equal(t, common.ErrorMatchTypeSubstring, project.UpstreamDefaults.IgnoreErrors[0].MatchType)

	// Verify network-level ignoreErrors
	assert.NotNil(t, project.NetworkDefaults)
	assert.Len(t, project.NetworkDefaults.IgnoreErrors, 1)
	assert.Equal(t, "network ignored error", project.NetworkDefaults.IgnoreErrors[0].Message)

	// Verify upstream-level ignoreErrors
	assert.Len(t, project.Upstreams, 2)
	upstream1 := project.Upstreams[0]
	assert.Equal(t, "upstream-with-ignore", upstream1.Id)
	assert.Len(t, upstream1.IgnoreErrors, 3)
	assert.Equal(t, "transaction underpriced", upstream1.IgnoreErrors[0].Message)
	assert.Equal(t, "nonce too low", upstream1.IgnoreErrors[1].Message)
	assert.Equal(t, "ErrEndpointCapacityExceeded", upstream1.IgnoreErrors[2].Code)

	// Second upstream should not have ignoreErrors
	upstream2 := project.Upstreams[1]
	assert.Equal(t, "upstream-without-ignore", upstream2.Id)
	assert.Nil(t, upstream2.IgnoreErrors)
}

func TestIgnoreErrors_EndToEnd_ErrorMatcher(t *testing.T) {
	// Test error matcher with various patterns
	patterns := []*common.IgnoreErrorConfig{
		{
			Message:   "transaction underpriced",
			MatchType: common.ErrorMatchTypeSubstring,
		},
		{
			Message:   "nonce too low",
			MatchType: common.ErrorMatchTypeSubstring,
		},
		{
			Code:      "ErrEndpointCapacityExceeded",
			MatchType: common.ErrorMatchTypeExact,
		},
		{
			Message:   "transaction*is*underpriced",
			MatchType: common.ErrorMatchTypeWildcard,
		},
		{
			Message:   "nonce (is )?too low",
			MatchType: common.ErrorMatchTypeRegex,
		},
	}

	matcher := common.NewErrorMatcher(patterns)

	// Test various error scenarios
	testCases := []struct {
		name         string
		err          error
		shouldIgnore bool
	}{
		{
			name:         "transaction underpriced - substring",
			err:          fmt.Errorf("transaction underpriced"),
			shouldIgnore: true,
		},
		{
			name:         "nonce too low - substring",
			err:          fmt.Errorf("nonce too low"),
			shouldIgnore: true,
		},
		{
			name: "capacity exceeded - code",
			err: &common.BaseError{
				Code:    "ErrEndpointCapacityExceeded",
				Message: "rate limit exceeded",
			},
			shouldIgnore: true,
		},
		{
			name:         "transaction is underpriced - wildcard",
			err:          fmt.Errorf("transaction is underpriced"),
			shouldIgnore: true,
		},
		{
			name:         "nonce is too low - regex",
			err:          fmt.Errorf("nonce is too low"),
			shouldIgnore: true,
		},
		{
			name:         "insufficient funds - no match",
			err:          fmt.Errorf("insufficient funds"),
			shouldIgnore: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tc.err)
			assert.Equal(t, tc.shouldIgnore, result,
				"Error matching result mismatch for: %s", tc.err.Error())
		})
	}
}

func TestIgnoreErrors_EndToEnd_MergePatterns(t *testing.T) {
	projectPatterns := []*common.IgnoreErrorConfig{
		{Message: "project error", MatchType: common.ErrorMatchTypeSubstring},
	}

	networkPatterns := []*common.IgnoreErrorConfig{
		{Message: "network error", MatchType: common.ErrorMatchTypeSubstring},
	}

	upstreamPatterns := []*common.IgnoreErrorConfig{
		{Message: "upstream error", MatchType: common.ErrorMatchTypeSubstring},
	}

	matcher := common.MergePatterns(projectPatterns, networkPatterns, upstreamPatterns)

	// Test that all pattern levels are included
	testCases := []struct {
		name         string
		err          error
		shouldIgnore bool
	}{
		{
			name:         "upstream pattern",
			err:          fmt.Errorf("upstream error occurred"),
			shouldIgnore: true,
		},
		{
			name:         "network pattern",
			err:          fmt.Errorf("network error occurred"),
			shouldIgnore: true,
		},
		{
			name:         "project pattern",
			err:          fmt.Errorf("project error occurred"),
			shouldIgnore: true,
		},
		{
			name:         "no match",
			err:          fmt.Errorf("other error"),
			shouldIgnore: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.ShouldIgnoreError(tc.err)
			assert.Equal(t, tc.shouldIgnore, result,
				"Pattern merge result mismatch for: %s", tc.err.Error())
		})
	}
}
