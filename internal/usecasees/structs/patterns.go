package structs

import (
	"binance/internal/controllers"
	"binance/models"
	"fmt"

	"github.com/sirupsen/logrus"
)

const (
	maxPercent = 65
	minPercent = 10
)

type Pattern struct {
	tgmController *controllers.TgmController
	logger        *logrus.Logger
}

func NewPattern(
	tgmController *controllers.TgmController,
	logger *logrus.Logger,
) *Pattern {
	return &Pattern{
		tgmController: tgmController,
		logger:        logger,
	}
}

func (p *Pattern) SELLPatterns(candles []models.Candle) bool {

	if p.basePattern(candles) {
		return true
	}

	if p.patternShootingStar(candles) {
		return true
	}

	if p.patternGallows(candles) {
		return true
	}

	return false
}

func (p *Pattern) BUYPatterns(candles []models.Candle) bool {

	if p.basePattern(candles) {
		return true
	}

	if p.patternHammer(candles) {
		return true
	}

	if p.patternInvertedHammer(candles) {
		return true
	}

	if p.patternThreeWhiteSoldiers(candles) {
		return true
	}

	return false
}

func (p *Pattern) basePattern(candles []models.Candle) bool {
	if candles[0].Body().WeightPercent < minPercent {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Base", candles)); err != nil {
			p.logger.
				WithField("func", "basePattern").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}
	return false
}

// https://academy.binance.com/ru/articles/beginners-candlestick-patterns

func (p *Pattern) patternHammer(candles []models.Candle) bool {
	// Молот

	if candles[1].Trend() == models.TrendDown &&
		candles[0].UpperShadow().WeightPercent < minPercent &&
		candles[0].LowerShadow().WeightPercent > maxPercent {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Молот", candles)); err != nil {
			p.logger.
				WithField("func", "patternHammer").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}

	return false
}

func (p *Pattern) patternInvertedHammer(candles []models.Candle) bool {
	// Перевернутый молот

	if candles[1].Trend() == models.TrendDown &&
		candles[0].UpperShadow().WeightPercent > maxPercent &&
		candles[0].LowerShadow().WeightPercent < minPercent {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Перевернутый молот", candles)); err != nil {
			p.logger.
				WithField("func", "patternInvertedHammer").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}

	return false
}

func (p *Pattern) patternThreeWhiteSoldiers(candles []models.Candle) bool {
	// Три белых солдата

	if candles[2].Trend() == models.TrendUp &&
		candles[2].LowerShadow().WeightPercent < minPercent &&
		candles[2].OpenPrice < candles[1].OpenPrice &&
		candles[2].ClosePrice > candles[1].OpenPrice &&
		candles[2].MaxPrice < candles[1].ClosePrice &&
		//
		candles[1].Trend() == models.TrendUp &&
		candles[1].LowerShadow().WeightPercent < minPercent &&
		candles[1].OpenPrice < candles[0].OpenPrice &&
		candles[1].ClosePrice > candles[0].OpenPrice &&
		candles[1].MaxPrice < candles[0].ClosePrice &&
		//
		candles[0].Trend() == models.TrendUp &&
		candles[0].LowerShadow().WeightPercent < minPercent {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Три белых солдата", candles)); err != nil {
			p.logger.
				WithField("func", "patternInvertedHammer").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}

	return false
}

func (p *Pattern) patternShootingStar(candles []models.Candle) bool {
	// Падающая звезда

	if candles[1].Trend() == models.TrendUp &&
		candles[0].UpperShadow().WeightPercent > maxPercent &&
		candles[0].LowerShadow().WeightPercent < minPercent &&
		candles[0].Trend() == models.TrendDown {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Падающая звезда", candles)); err != nil {
			p.logger.
				WithField("func", "patternGallows").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}

	return false
}

func (p *Pattern) patternGallows(candles []models.Candle) bool {
	//Висельник

	if candles[1].Trend() == models.TrendUp &&
		candles[0].UpperShadow().WeightPercent < minPercent &&
		candles[0].LowerShadow().WeightPercent > maxPercent &&
		candles[0].Trend() == models.TrendDown {

		if err := p.tgmController.Send(
			fmt.Sprintf("[ Pattern Detected ]\n%s\n%+v", "Висельник", candles)); err != nil {
			p.logger.
				WithField("func", "patternGallows").
				WithField("method", "Pattern").
				Debug(err)
		}

		return true
	}

	return false
}
