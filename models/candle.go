package models

import "time"

type Candle struct {
	ID         int       `db:"id"`
	Symbol     string    `db:"symbol"`
	OpenPrice  float64   `db:"open_price"`
	ClosePrice float64   `db:"close_price"`
	MaxPrice   float64   `db:"max_price"`
	MinPrice   float64   `db:"min_price"`
	TimeFrame  string    `db:"time_frame"`
	OpenTime   time.Time `db:"open_time"`
	CloseTime  time.Time `db:"close_time"`
	CreatedAt  time.Time `db:"created_at"`
}

type Shadow struct {
	Weight        float64
	WeightPercent float64
}
type Body struct {
	Weight        float64
	WeightPercent float64
}

func (c *Candle) Body() *Body {
	deltaMAXMIN := c.MaxPrice - c.MinPrice

	var b *Body
	switch true {
	case c.ClosePrice >= c.OpenPrice:
		b.Weight = c.ClosePrice - c.OpenPrice
	case c.ClosePrice < c.OpenPrice:
		b.Weight = c.OpenPrice - c.ClosePrice
	}

	b.WeightPercent = b.Weight * 100 / deltaMAXMIN

	return b
}

func (c *Candle) UpperShadow() *Shadow {
	deltaMAXMIN := c.MaxPrice - c.MinPrice

	var s *Shadow
	switch true {
	case c.ClosePrice >= c.OpenPrice:
		s.Weight = c.MaxPrice - c.ClosePrice
	case c.ClosePrice < c.OpenPrice:
		s.Weight = c.MaxPrice - c.OpenPrice
	}

	s.WeightPercent = s.Weight * 100 / deltaMAXMIN

	return s
}

func (c *Candle) LowerShadow() *Shadow {
	deltaMAXMIN := c.MaxPrice - c.MinPrice

	var s *Shadow
	switch true {
	case c.ClosePrice >= c.OpenPrice:
		s.Weight = c.OpenPrice - c.MinPrice
	case c.ClosePrice < c.OpenPrice:
		s.Weight = c.ClosePrice - c.MinPrice
	}

	s.WeightPercent = s.Weight * 100 / deltaMAXMIN

	return s
}
