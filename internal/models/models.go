package models

import (
    "database/sql"
    "time"
)

type User struct {
    ID           int            `json:"id"`
    Email        string         `json:"email"`
    PasswordHash string         `json:"-"`
    Role         string         `json:"role"`
    FullName     sql.NullString `json:"full_name"`
    IsActive     bool           `json:"is_active"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
}

type Plan struct {
    ID               int             `json:"id"`
    Name             string          `json:"name"`
    Description      sql.NullString  `json:"description"`
    PriceMonthly     float64         `json:"price_monthly"`
    BandwidthLimit   sql.NullInt64   `json:"bandwidth_limit_mbps"`
    CacheSizeGB      sql.NullInt64   `json:"cache_size_gb"`
    MaxConnections   sql.NullInt64   `json:"max_connections"`
    Features         []byte          `json:"features"`
    IsActive         bool            `json:"is_active"`
    CreatedAt        time.Time       `json:"created_at"`
}

type ISP struct {
    ID              int            `json:"id"`
    UserID          sql.NullInt64  `json:"user_id"`
    Name            string         `json:"name"`
    ServerIP        string         `json:"server_ip"`
    HWID            string         `json:"hw_id"`
    Status          string         `json:"status"`
    PlanID          sql.NullInt64  `json:"plan_id"`
    CacheSizeGB     int            `json:"cache_size_gb"`
    BandwidthLimit  int            `json:"bandwidth_limit_mbps"`
    LastSeen        sql.NullTime   `json:"last_seen"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
}

type License struct {
    ID         int       `json:"id"`
    ISPID      int       `json:"isp_id"`
    LicenseKey string    `json:"license_key"`
    Token      string    `json:"token"`
    ExpiresAt  time.Time `json:"expires_at"`
    IsActive   bool      `json:"is_active"`
    Modules    []byte    `json:"modules"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

type Telemetry struct {
    ID              int       `json:"id"`
    ISPID           int       `json:"isp_id"`
    CacheHits       int64     `json:"cache_hits"`
    CacheMisses     int64     `json:"cache_misses"`
    BandwidthSaved  int64     `json:"bandwidth_saved_mb"`
    TotalRequests   int64     `json:"total_requests"`
    CacheSizeUsed   int       `json:"cache_size_used_mb"`
    CPUUsage        float64   `json:"cpu_usage"`
    MemoryUsage     float64   `json:"memory_usage"`
    CreatedAt       time.Time `json:"created_at"`
}

type Invoice struct {
    ID        int            `json:"id"`
    ISPID     int            `json:"isp_id"`
    Amount    float64        `json:"amount"`
    Status    string         `json:"status"`
    DueDate   time.Time      `json:"due_date"`
    PaidAt    sql.NullTime   `json:"paid_at"`
    CreatedAt time.Time      `json:"created_at"`
}

type SystemLog struct {
    ID        int       `json:"id"`
    Level     string    `json:"level"`
    Source    string    `json:"source"`
    Message   string    `json:"message"`
    Metadata  []byte    `json:"metadata"`
    CreatedAt time.Time `json:"created_at"`
}
