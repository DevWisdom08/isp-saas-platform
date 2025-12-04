package handlers

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/gorilla/mux"
    "isp-saas.com/platform/internal/middleware"
)

type LicenseResponse struct {
    ID         int      `json:"id"`
    ISPID      int      `json:"isp_id"`
    ISPName    string   `json:"isp_name,omitempty"`
    LicenseKey string   `json:"license_key"`
    ExpiresAt  string   `json:"expires_at"`
    IsActive   bool     `json:"is_active"`
    Modules    []string `json:"modules"`
    CreatedAt  string   `json:"created_at"`
}

type CreateLicenseRequest struct {
    ISPID      int      `json:"isp_id"`
    DaysValid  int      `json:"days_valid"`
    Modules    []string `json:"modules"`
}

type ValidateLicenseRequest struct {
    LicenseKey string `json:"license_key"`
    HWID       string `json:"hw_id"`
}

func (h *Handler) GetLicenses(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" && claims.Role != "distributor" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Access denied"})
        return
    }

    rows, err := h.db.Query(`
        SELECT l.id, l.isp_id, i.name as isp_name, l.license_key, l.expires_at, l.is_active, l.modules, l.created_at
        FROM licenses l
        JOIN isps i ON l.isp_id = i.id
        ORDER BY l.created_at DESC
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var licenses []LicenseResponse
    for rows.Next() {
        var l LicenseResponse
        var modulesJSON []byte
        if err := rows.Scan(&l.ID, &l.ISPID, &l.ISPName, &l.LicenseKey, &l.ExpiresAt, &l.IsActive, &modulesJSON, &l.CreatedAt); err != nil {
            continue
        }
        json.Unmarshal(modulesJSON, &l.Modules)
        licenses = append(licenses, l)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: licenses})
}

func (h *Handler) GetLicense(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var l LicenseResponse
    var modulesJSON []byte
    err := h.db.QueryRow(`
        SELECT l.id, l.isp_id, i.name as isp_name, l.license_key, l.expires_at, l.is_active, l.modules, l.created_at
        FROM licenses l
        JOIN isps i ON l.isp_id = i.id
        WHERE l.id = $1
    `, id).Scan(&l.ID, &l.ISPID, &l.ISPName, &l.LicenseKey, &l.ExpiresAt, &l.IsActive, &modulesJSON, &l.CreatedAt)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "License not found"})
        return
    }
    json.Unmarshal(modulesJSON, &l.Modules)

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: l})
}

func (h *Handler) CreateLicense(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    var req CreateLicenseRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.ISPID == 0 {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "ISP ID is required"})
        return
    }

    if req.DaysValid == 0 {
        req.DaysValid = 30
    }

    if len(req.Modules) == 0 {
        req.Modules = []string{"cache", "https", "monitoring"}
    }

    licenseKey := generateLicenseKey()
    expiresAt := time.Now().AddDate(0, 0, req.DaysValid)
    token := generateLicenseToken(req.ISPID, licenseKey, expiresAt)
    modulesJSON, _ := json.Marshal(req.Modules)

    var licenseID int
    err := h.db.QueryRow(`
        INSERT INTO licenses (isp_id, license_key, token, expires_at, modules)
        VALUES ($1, $2, $3, $4, $5) RETURNING id
    `, req.ISPID, licenseKey, token, expiresAt, modulesJSON).Scan(&licenseID)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to create license"})
        return
    }

    h.logger.Info("License created", "license_id", licenseID, "isp_id", req.ISPID, "by", claims.UserID)
    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "License created successfully",
        Data: map[string]interface{}{
            "id":          licenseID,
            "license_key": licenseKey,
            "expires_at":  expiresAt.Format(time.RFC3339),
        },
    })
}

func (h *Handler) ValidateLicense(w http.ResponseWriter, r *http.Request) {
    var req ValidateLicenseRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    var ispID int
    var expiresAt time.Time
    var isActive bool
    var modulesJSON []byte
    var ispStatus string

    err := h.db.QueryRow(`
        SELECT l.isp_id, l.expires_at, l.is_active, l.modules, i.status
        FROM licenses l
        JOIN isps i ON l.isp_id = i.id
        WHERE l.license_key = $1 AND i.hw_id = $2
    `, req.LicenseKey, req.HWID).Scan(&ispID, &expiresAt, &isActive, &modulesJSON, &ispStatus)

    if err != nil {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "Invalid license or hardware ID"})
        return
    }

    if !isActive {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "License is deactivated"})
        return
    }

    if time.Now().After(expiresAt) {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "License has expired"})
        return
    }

    if ispStatus == "suspended" {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "ISP account is suspended"})
        return
    }

    h.db.Exec("UPDATE isps SET last_seen = NOW() WHERE id = $1", ispID)

    var modules []string
    json.Unmarshal(modulesJSON, &modules)

    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "License is valid",
        Data: map[string]interface{}{
            "isp_id":     ispID,
            "expires_at": expiresAt.Format(time.RFC3339),
            "modules":    modules,
            "status":     "active",
        },
    })
}

func (h *Handler) RevokeLicense(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    _, err := h.db.Exec("UPDATE licenses SET is_active = false, updated_at = NOW() WHERE id = $1", id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to revoke license"})
        return
    }

    h.logger.Info("License revoked", "license_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "License revoked successfully"})
}

func generateLicenseKey() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return "ISP-" + hex.EncodeToString(bytes)[:24]
}

func generateLicenseToken(ispID int, licenseKey string, expiresAt time.Time) string {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "your-super-secret-key-change-in-production"
    }

    claims := jwt.MapClaims{
        "isp_id":      ispID,
        "license_key": licenseKey,
        "exp":         expiresAt.Unix(),
        "iat":         time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, _ := token.SignedString([]byte(secret))
    return signedToken
}
