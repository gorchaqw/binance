package usecasees

import (
	"testing"
)

func Test_OrderUseCase(t *testing.T) {
	t.Run("order SELL", func(t *testing.T) {
		newMonitoring(testCaseOrderSELL).run(t)
	})

	t.Run("liquidation BUY", func(t *testing.T) {
		newMonitoring(testCaseLiquidationBUY).run(t)
	})

	t.Run("liquidation SELL", func(t *testing.T) {
		newMonitoring(testCaseLiquidationSELL).run(t)
	})

	t.Run("settings status NEW", func(t *testing.T) {
		newMonitoring(testSettingsStatusNEW).run(t)
	})
}
