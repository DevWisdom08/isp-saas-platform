package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"
    "isp-saas.com/platform/internal/middleware"
)

type TopSiteResponse struct {
    ID             int    `json:"id"`
    Domain         string `json:"domain"`
    Hits           int64  `json:"hits"`
    BandwidthSaved int64  `json:"bandwidth_saved_mb"`
    Category       string `json:"category,omitempty"`
    Icon           string `json:"icon,omitempty"`
    LastAccessed   string `json:"last_accessed"`
}

type AppCategoryResponse struct {
    ID      int      `json:"id"`
    Name    string   `json:"name"`
    Icon    string   `json:"icon"`
    Domains []string `json:"domains"`
}

type ReportSiteRequest struct {
    ISPID          int    `json:"isp_id"`
    Domain         string `json:"domain"`
    Hits           int64  `json:"hits"`
    BandwidthSaved int64  `json:"bandwidth_saved_mb"`
}

// GetTopSites returns top 10 cached sites globally or per ISP
func (h *Handler) GetTopSites(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    ispID := r.URL.Query().Get("isp_id")
    limit := r.URL.Query().Get("limit")
    if limit == "" {
        limit = "10"
    }

    var query string
    var args []interface{}

    if claims.Role == "admin" {
        if ispID != "" {
            query = `
                SELECT cs.id, cs.domain, cs.hits, cs.bandwidth_saved_mb, 
                       COALESCE(ac.name, 'Other') as category,
                       COALESCE(ac.icon, '🌐') as icon,
                       cs.last_accessed
                FROM cached_sites cs
                LEFT JOIN app_categories ac ON cs.domain LIKE ANY(ac.domains)
                WHERE cs.isp_id = $1
                ORDER BY cs.hits DESC
                LIMIT $2
            `
            args = []interface{}{ispID, limit}
        } else {
            query = `
                SELECT cs.id, cs.domain, SUM(cs.hits) as hits, SUM(cs.bandwidth_saved_mb) as bandwidth_saved_mb,
                       COALESCE(ac.name, 'Other') as category,
                       COALESCE(ac.icon, '🌐') as icon,
                       MAX(cs.last_accessed) as last_accessed
                FROM cached_sites cs
                LEFT JOIN app_categories ac ON cs.domain LIKE ANY(ac.domains)
                GROUP BY cs.domain, ac.name, ac.icon
                ORDER BY hits DESC
                LIMIT $1
            `
            args = []interface{}{limit}
        }
    } else if claims.Role == "distributor" {
        query = `
            SELECT cs.id, cs.domain, SUM(cs.hits) as hits, SUM(cs.bandwidth_saved_mb) as bandwidth_saved_mb,
                   COALESCE(ac.name, 'Other') as category,
                   COALESCE(ac.icon, '🌐') as icon,
                   MAX(cs.last_accessed) as last_accessed
            FROM cached_sites cs
            LEFT JOIN app_categories ac ON cs.domain LIKE ANY(ac.domains)
            JOIN isps i ON cs.isp_id = i.id
            WHERE i.user_id = $1
            GROUP BY cs.domain, ac.name, ac.icon
            ORDER BY hits DESC
            LIMIT $2
        `
        args = []interface{}{claims.UserID, limit}
    } else {
        query = `
            SELECT cs.id, cs.domain, cs.hits, cs.bandwidth_saved_mb,
                   COALESCE(ac.name, 'Other') as category,
                   COALESCE(ac.icon, '🌐') as icon,
                   cs.last_accessed
            FROM cached_sites cs
            LEFT JOIN app_categories ac ON cs.domain LIKE ANY(ac.domains)
            JOIN isps i ON cs.isp_id = i.id
            WHERE i.user_id = $1
            ORDER BY cs.hits DESC
            LIMIT $2
        `
        args = []interface{}{claims.UserID, limit}
    }

    rows, err := h.db.Query(query, args...)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error: " + err.Error()})
        return
    }
    defer rows.Close()

    var sites []TopSiteResponse
    for rows.Next() {
        var s TopSiteResponse
        rows.Scan(&s.ID, &s.Domain, &s.Hits, &s.BandwidthSaved, &s.Category, &s.Icon, &s.LastAccessed)
        sites = append(sites, s)
    }

    if sites == nil {
        sites = []TopSiteResponse{}
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: sites})
}

// GetTopApps returns top apps by category
func (h *Handler) GetTopApps(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)

    var query string
    var args []interface{}

    if claims.Role == "admin" {
        query = `
            SELECT ac.name, ac.icon, COALESCE(SUM(cs.hits), 0) as total_hits, 
                   COALESCE(SUM(cs.bandwidth_saved_mb), 0) as total_bandwidth
            FROM app_categories ac
            LEFT JOIN cached_sites cs ON cs.domain LIKE ANY(ac.domains)
            GROUP BY ac.id, ac.name, ac.icon
            ORDER BY total_hits DESC
            LIMIT 10
        `
    } else if claims.Role == "distributor" {
        query = `
            SELECT ac.name, ac.icon, COALESCE(SUM(cs.hits), 0) as total_hits,
                   COALESCE(SUM(cs.bandwidth_saved_mb), 0) as total_bandwidth
            FROM app_categories ac
            LEFT JOIN cached_sites cs ON cs.domain LIKE ANY(ac.domains)
            LEFT JOIN isps i ON cs.isp_id = i.id
            WHERE i.user_id = $1 OR cs.id IS NULL
            GROUP BY ac.id, ac.name, ac.icon
            ORDER BY total_hits DESC
            LIMIT 10
        `
        args = []interface{}{claims.UserID}
    } else {
        query = `
            SELECT ac.name, ac.icon, COALESCE(SUM(cs.hits), 0) as total_hits,
                   COALESCE(SUM(cs.bandwidth_saved_mb), 0) as total_bandwidth
            FROM app_categories ac
            LEFT JOIN cached_sites cs ON cs.domain LIKE ANY(ac.domains)
            LEFT JOIN isps i ON cs.isp_id = i.id
            WHERE i.user_id = $1 OR cs.id IS NULL
            GROUP BY ac.id, ac.name, ac.icon
            ORDER BY total_hits DESC
            LIMIT 10
        `
        args = []interface{}{claims.UserID}
    }

    rows, err := h.db.Query(query, args...)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    type AppStats struct {
        Name           string `json:"name"`
        Icon           string `json:"icon"`
        TotalHits      int64  `json:"total_hits"`
        TotalBandwidth int64  `json:"total_bandwidth_mb"`
    }

    var apps []AppStats
    for rows.Next() {
        var a AppStats
        rows.Scan(&a.Name, &a.Icon, &a.TotalHits, &a.TotalBandwidth)
        apps = append(apps, a)
    }

    if apps == nil {
        apps = []AppStats{}
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: apps})
}

