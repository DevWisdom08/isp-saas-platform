package database

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "time"

    _ "github.com/lib/pq"
)

type DB struct {
    *sql.DB
}

func Connect() (*DB, error) {
    host := getEnv("DB_HOST", "localhost")
    port := getEnv("DB_PORT", "5432")
    user := getEnv("DB_USER", "postgres")
    password := getEnv("DB_PASSWORD", "password")
    dbname := getEnv("DB_NAME", "isp_saas")
    sslmode := getEnv("DB_SSLMODE", "disable")

    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        host, port, user, password, dbname, sslmode,
    )

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{db}, nil
}

func (db *DB) RunMigrations(migrationsPath string) error {
    files, err := os.ReadDir(migrationsPath)
    if err != nil {
        return fmt.Errorf("failed to read migrations directory: %w", err)
    }

    var sqlFiles []string
    for _, file := range files {
        if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
            sqlFiles = append(sqlFiles, file.Name())
        }
    }
    sort.Strings(sqlFiles)

    for _, file := range sqlFiles {
        content, err := os.ReadFile(filepath.Join(migrationsPath, file))
        if err != nil {
            return fmt.Errorf("failed to read migration %s: %w", file, err)
        }

        if _, err := db.Exec(string(content)); err != nil {
            return fmt.Errorf("failed to execute migration %s: %w", file, err)
        }

        fmt.Printf("Migration executed: %s\n", file)
    }

    return nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
