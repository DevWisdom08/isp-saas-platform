package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"
    "isp-saas.com/platform/internal/middleware"
)

type TelemetryData struct {
    ISPID          int     `json:"isp_id"`
    CacheHits      int64   `json:"cache_hits"`
    CacheMisses    int64   `json:"cache_misses"`
    BandwidthSaved int64   `json:"bandwidth_saved_mb"`
    TotalRequests  int64   `json:"total_requests"`
    CacheSizeUsed  int     `json:"cache_size_used_mb"`
    CPUUsage       float64 `json:"cpu_usage"`
    MemoryUsage    float64 `json:"memory_usage"`
}

type TelemetryResponse struct {
    ID             int     `json:"id"`
    ISPID          int     `json:"isp_id"`
    CacheHits      int64   `json:"cache_hits"`
    CacheMisses    int64   `json:"cache_misses"`
    HitRate        float64 `json:"hit_rate"`
    BandwidthSaved int64   `json:"bandwidth_saved_mb"`
    TotalRequests  int64   `json:"total_requests"`
    CacheSizeUsed  int     `json:"cache_size_used_mb"`
    CPUUsage       float64 `json:"cpu_usage"`
    MemoryUsage    float64 `json:"memory_usage"`
    CreatedAt      string  `json:"created_at"`
}

func (h *Handler) SubmitTelemetry(w http.ResponseWriter, r *http.Request) {
    var data TelemetryData
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if data.ISPID == 0 {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "ISP ID is required"})
        return
    }

    var ispStatus string
    err := h.db.QueryRow("SELECT status FROM isps WHERE id = $1", data.ISPID).Scan(&ispStatus)
    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "ISP not found"})
        return
    }

    _, err = h.db.Exec(`
        INSERT INTO telemetry (isp_id, cache_hits, cache_misses, bandwidth_saved_mb, total_requests, cache_size_used_mb, cpu_usage, memory_usage)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, data.ISPID, data.CacheHits, data.CacheMisses, data.BandwidthSaved, data.TotalRequests, data.CacheSizeUsed, data.CPUUsage, data.MemoryUsage)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to save telemetry"})
        return
    }

    h.db.Exec("UPDATE isps SET last_seen = NOW() WHERE id = $1", data.ISPID)

    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "Telemetry received",
        Data: map[string]interface{}{
            "status": ispStatus,
        },
    })
}

func (h *Handler) GetTelemetryStats(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    
    ispID := r.URL.Query().Get("isp_id")
    limit := r.URL.Query().Get("limit")
    if limit == "" {
        limit = "100"
    }

    var query string
    var args []interface{}

    if claims.Role == "admin" || claims.Role == "distributor" {
        if ispID != "" {
            query = `
                SELECT id, isp_id, cache_hits, cache_misses, bandwidth_saved_mb, total_requests, 
                       cache_size_used_mb, COALESCE(cpu_usage, 0), COALESCE(memory_usage, 0), created_at
                FROM telemetry WHERE isp_id = $1
                ORDER BY created_at DESC LIMIT $2
            `
            args = []interface{}{ispID, limit}
        } else {
            query = `
                SELECT id, isp_id, cache_hits, cache_misses, bandwidth_saved_mb, total_requests,
                       cache_size_used_mb, COALESCE(cpu_usage, 0), COALESCE(memory_usage, 0), created_at
                FROM telemetry
                ORDER BY created_at DESC LIMIT $1
            `
            args = []interface{}{limit}
        }
    } else {
        query = `
            SELECT t.id, t.isp_id, t.cache_hits, t.cache_misses, t.bandwidth_saved_mb, t.total_requests,
                   t.cache_size_used_mb, COALESCE(t.cpu_usage, 0), COALESCE(t.memory_usage, 0), t.created_at
            FROM telemetry t
            JOIN isps i ON t.isp_id = i.id
            WHERE i.user_id = $1
            ORDER BY t.created_at DESC LIMIT $2
        `
        args = []interface{}{claims.UserID, limit}
    }

    rows, err := h.db.Query(query, args...)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var telemetry []TelemetryResponse
    for rows.Next() {
        var t TelemetryResponse
        rows.Scan(&t.ID, &t.ISPID, &t.CacheHits, &t.CacheMisses, &t.BandwidthSaved, 
            &t.TotalRequests, &t.CacheSizeUsed, &t.CPUUsage, &t.MemoryUsage, &t.CreatedAt)
        
        if t.CacheHits+t.CacheMisses > 0 {
            t.HitRate = float64(t.CacheHits) / float64(t.CacheHits+t.CacheMisses) * 100
        }
        telemetry = append(telemetry, t)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: telemetry})
}

func (h *Handler) GetISPTelemetry(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    ispID := vars["id"]

    row := h.db.QueryRow(`
        SELECT 
            COALESCE(SUM(cache_hits), 0) as total_hits,
            COALESCE(SUM(cache_misses), 0) as total_misses,
            COALESCE(SUM(bandwidth_saved_mb), 0) as total_bandwidth_saved,
            COALESCE(SUM(total_requests), 0) as total_requests,
            COALESCE(AVG(cpu_usage), 0) as avg_cpu,
            COALESCE(AVG(memory_usage), 0) as avg_memory,
            COUNT(*) as data_points
        FROM telemetry 
        WHERE isp_id = $1 AND created_at > NOW() - INTERVAL '24 hours'
    `, ispID)

    var stats struct {
        TotalHits          int64   `json:"total_hits"`
        TotalMisses        int64   `json:"total_misses"`
        TotalBandwidthSaved int64  `json:"total_bandwidth_saved_mb"`
        TotalRequests      int64   `json:"total_requests"`
        AvgCPU             float64 `json:"avg_cpu_usage"`
        AvgMemory          float64 `json:"avg_memory_usage"`
        DataPoints         int     `json:"data_points"`
        HitRate            float64 `json:"hit_rate"`
    }

    err := row.Scan(&stats.TotalHits, &stats.TotalMisses, &stats.TotalBandwidthSaved, 
        &stats.TotalRequests, &stats.AvgCPU, &stats.AvgMemory, &stats.DataPoints)
    
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }

    if stats.TotalHits+stats.TotalMisses > 0 {
        stats.HitRate = float64(stats.TotalHits) / float64(stats.TotalHits+stats.TotalMisses) * 100
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}
