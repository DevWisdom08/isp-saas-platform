package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"
    "golang.org/x/crypto/bcrypt"
    "isp-saas.com/platform/internal/middleware"
)

type DistributorResponse struct {
    ID           int     `json:"id"`
    Email        string  `json:"email"`
    FullName     string  `json:"full_name"`
    CompanyName  string  `json:"company_name"`
    Commission   float64 `json:"commission_percent"`
    TotalISPs    int     `json:"total_isps"`
    TotalRevenue float64 `json:"total_revenue"`
    IsActive     bool    `json:"is_active"`
    CreatedAt    string  `json:"created_at"`
}

type CreateDistributorRequest struct {
    Email       string  `json:"email"`
    Password    string  `json:"password"`
    FullName    string  `json:"full_name"`
    CompanyName string  `json:"company_name"`
    Commission  float64 `json:"commission_percent"`
}

func (h *Handler) GetDistributors(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    rows, err := h.db.Query(`
        SELECT u.id, u.email, COALESCE(u.full_name, '') as full_name, 
               COALESCE(d.company_name, '') as company_name,
               COALESCE(d.commission_percent, 0) as commission,
               u.is_active, u.created_at,
               (SELECT COUNT(*) FROM isps WHERE user_id = u.id) as total_isps
        FROM users u
        LEFT JOIN distributors d ON u.id = d.user_id
        WHERE u.role = 'distributor'
        ORDER BY u.id
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var distributors []DistributorResponse
    for rows.Next() {
        var d DistributorResponse
        rows.Scan(&d.ID, &d.Email, &d.FullName, &d.CompanyName, &d.Commission, &d.IsActive, &d.CreatedAt, &d.TotalISPs)
        distributors = append(distributors, d)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: distributors})
}

func (h *Handler) GetDistributor(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" && claims.Role != "distributor" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Access denied"})
        return
    }

    var d DistributorResponse
    err := h.db.QueryRow(`
        SELECT u.id, u.email, COALESCE(u.full_name, '') as full_name,
               COALESCE(d.company_name, '') as company_name,
               COALESCE(d.commission_percent, 0) as commission,
               u.is_active, u.created_at,
               (SELECT COUNT(*) FROM isps WHERE user_id = u.id) as total_isps
        FROM users u
        LEFT JOIN distributors d ON u.id = d.user_id
        WHERE u.id = $1 AND u.role = 'distributor'
    `, id).Scan(&d.ID, &d.Email, &d.FullName, &d.CompanyName, &d.Commission, &d.IsActive, &d.CreatedAt, &d.TotalISPs)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "Distributor not found"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: d})
}

func (h *Handler) CreateDistributor(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    var req CreateDistributorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Email == "" || req.Password == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email and password are required"})
        return
    }

    if req.Commission < 0 || req.Commission > 100 {
        req.Commission = 10 // Default 10%
    }

    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

    tx, err := h.db.Begin()
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }

    var userID int
    err = tx.QueryRow(`
        INSERT INTO users (email, password_hash, role, full_name) 
        VALUES ($1, $2, 'distributor', $3) RETURNING id
    `, req.Email, string(hashedPassword), req.FullName).Scan(&userID)

    if err != nil {
        tx.Rollback()
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email already exists"})
        return
    }

    _, err = tx.Exec(`
        INSERT INTO distributors (user_id, company_name, commission_percent) VALUES ($1, $2, $3)
    `, userID, req.CompanyName, req.Commission)

    if err != nil {
        tx.Rollback()
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to create distributor profile"})
        return
    }

    tx.Commit()

    h.logger.Info("Distributor created", "user_id", userID, "email", req.Email, "by", claims.UserID)
    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "Distributor created successfully",
        Data:    map[string]int{"id": userID},
    })
}

func (h *Handler) UpdateDistributor(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    var req CreateDistributorRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.FullName != "" {
        h.db.Exec("UPDATE users SET full_name = $1, updated_at = NOW() WHERE id = $2", req.FullName, id)
    }

    if req.CompanyName != "" || req.Commission > 0 {
        h.db.Exec(`
            UPDATE distributors SET 
                company_name = COALESCE(NULLIF($1, ''), company_name),
                commission_percent = CASE WHEN $2 > 0 THEN $2 ELSE commission_percent END
            WHERE user_id = $3
        `, req.CompanyName, req.Commission, id)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "Distributor updated successfully"})
}

func (h *Handler) GetDistributorISPs(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    rows, err := h.db.Query(`
        SELECT id, name, server_ip, hw_id, status, cache_size_gb, bandwidth_limit_mbps, created_at
        FROM isps WHERE user_id = $1 ORDER BY id
    `, id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var isps []ISPResponse
    for rows.Next() {
        var isp ISPResponse
        rows.Scan(&isp.ID, &isp.Name, &isp.ServerIP, &isp.HWID, &isp.Status, &isp.CacheSizeGB, &isp.BandwidthLimit, &isp.CreatedAt)
        isps = append(isps, isp)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: isps})
}
