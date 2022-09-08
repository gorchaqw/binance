package postgres

import (
	"binance/models"
	"time"
)

//go:generate mockery --case=snake --name=OrderRepo
//go:generate mockery --case=snake --name=PriceRepo

type OrderRepo interface {
	Store(m *models.Order) error
	GetLast(symbol string) (*models.Order, error)
	GetFirst(symbol string) (*models.Order, error)
	GetByID(id int) (*models.Order, error)
	GetBySessionID(sessionID string) ([]models.Order, error)
	GetBySessionIDWithSide(sessionID, side string) ([]models.Order, error)
	GetLastWithInterval(symbol string, sTime, eTime time.Time) ([]models.Order, error)
	SetActualPrice(id int, price float64) error
	SetTry(id, try int) error
	SetStatus(id int64, status string) error
}

type PriceRepo interface {
	GetByCreatedByInterval(symbol string, sTime, eTime time.Time) ([]models.Price, error)
	GetMaxMinByCreatedByInterval(symbol string, sTime, eTime time.Time) (float64, float64, float64, float64, error)
	Store(m *models.Price) (err error)
	GetLast(symbol string, sTime, eTime time.Time) (*models.Price, error)
	GetByID(symbol string, id uint) (*models.Price, error)
}
