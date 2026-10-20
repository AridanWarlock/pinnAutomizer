package mlrunner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContainerNameUsesConfiguredPrefixAndRunIdentity(t *testing.T) {
	id := uuid.New()
	require.Equal(t, "pinn-solver-"+id.String(), (&PinnRunner{}).containerName(id))
	runner := &PinnRunner{containerNamePrefix: "experiment"}
	require.Equal(t, "experiment-"+id.String(), runner.containerName(id))
	require.Equal(t, runner.containerName(id), runner.containerName(id))
	require.NotEqual(t, runner.containerName(id), runner.containerName(uuid.New()))
}
