package structs

import "binance/models"

const (
	maxPercent = 75
	minPercent = 5
	avgPercent = 35
)

func SELLPatterns(candle *models.Candle) bool {
	if patternBase(candle, models.TREND_DOWN) {
		return true
	}

	if patternShootingStar(candle) {
		return true
	}

	if patternGallows(candle) {
		return true
	}

	return false
}

func BUYPatterns(candle *models.Candle) bool {
	if patternBase(candle, models.TREND_UP) {
		return true
	}

	if patternHammer(candle) {
		return true
	}

	if patternInvertedHammer(candle) {
		return true
	}
	return false
}

func patternBase(candle *models.Candle, trend models.Trend) bool {
	if candle.UpperShadow().WeightPercent > avgPercent &&
		candle.LowerShadow().WeightPercent > avgPercent &&
		candle.Trend() == trend {
		return true
	}

	return false
}

// https://academy.binance.com/ru/articles/beginners-candlestick-patterns

func patternHammer(candle *models.Candle) bool {
	// Молот

	if candle.UpperShadow().WeightPercent < minPercent &&
		candle.LowerShadow().WeightPercent > maxPercent &&
		candle.Trend() == models.TREND_UP {
		return true
	}

	return false
}

func patternInvertedHammer(candle *models.Candle) bool {
	// Перевернутый молот

	if candle.UpperShadow().WeightPercent < maxPercent &&
		candle.LowerShadow().WeightPercent < minPercent &&
		candle.Trend() == models.TREND_UP {
		return true
	}

	return false
}

func patternShootingStar(candle *models.Candle) bool {
	// Падающая звезда

	if candle.UpperShadow().WeightPercent > maxPercent &&
		candle.LowerShadow().WeightPercent < minPercent &&
		candle.Trend() == models.TREND_DOWN {
		return true
	}

	return false
}

func patternGallows(candle *models.Candle) bool {
	//Висельник

	if candle.UpperShadow().WeightPercent < minPercent &&
		candle.LowerShadow().WeightPercent > maxPercent &&
		candle.Trend() == models.TREND_DOWN {
		return true
	}

	return false
}
