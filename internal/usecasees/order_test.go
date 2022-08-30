package usecasees

import (
	"testing"
)

func Test_OrderUseCase(t *testing.T) {
	t.Run("SELL Order", func(t *testing.T) {
		newMockGenSELLOrder().run(t)
	})

	t.Run("Settings status NEW", func(t *testing.T) {
		newMockGenSettingsStatusNew().run(t)
	})

	t.Run("Settings status LiquidationSELL", func(t *testing.T) {
		newMockGenSettingsStatusLiquidationSELL().run(t)
	})

	t.Run("Settings status LiquidationBUY", func(t *testing.T) {
		newMockGenSettingsStatusLiquidationBUY().run(t)
	})
}
