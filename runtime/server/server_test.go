package server_test

import (
	"context"
	"testing"

	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/pkg/activity"
	"github.com/fridencao/stardata/runtime/pkg/ratelimit"
	_ "github.com/fridencao/stardata/runtime/resolvers"
	"github.com/fridencao/stardata/runtime/server"
	"github.com/fridencao/stardata/runtime/server/auth"
	"github.com/fridencao/stardata/runtime/testruntime"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func getTestServer(t *testing.T) (*server.Server, string) {
	rt, instanceID := testruntime.NewInstance(t)

	server, err := server.NewServer(context.Background(), &server.Options{}, rt, zap.NewNop(), ratelimit.NewNoop(), activity.NewNoopClient())
	require.NoError(t, err)

	return server, instanceID
}

func testCtx() context.Context {
	return auth.WithClaims(context.Background(), &runtime.SecurityClaims{SkipChecks: true})
}
