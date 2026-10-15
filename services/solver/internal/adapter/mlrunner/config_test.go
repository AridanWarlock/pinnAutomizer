package mlrunner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContainerNameUsesConfiguredPrefixAndRunIdentity(t *testing.T) {
	id := uuid.New()
	require.Equal(t, "pinn-solver-"+id.String(), (Config{}).ContainerName(id))
	cfg := Config{ContainerNamePrefix: "experiment"}
	require.Equal(t, "experiment-"+id.String(), cfg.ContainerName(id))
	require.Equal(t, cfg.ContainerName(id), cfg.ContainerName(id))
	require.NotEqual(t, cfg.ContainerName(id), cfg.ContainerName(uuid.New()))
}
