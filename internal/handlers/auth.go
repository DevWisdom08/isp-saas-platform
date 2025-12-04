package handlers

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "isp-saas.com/platform/internal/middleware"
)

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type RegisterRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    FullName string `json:"full_name"`
    Role     string `json:"role"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Email == "" || req.Password == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email and password are required"})
        return
    }

    var userID int
    var email, passwordHash, role, fullName string
    var isActive bool
    err := h.db.QueryRow(
        "SELECT id, email, password_hash, role, COALESCE(full_name, ''), is_active FROM users WHERE email = $1",
        req.Email,
    ).Scan(&userID, &email, &passwordHash, &role, &fullName, &isActive)

    if err != nil {
        h.logger.Warn("Login failed - user not found", "email", req.Email)
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "Invalid credentials"})
        return
    }

    if !isActive {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "Account is disabled"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
        h.logger.Warn("Login failed - invalid password", "email", req.Email)
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "Invalid credentials"})
        return
    }

    token, err := generateJWT(userID, email, role)
    if err != nil {
        h.logger.Error("Failed to generate JWT", "error", err)
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to generate token"})
        return
    }

    h.logger.Info("User logged in", "user_id", userID, "email", email)

    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "Login successful",
        Data: map[string]interface{}{
            "token": token,
            "user": map[string]interface{}{
                "id":        userID,
                "email":     email,
                "role":      role,
                "full_name": fullName,
            },
        },
    })
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Email == "" || req.Password == "" {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email and password are required"})
        return
    }

    if len(req.Password) < 8 {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Password must be at least 8 characters"})
        return
    }

    if req.Role == "" {
        req.Role = "isp"
    }

    validRoles := map[string]bool{"admin": true, "distributor": true, "isp": true}
    if !validRoles[req.Role] {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid role"})
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to process password"})
        return
    }

    var userID int
    err = h.db.QueryRow(
        "INSERT INTO users (email, password_hash, role, full_name) VALUES ($1, $2, $3, $4) RETURNING id",
        req.Email, string(hashedPassword), req.Role, req.FullName,
    ).Scan(&userID)

    if err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email already exists"})
        return
    }

    token, _ := generateJWT(userID, req.Email, req.Role)

    h.logger.Info("User registered", "user_id", userID, "email", req.Email, "role", req.Role)

    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "User registered successfully",
        Data: map[string]interface{}{
            "token":   token,
            "user_id": userID,
        },
    })
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims == nil {
        h.sendJSON(w, http.StatusUnauthorized, Response{Success: false, Error: "Invalid token"})
        return
    }

    token, err := generateJWT(claims.UserID, claims.Email, claims.Role)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to refresh token"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Data:    map[string]string{"token": token},
    })
}

func generateJWT(userID int, email, role string) (string, error) {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "your-super-secret-key-change-in-production"
    }

    claims := middleware.Claims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

func generateRandomKey(length int) string {
    bytes := make([]byte, length)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}
