package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramApiToken string
	TelegramChatID   string
	BinanceApiKey    string
	BinanceSecretKey string
	BinanceUrl       string
	BinanceUrl1      string
	BinanceUrl2      string
	BinanceUrl3      string
	DB               *DB
}

type DB struct {
	Host     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

var ErrEnvNotFound = errors.New("err env not found")

func (a *App) loadConfig(confFileName string) error {
	var cfg Config
	var db DB

	err := godotenv.Load(confFileName)
	if err != nil {
		return err
	}

	if cfg.TelegramApiToken, err = cfg.set("TELEGRAM_API_TOKEN"); err != nil {
		return err
	}

	if cfg.TelegramChatID, err = cfg.set("TELEGRAM_CHAT_ID"); err != nil {
		return err
	}

	if cfg.BinanceApiKey, err = cfg.set("BINANCE_API_KEY"); err != nil {
		return err
	}

	if cfg.BinanceSecretKey, err = cfg.set("BINANCE_SECRET_KEY"); err != nil {
		return err
	}

	if cfg.BinanceUrl, err = cfg.set("BINANCE_URL"); err != nil {
		return err
	}

	if cfg.BinanceUrl1, err = cfg.set("BINANCE_URL_1"); err != nil {
		return err
	}

	if cfg.BinanceUrl2, err = cfg.set("BINANCE_URL_2"); err != nil {
		return err
	}

	if cfg.BinanceUrl3, err = cfg.set("BINANCE_URL_3"); err != nil {
		return err
	}

	if db.Host, err = cfg.set("PG_HOST"); err != nil {
		return err
	}

	if db.User, err = cfg.set("PG_USER"); err != nil {
		return err
	}

	if db.Password, err = cfg.set("PG_PASSWORD"); err != nil {
		return err
	}

	if db.DBName, err = cfg.set("PG_DBNAME"); err != nil {
		return err
	}

	if db.SSLMode, err = cfg.set("PG_SSL_MODE"); err != nil {
		return err
	}

	cfg.DB = &db

	a.Config = &cfg

	return nil
}

func (d *DB) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host,
		d.User,
		d.Password,
		d.DBName,
		d.SSLMode)
}

func (c *Config) set(key string) (string, error) {
	if os.Getenv(key) == "" {
		return "", ErrEnvNotFound
	}

	return os.Getenv(key), nil
}
