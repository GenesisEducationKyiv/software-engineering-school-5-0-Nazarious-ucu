// architecture_test.go
package architecture_test

import (
	"testing"

	"github.com/mstrYoda/go-arctest/pkg/arctest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mod = `github\.com/Nazarious-ucu/weather-subscription-api`

func TestLayeredArchitecture(t *testing.T) {
	arch, err := arctest.New("../")
	require.NoError(t, err)

	// 2. Parse *all* packages beneath your module
	err = arch.ParsePackages()
	require.NoError(t, err, "failed to parse packages")

	// 3. Define your layers (regexes match import-path prefixes)
	domainLayer, err := arctest.NewLayer("domain", `^`+mod+`/internal/models`)
	require.NoError(t, err)

	appLayer, err := arctest.NewLayer("application",
		`^`+mod+`/internal/(app|cfg|notifier|services/logger|services/email|services/subscription)`)
	require.NoError(t, err)

	userLayer, err := arctest.NewLayer("application", `^`+mod+`/internal/services/handlers`)
	require.NoError(t, err)

	infraLayer, err := arctest.NewLayer("infrastructure",
		`^`+mod+`internal/(repository/sqlite|repository/cache|emailer|services/weather
|services/metrics)`,
		`^pkg/logger`,
	)
	require.NoError(t, err)

	layered := arch.NewLayeredArchitecture(domainLayer, appLayer, infraLayer, userLayer)

	// 5. Declare allowed dependencies between layers:
	err = appLayer.DependsOnLayer(domainLayer)
	assert.NoError(t, err)

	err = infraLayer.DependsOnLayer(domainLayer)
	assert.NoError(t, err)

	err = infraLayer.DependsOnLayer(appLayer)
	assert.NoError(t, err)

	err = infraLayer.DependsOnLayer(userLayer)
	assert.NoError(t, err)

	err = userLayer.DependsOnLayer(domainLayer)
	assert.NoError(t, err)

	violations, err := layered.Check()
	require.NoError(t, err)

	assert.Len(t, violations, 0)

	for _, v := range violations {
		assert.Failf(t, "", "violation: %s", v)
	}
}
