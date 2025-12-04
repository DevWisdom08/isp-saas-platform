package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"
    "isp-saas.com/platform/internal/middleware"
)

type ISPResponse struct {
    ID             int     `json:"id"`
    UserID         *int    `json:"user_id"`
    Name           string  `json:"name"`
    ServerIP       string  `json:"server_ip"`
    HWID           string  `json:"hw_id"`
    Status         string  `json:"status"`
    PlanID         *int    `json:"plan_id"`
    PlanName       *string `json:"plan_name,omitempty"`
    CacheSizeGB    int     `json:"cache_size_gb"`
    BandwidthLimit int     `json:"bandwidth_limit_mbps"`
    LastSeen       *string `json:"last_seen"`
    CreatedAt      string  `json:"created_at"`
}

type CreateISPRequest struct {
    Name           string `json:"name"`
    ServerIP       string `json:"server_ip"`
    HWID           string `json:"hw_id"`
    PlanID         *int   `json:"plan_id"`
    CacheSizeGB    int    `json:"cache_size_gb"`
    BandwidthLimit int    `json:"bandwidth_limit_mbps"`
}

type UpdateISPRequest struct {
    Name           string `json:"name,omitempty"`
    ServerIP       string `json:"server_ip,omitempty"`
    PlanID         *int   `json:"plan_id,omitempty"`
    CacheSizeGB    int    `json:"cache_size_gb,omitempty"`
    BandwidthLimit int    `json:"bandwidth_limit_mbps,omitempty"`
    Status         string `json:"status,omitempty"`
}

func (h *Handler) GetISPs(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    
    var rows interface{ Scan(...interface{}) error }
    var err error
    var isps []ISPResponse

    query := `
        SELECT i.id, i.user_id, i.name, i.server_ip, i.hw_id, i.status, i.plan_id, 
               p.name as plan_name, i.cache_size_gb, i.bandwidth_limit_mbps, i.last_seen, i.created_at 
        FROM isps i
        LEFT JOIN plans p ON i.plan_id = p.id
    `

    if claims.Role == "admin" || claims.Role == "distributor" {
        rowsResult, e := h.db.Query(query + " ORDER BY i.id")
        err = e
        if err == nil {
            defer rowsResult.Close()
            for rowsResult.Next() {
                var isp ISPResponse
                rowsResult.Scan(&isp.ID, &isp.UserID, &isp.Name, &isp.ServerIP, &isp.HWID, 
                    &isp.Status, &isp.PlanID, &isp.PlanName, &isp.CacheSizeGB, &isp.BandwidthLimit, 
                    &isp.LastSeen, &isp.CreatedAt)
                isps = append(isps, isp)
            }
        }
    } else {
        rowsResult, e := h.db.Query(query+" WHERE i.user_id = $1 ORDER BY i.id", claims.UserID)
        err = e
        if err == nil {
            defer rowsResult.Close()
            for rowsResult.Next() {
                var isp ISPResponse
                rowsResult.Scan(&isp.ID, &isp.UserID, &isp.Name, &isp.ServerIP, &isp.HWID, 
                    &isp.Status, &isp.PlanID, &isp.PlanName, &isp.CacheSizeGB, &isp.BandwidthLimit, 
                    &isp.LastSeen, &isp.CreatedAt)
                isps = append(isps, isp)
            }
        }
    }

    _ = rows
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: isps})
}

func (h *Handler) GetISP(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var isp ISPResponse
    err := h.db.QueryRow(`
        SELECT i.id, i.user_id, i.name, i.server_ip, i.hw_id, i.status, i.plan_id,
               p.name as plan_name, i.cache_size_gb, i.bandwidth_limit_mbps, i.last_seen, i.created_at
        FROM isps i
        LEFT JOIN plans p ON i.plan_id = p.id
        WHERE i.id = $1
    `, id).Scan(&isp.ID, &isp.UserID, &isp.Name, &isp.ServerIP, &isp.HWID, 
        &isp.Status, &isp.PlanID, &isp.PlanName, &isp.CacheSizeGB, &isp.BandwidthLimit, 
        &isp.LastSeen, &isp.CreatedAt)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "ISP not found"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: isp})
}

func (h *Handler) CreateISP(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    
    var req CreateISPRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Name == "" || req.ServerIP == "" || req.HWID == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Name, server_ip, and hw_id are required"})
        return
    }

    if req.CacheSizeGB == 0 {
        req.CacheSizeGB = 10
    }
    if req.BandwidthLimit == 0 {
        req.BandwidthLimit = 1000
    }

    var ispID int
    err := h.db.QueryRow(`
        INSERT INTO isps (user_id, name, server_ip, hw_id, plan_id, cache_size_gb, bandwidth_limit_mbps) 
        VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
    `, claims.UserID, req.Name, req.ServerIP, req.HWID, req.PlanID, req.CacheSizeGB, req.BandwidthLimit).Scan(&ispID)

    if err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Failed to create ISP. HWID may already exist."})
        return
    }

    h.logger.Info("ISP created", "isp_id", ispID, "name", req.Name, "by", claims.UserID)
    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "ISP created successfully",
        Data:    map[string]int{"id": ispID},
    })
}

func (h *Handler) UpdateISP(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var req UpdateISPRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    _, err := h.db.Exec(`
        UPDATE isps SET 
            name = COALESCE(NULLIF($1, ''), name),
            server_ip = COALESCE(NULLIF($2, ''), server_ip),
            cache_size_gb = CASE WHEN $3 > 0 THEN $3 ELSE cache_size_gb END,
            bandwidth_limit_mbps = CASE WHEN $4 > 0 THEN $4 ELSE bandwidth_limit_mbps END,
            updated_at = NOW()
        WHERE id = $5
    `, req.Name, req.ServerIP, req.CacheSizeGB, req.BandwidthLimit, id)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to update ISP"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "ISP updated successfully"})
}

func (h *Handler) SuspendISP(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    _, err := h.db.Exec("UPDATE isps SET status = 'suspended', updated_at = NOW() WHERE id = $1", id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to suspend ISP"})
        return
    }

    h.logger.Info("ISP suspended", "isp_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "ISP suspended successfully"})
}

func (h *Handler) ActivateISP(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    _, err := h.db.Exec("UPDATE isps SET status = 'active', updated_at = NOW() WHERE id = $1", id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to activate ISP"})
        return
    }

    h.logger.Info("ISP activated", "isp_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "ISP activated successfully"})
}

func (h *Handler) DeleteISP(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    _, err := h.db.Exec("DELETE FROM isps WHERE id = $1", id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to delete ISP"})
        return
    }

    h.logger.Info("ISP deleted", "isp_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "ISP deleted successfully"})
}
