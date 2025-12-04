package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "isp-saas.com/platform/internal/middleware"
)

type SystemLogResponse struct {
    ID        int             `json:"id"`
    Level     string          `json:"level"`
    Source    string          `json:"source"`
    Message   string          `json:"message"`
    Metadata  json.RawMessage `json:"metadata,omitempty"`
    CreatedAt string          `json:"created_at"`
}

type CreateLogRequest struct {
    Level    string                 `json:"level"`
    Source   string                 `json:"source"`
    Message  string                 `json:"message"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (h *Handler) GetSystemLogs(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    level := r.URL.Query().Get("level")
    source := r.URL.Query().Get("source")
    limitStr := r.URL.Query().Get("limit")
    limit := 100
    if limitStr != "" {
        if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
            limit = l
        }
    }

    query := `SELECT id, level, COALESCE(source, ''), message, metadata, created_at FROM system_logs WHERE 1=1`
    args := []interface{}{}
    argCount := 0

    if level != "" {
        argCount++
        query += " AND level = $" + strconv.Itoa(argCount)
        args = append(args, level)
    }

    if source != "" {
        argCount++
        query += " AND source = $" + strconv.Itoa(argCount)
        args = append(args, source)
    }

    argCount++
    query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argCount)
    args = append(args, limit)

    rows, err := h.db.Query(query, args...)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var logs []SystemLogResponse
    for rows.Next() {
        var log SystemLogResponse
        rows.Scan(&log.ID, &log.Level, &log.Source, &log.Message, &log.Metadata, &log.CreatedAt)
        logs = append(logs, log)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: logs})
}

func (h *Handler) CreateSystemLog(w http.ResponseWriter, r *http.Request) {
    var req CreateLogRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Level == "" {
        req.Level = "INFO"
    }

    validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true, "FATAL": true}
    if !validLevels[req.Level] {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid log level"})
        return
    }

    metadataJSON, _ := json.Marshal(req.Metadata)

    var logID int
    err := h.db.QueryRow(`
        INSERT INTO system_logs (level, source, message, metadata) VALUES ($1, $2, $3, $4) RETURNING id
    `, req.Level, req.Source, req.Message, metadataJSON).Scan(&logID)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to create log"})
        return
    }

    h.sendJSON(w, http.StatusCreated, Response{Success: true, Data: map[string]int{"id": logID}})
}

func (h *Handler) GetLogStats(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    rows, err := h.db.Query(`
        SELECT level, COUNT(*) as count 
        FROM system_logs 
        WHERE created_at > NOW() - INTERVAL '24 hours'
        GROUP BY level
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    stats := make(map[string]int)
    for rows.Next() {
        var level string
        var count int
        rows.Scan(&level, &count)
        stats[level] = count
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}

func (h *Handler) DeleteOldLogs(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    result, err := h.db.Exec("DELETE FROM system_logs WHERE created_at < NOW() - INTERVAL '30 days'")
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to delete logs"})
        return
    }

    rows, _ := result.RowsAffected()
    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "Old logs deleted",
        Data:    map[string]int64{"deleted": rows},
    })
}
