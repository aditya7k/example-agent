package registry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"example.com/gh-models-agent/internal/adapters/tools/clock"
	"example.com/gh-models-agent/internal/adapters/tools/registry"
)

func TestRegistry_Get(t *testing.T) {
	c := clock.New()
	r := registry.New(c)

	t.Run("resolves a registered tool", func(t *testing.T) {
		got, ok := r.Get(clock.ToolName)
		require.True(t, ok)
		assert.NotNil(t, got)
	})

	t.Run("misses unknown tools", func(t *testing.T) {
		_, ok := r.Get("nope")
		assert.False(t, ok)
	})
}

func TestRegistry_Schemas(t *testing.T) {
	r := registry.New(clock.New())
	schemas := r.Schemas()

	require.Len(t, schemas, 1)
	assert.Equal(t, clock.ToolName, schemas[0].Name)
}

// Compile-time assurance the registry is usable in agent contexts.
func TestRegistry_ContextFreeOfPanics(t *testing.T) {
	r := registry.New(clock.New())
	tool, ok := r.Get(clock.ToolName)
	require.True(t, ok)

	_, err := tool.Execute(context.Background(), map[string]any{})
	require.NoError(t, err)
}
