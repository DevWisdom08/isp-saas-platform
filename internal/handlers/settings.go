package handlers

import (
    "encoding/json"
    "net/http"

    "isp-saas.com/platform/internal/middleware"
)

type SettingResponse struct {
    ID          int    `json:"id"`
    Key         string `json:"key"`
    Value       string `json:"value"`
    Description string `json:"description"`
    UpdatedAt   string `json:"updated_at"`
}

type UpdateSettingRequest struct {
    Value string `json:"value"`
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    rows, err := h.db.Query(`
        SELECT id, key, COALESCE(value, ''), COALESCE(description, ''), updated_at
        FROM settings ORDER BY key
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var settings []SettingResponse
    for rows.Next() {
        var s SettingResponse
        rows.Scan(&s.ID, &s.Key, &s.Value, &s.Description, &s.UpdatedAt)
        settings = append(settings, s)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: settings})
}

func (h *Handler) GetSetting(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if key == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Key parameter required"})
        return
    }

    var value string
    err := h.db.QueryRow("SELECT COALESCE(value, '') FROM settings WHERE key = $1", key).Scan(&value)
    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "Setting not found"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"key": key, "value": value}})
}

func (h *Handler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    key := r.URL.Query().Get("key")
    if key == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Key parameter required"})
        return
    }

    var req UpdateSettingRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    result, err := h.db.Exec(`
        UPDATE settings SET value = $1, updated_by = $2, updated_at = NOW() WHERE key = $3
    `, req.Value, claims.UserID, key)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to update setting"})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "Setting not found"})
        return
    }

    h.logger.Info("Setting updated", "key", key, "value", req.Value, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "Setting updated successfully"})
}

func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" && claims.Role != "distributor" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Access denied"})
        return
    }

    stats := make(map[string]interface{})

    // Total ISPs
    var totalISPs, activeISPs, suspendedISPs int
    h.db.QueryRow("SELECT COUNT(*) FROM isps").Scan(&totalISPs)
    h.db.QueryRow("SELECT COUNT(*) FROM isps WHERE status = 'active'").Scan(&activeISPs)
    h.db.QueryRow("SELECT COUNT(*) FROM isps WHERE status = 'suspended'").Scan(&suspendedISPs)

    // Total Users
    var totalUsers int
    h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)

    // Revenue stats
    var totalRevenue, pendingRevenue float64
    h.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM invoices WHERE status = 'paid'").Scan(&totalRevenue)
    h.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM invoices WHERE status = 'pending'").Scan(&pendingRevenue)

    // Telemetry stats (last 24h)
    var totalHits, totalMisses, bandwidthSaved int64
    h.db.QueryRow(`
        SELECT COALESCE(SUM(cache_hits), 0), COALESCE(SUM(cache_misses), 0), COALESCE(SUM(bandwidth_saved_mb), 0)
        FROM telemetry WHERE created_at > NOW() - INTERVAL '24 hours'
    `).Scan(&totalHits, &totalMisses, &bandwidthSaved)

    var hitRate float64
    if totalHits+totalMisses > 0 {
        hitRate = float64(totalHits) / float64(totalHits+totalMisses) * 100
    }

    // Active licenses
    var activeLicenses, expiringSoon int
    h.db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = true AND expires_at > NOW()").Scan(&activeLicenses)
    h.db.QueryRow("SELECT COUNT(*) FROM licenses WHERE is_active = true AND expires_at BETWEEN NOW() AND NOW() + INTERVAL '7 days'").Scan(&expiringSoon)

    stats["isps"] = map[string]int{
        "total":     totalISPs,
        "active":    activeISPs,
        "suspended": suspendedISPs,
    }
    stats["users"] = map[string]int{
        "total": totalUsers,
    }
    stats["revenue"] = map[string]float64{
        "total":   totalRevenue,
        "pending": pendingRevenue,
    }
    stats["cache"] = map[string]interface{}{
        "hits":            totalHits,
        "misses":          totalMisses,
        "hit_rate":        hitRate,
        "bandwidth_saved": bandwidthSaved,
    }
    stats["licenses"] = map[string]int{
        "active":        activeLicenses,
        "expiring_soon": expiringSoon,
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}
