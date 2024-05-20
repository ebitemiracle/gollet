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


// package database

// import (

// 	"fmt"
// 	"os"
// 	"database/sql"

// 	"github.com/joho/godotenv"
// )

// func MySQLConnect() (*sql.DB, error) {
// 	// Load environment variables from .env file
// 	err := godotenv.Load()
// 	if err != nil {
// 		return nil, fmt.Errorf("error loading .env file: %w", err)
// 	}

// 	// Get database connection details from environment variables
// 	dbUsername := os.Getenv("DB_USERNAME")
// 	dbPassword := os.Getenv("DB_PASSWORD")
// 	dbHost := os.Getenv("DB_HOST")
// 	dbPort := os.Getenv("DB_PORT")
// 	dbName := os.Getenv("DB_NAME")

// 	// Establish connection to MySQL database
// 	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUsername, dbPassword, dbHost, dbPort, dbName)
// 	db, err := sql.Open("mysql", dataSourceName)
// 	if err != nil {
// 		return nil, fmt.Errorf("error connecting to database: %w", err)
// 	}

// 	// Ping database
// 	err = db.Ping()
// 	if err != nil {
// 		return nil, fmt.Errorf("error pinging database: %w", err)
// 	}

// 	fmt.Println("Connected to MySQL database!")
// 	return db, nil
// }