// ReportCachedSite receives cache hit reports from agents
func (h *Handler) ReportCachedSite(w http.ResponseWriter, r *http.Request) {
    var req ReportSiteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.ISPID == 0 || req.Domain == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "ISP ID and domain are required"})
        return
    }

    _, err := h.db.Exec(`
        INSERT INTO cached_sites (isp_id, domain, hits, bandwidth_saved_mb, last_accessed, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (isp_id, domain)
        DO UPDATE SET 
            hits = cached_sites.hits + EXCLUDED.hits,
            bandwidth_saved_mb = cached_sites.bandwidth_saved_mb + EXCLUDED.bandwidth_saved_mb,
            last_accessed = NOW(),
            updated_at = NOW()
    `, req.ISPID, req.Domain, req.Hits, req.BandwidthSaved)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to record site"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "Site recorded"})
}

// GetISPDashboard returns dashboard data specific to an ISP
func (h *Handler) GetISPDashboard(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    ispID := vars["id"]

    // Get ISP info
    var ispName, status string
    var cacheSize, bandwidth int
    err := h.db.QueryRow(`
        SELECT name, status, cache_size_gb, bandwidth_limit_mbps 
        FROM isps WHERE id = $1
    `, ispID).Scan(&ispName, &status, &cacheSize, &bandwidth)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "ISP not found"})
        return
    }

    // Get telemetry stats (last 24h)
    var totalHits, totalMisses, bandwidthSaved int64
    var avgCPU, avgMemory float64
    h.db.QueryRow(`
        SELECT COALESCE(SUM(cache_hits), 0), COALESCE(SUM(cache_misses), 0),
               COALESCE(SUM(bandwidth_saved_mb), 0), COALESCE(AVG(cpu_usage), 0),
               COALESCE(AVG(memory_usage), 0)
        FROM telemetry 
        WHERE isp_id = $1 AND created_at > NOW() - INTERVAL '24 hours'
    `, ispID).Scan(&totalHits, &totalMisses, &bandwidthSaved, &avgCPU, &avgMemory)

    // Get top 5 sites for this ISP
    rows, _ := h.db.Query(`
        SELECT domain, hits, bandwidth_saved_mb 
        FROM cached_sites WHERE isp_id = $1 
        ORDER BY hits DESC LIMIT 5
    `, ispID)
    defer rows.Close()

    var topSites []map[string]interface{}
    for rows.Next() {
        var domain string
        var hits, bw int64
        rows.Scan(&domain, &hits, &bw)
        topSites = append(topSites, map[string]interface{}{
            "domain": domain, "hits": hits, "bandwidth_saved_mb": bw,
        })
    }

    // Get license info
    var licenseKey string
    var expiresAt string
    var licenseActive bool
    h.db.QueryRow(`
        SELECT license_key, expires_at, is_active 
        FROM licenses WHERE isp_id = $1 AND is_active = true
        ORDER BY expires_at DESC LIMIT 1
    `, ispID).Scan(&licenseKey, &expiresAt, &licenseActive)

    // Calculate hit rate
    var hitRate float64
    if totalHits+totalMisses > 0 {
        hitRate = float64(totalHits) / float64(totalHits+totalMisses) * 100
    }

    dashboard := map[string]interface{}{
        "isp": map[string]interface{}{
            "name":       ispName,
            "status":     status,
            "cache_size": cacheSize,
            "bandwidth":  bandwidth,
        },
        "stats": map[string]interface{}{
            "cache_hits":      totalHits,
            "cache_misses":    totalMisses,
            "hit_rate":        hitRate,
            "bandwidth_saved": bandwidthSaved,
            "avg_cpu":         avgCPU,
            "avg_memory":      avgMemory,
        },
        "top_sites": topSites,
        "license": map[string]interface{}{
            "key":        licenseKey,
            "expires_at": expiresAt,
            "is_active":  licenseActive,
        },
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: dashboard})
}

// GetAppCategories returns all app categories
func (h *Handler) GetAppCategories(w http.ResponseWriter, r *http.Request) {
    rows, err := h.db.Query("SELECT id, name, icon, domains FROM app_categories ORDER BY name")
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var categories []AppCategoryResponse
    for rows.Next() {
        var c AppCategoryResponse
        var domains []byte
        rows.Scan(&c.ID, &c.Name, &c.Icon, &domains)
        json.Unmarshal(domains, &c.Domains)
        categories = append(categories, c)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: categories})
}
