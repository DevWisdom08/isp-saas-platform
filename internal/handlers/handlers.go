package handlers

import (
    "encoding/json"
    "net/http"
    "time"

    "isp-saas.com/platform/pkg/database"
    "isp-saas.com/platform/pkg/logger"
)

type Handler struct {
    db     *database.DB
    logger *logger.Logger
}

func New(db *database.DB, l *logger.Logger) *Handler {
    return &Handler{db: db, logger: l}
}

type Response struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, resp Response) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(resp)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
    var dbStatus string
    if err := h.db.Ping(); err != nil {
        dbStatus = "disconnected"
    } else {
        dbStatus = "connected"
    }

    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "ISP SaaS Platform API is running",
        Data: map[string]interface{}{
            "version":   "1.0.0",
            "timestamp": time.Now().Format(time.RFC3339),
            "database":  dbStatus,
        },
    })
}
