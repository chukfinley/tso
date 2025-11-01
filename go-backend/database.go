package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	*sql.DB
}

func NewDatabase() (*Database, error) {
	host := getEnv("DB_HOST", "localhost")
	name := getEnv("DB_NAME", "servermanager")
	user := getEnv("DB_USER", "root")
	pass := getEnv("DB_PASS", "")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

