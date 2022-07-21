package structs

import "binance/models"

func SELLPatterns(candle *models.Candle) bool {
	if patternBase(candle) {
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
	if patternBase(candle) {
		return true
	}
	return false
}

func patternBase(candle *models.Candle) bool {
	if candle.UpperShadow().WeightPercent > 35 && candle.LowerShadow().WeightPercent > 35 {
		return true
	}

	return false
}

// https://academy.binance.com/ru/articles/beginners-candlestick-patterns

func patternShootingStar(candle *models.Candle) bool {
	// Падающая звезда

	if candle.UpperShadow().WeightPercent > 60 && candle.LowerShadow().WeightPercent < 10 {
		return true
	}

	return false
}

func patternGallows(candle *models.Candle) bool {
	//Висельник

	if candle.UpperShadow().WeightPercent < 10 && candle.LowerShadow().WeightPercent > 60 {
		return true
	}

	return false
}
