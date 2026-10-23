package config

import (
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestWorkerConfigDefaultsAndValidation(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	err := env.ParseWithOptions(&cfg, env.Options{
		Environment: map[string]string{
			"ENVIRONMENT":                       "test",
			"LOGGER_ENVIRONMENT":                "test",
			"LOGGER_LEVEL":                      "info",
			"LOGGER_FOLDER":                     "unused",
			"PINN_RUNNER_IMAGE":                 "unused",
			"PINN_RUNNER_HOST_TASKS_DATA_DIR":   "unused",
			"PINN_RUNNER_HOST_TASKS_OUTPUT_DIR": "unused",
			"PINN_RUNNER_TIMEOUT":               "1m",
			"SQLITE_PATH":                       "unused",
			"KAFKA_PRODUCER_BROKER":             "unused",
			"KAFKA_CONSUMER_BROKER":             "unused",
			"KAFKA_CONSUMER_GROUP_ID":           "unused",
		},
	})
	require.NoError(t, err)
	require.NoError(t, cfg.Validate())
	require.Equal(t, 1, cfg.Recovery.ActiveLimit)
	require.Equal(t, 8, cfg.Recovery.BatchSize)
	require.Equal(t, time.Second, cfg.Recovery.Interval)
	require.Equal(t, 10*time.Second, cfg.Recovery.AttemptTimeout)
	require.Equal(t, time.Second, cfg.OutboxInterval)
	for _, change := range []func(*Config){
		func(c *Config) { c.Recovery.ActiveLimit = 0 },
		func(c *Config) { c.Recovery.BatchSize = 0 },
		func(c *Config) { c.Recovery.Interval = -time.Second },
		func(c *Config) { c.Recovery.AttemptTimeout = 0 },
		func(c *Config) { c.OutboxInterval = 0 },
		func(c *Config) { c.PinnRunner.Timeout = 0 },
	} {
		invalid := cfg
		change(&invalid)
		require.ErrorIs(t, invalid.Validate(), errs.ErrInvalidArgument, "config: %+v", invalid.Recovery)
	}
}
