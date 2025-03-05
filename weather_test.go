package weather_test

import (
	"fmt"
	"testing"

	// "github.com/stretchr/testify/assert"
	weather "github.com/CaptainStandby/openmeteo"
	"github.com/stretchr/testify/require"
)

func TestStuff(t *testing.T) {
	t.Run("Test1", func(t *testing.T) {
		api, err := weather.NewWeatherAPI(54.6167, 9.9167)
		require.NoError(t, err)

		weather, err := api.Current(t.Context())
		require.NoError(t, err)

		fmt.Printf("Weather: %+v\n", weather)
	})
}
