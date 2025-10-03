package upstream

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/erpc/erpc/common"
	"github.com/erpc/erpc/health"
	"github.com/erpc/erpc/util"
	"github.com/rs/zerolog"
)

func init() {
	util.ConfigureTestLogger()
}

func TestUpstream_IgnoreErrors_SubstringMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		IgnoreErrors: []*common.IgnoreErrorConfig{
			{
				Message:   "transaction underpriced",
				MatchType: common.ErrorMatchTypeSubstring,
			},
			{
				Message:   "nonce too low",
				MatchType: common.ErrorMatchTypeSubstring,
			},
		},
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	// Test that ignored errors are correctly identified
	tests := []struct {
		name            string
		err             error
		shouldBeIgnored bool
	}{
		{
			name:            "transaction underpriced error",
			err:             fmt.Errorf("Error: transaction underpriced"),
			shouldBeIgnored: true,
		},
		{
			name:            "nonce too low error",
			err:             fmt.Errorf("nonce too low for this transaction"),
			shouldBeIgnored: true,
		},
		{
			name:            "different error",
			err:             fmt.Errorf("insufficient funds"),
			shouldBeIgnored: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := upstream.shouldIgnoreError(tt.err)
			if result != tt.shouldBeIgnored {
				t.Errorf("shouldIgnoreError() = %v, want %v", result, tt.shouldBeIgnored)
			}
		})
	}
}

func TestUpstream_IgnoreErrors_CodeMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		IgnoreErrors: []*common.IgnoreErrorConfig{
			{
				Code:      "ErrEndpointCapacityExceeded",
				MatchType: common.ErrorMatchTypeExact,
			},
		},
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	// Test with StandardError
	err1 := &common.BaseError{
		Code:    "ErrEndpointCapacityExceeded",
		Message: "capacity exceeded",
	}

	if !upstream.shouldIgnoreError(err1) {
		t.Errorf("shouldIgnoreError() should return true for matching error code")
	}

	// Test with different error code
	err2 := &common.BaseError{
		Code:    "ErrSomeOtherError",
		Message: "some error",
	}

	if upstream.shouldIgnoreError(err2) {
		t.Errorf("shouldIgnoreError() should return false for non-matching error code")
	}
}

func TestUpstream_IgnoreErrors_WildcardMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		IgnoreErrors: []*common.IgnoreErrorConfig{
			{
				Message:   "transaction*underpriced",
				MatchType: common.ErrorMatchTypeWildcard,
			},
			{
				Code:      "ErrEndpoint*",
				MatchType: common.ErrorMatchTypeWildcard,
			},
		},
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	// Test message wildcard
	err1 := fmt.Errorf("transaction is underpriced")
	if !upstream.shouldIgnoreError(err1) {
		t.Errorf("shouldIgnoreError() should return true for wildcard message match")
	}

	// Test code wildcard
	err2 := &common.BaseError{
		Code:    "ErrEndpointTimeout",
		Message: "timeout",
	}
	if !upstream.shouldIgnoreError(err2) {
		t.Errorf("shouldIgnoreError() should return true for wildcard code match")
	}

	// Test non-match
	err3 := fmt.Errorf("insufficient funds")
	if upstream.shouldIgnoreError(err3) {
		t.Errorf("shouldIgnoreError() should return false for non-matching error")
	}
}

func TestUpstream_IgnoreErrors_NoConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		// No IgnoreErrors configured
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	err := fmt.Errorf("any error")
	if upstream.shouldIgnoreError(err) {
		t.Errorf("shouldIgnoreError() should return false when no ignoreErrors configured")
	}
}

func TestHealthTracker_RecordUpstreamFailure_WithIgnoreErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		IgnoreErrors: []*common.IgnoreErrorConfig{
			{
				Message:   "transaction underpriced",
				MatchType: common.ErrorMatchTypeSubstring,
			},
		},
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	method := "eth_sendRawTransaction"

	// Record an ignored error
	ignoredErr := fmt.Errorf("transaction underpriced")
	tracker.RecordUpstreamFailure(upstream, method, ignoredErr)

	// Check that error was not recorded
	metrics := tracker.GetUpstreamMethodMetrics(upstream, method)
	if metrics.ErrorsTotal.Load() != 0 {
		t.Errorf("ErrorsTotal should be 0 for ignored error, got %d", metrics.ErrorsTotal.Load())
	}

	// Record a non-ignored error
	normalErr := fmt.Errorf("insufficient funds")
	tracker.RecordUpstreamFailure(upstream, method, normalErr)

	// Check that error was recorded
	metrics = tracker.GetUpstreamMethodMetrics(upstream, method)
	if metrics.ErrorsTotal.Load() != 1 {
		t.Errorf("ErrorsTotal should be 1 for non-ignored error, got %d", metrics.ErrorsTotal.Load())
	}
}

func TestUpstream_IgnoreErrors_RegexMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter())
	tracker := health.NewTracker(&logger, "test-project", 10*time.Second)

	cfg := &common.UpstreamConfig{
		Id:       "test-upstream",
		Endpoint: "http://rpc.example.com",
		Type:     common.UpstreamTypeEvm,
		Evm: &common.EvmUpstreamConfig{
			ChainId: 1,
		},
		IgnoreErrors: []*common.IgnoreErrorConfig{
			{
				Message:   "transaction.+underpriced",
				MatchType: common.ErrorMatchTypeRegex,
			},
		},
	}

	upstream := &Upstream{
		ProjectId:      "test-project",
		logger:         &logger,
		appCtx:         ctx,
		config:         cfg,
		metricsTracker: tracker,
	}

	// Test regex match with multiple characters
	err1 := fmt.Errorf("transaction is underpriced")
	if !upstream.shouldIgnoreError(err1) {
		t.Errorf("shouldIgnoreError() should return true for regex match")
	}

	// Test regex match with space (space counts as a character for .+)
	err2 := fmt.Errorf("transaction underpriced")
	if !upstream.shouldIgnoreError(err2) {
		t.Errorf("shouldIgnoreError() should return true for regex match with space")
	}

	// Test non-match (no characters between, .+ requires at least one)
	err3 := fmt.Errorf("transactionunderpriced")
	if upstream.shouldIgnoreError(err3) {
		t.Errorf("shouldIgnoreError() should return false when regex doesn't match")
	}
}
