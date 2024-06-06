package database

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v4"
)

func PostgreSQLConnect() (*pgx.Conn, error) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// Get database connection details from environment variables
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// Establish connection to PostgreSQL database
	dataSourceName := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
	conn, err := pgx.Connect(context.Background(), dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	// Ping database
	err = conn.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	fmt.Println("Connected to PostgreSQL database!")
	return conn, nil
}