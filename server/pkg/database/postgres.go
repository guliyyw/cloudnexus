package database

import (
	"log"
	"reflect"

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
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(20)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(5)
	}

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
			rv := db.Statement.ReflectValue
			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < rv.Len(); i++ {
					elem := rv.Index(i)
					if elem.Kind() == reflect.Ptr {
						elem = elem.Elem()
					}
					_, isZero := field.ValueOf(db.Statement.Context, elem)
					if isZero {
						field.Set(db.Statement.Context, elem, snowflake.Uint64())
					}
				}
			default:
				_, isZero := field.ValueOf(db.Statement.Context, rv)
				if isZero {
					field.Set(db.Statement.Context, rv, snowflake.Uint64())
				}
			}
		}
	})
}

