package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cloudnexus/server/pkg/snowflake"
)

type Config struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

func NewPostgres(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	registerSnowflakeCallback(db)

	log.Println("postgres connected")
	return db, nil
}

func registerSnowflakeCallback(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("snowflake:gen_id", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
		if field := db.Statement.Schema.LookUpField("ID"); field != nil {
			_, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
			if isZero {
				field.Set(db.Statement.Context, db.Statement.ReflectValue, snowflake.Uint64())
			}
		}
	})
}

