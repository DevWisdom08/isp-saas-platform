package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "golang.org/x/crypto/bcrypt"
    "isp-saas.com/platform/internal/middleware"
)

type UserResponse struct {
    ID        int    `json:"id"`
    Email     string `json:"email"`
    Role      string `json:"role"`
    FullName  string `json:"full_name"`
    IsActive  bool   `json:"is_active"`
    CreatedAt string `json:"created_at"`
}

type UpdateUserRequest struct {
    Email    string `json:"email,omitempty"`
    FullName string `json:"full_name,omitempty"`
    Role     string `json:"role,omitempty"`
    Password string `json:"password,omitempty"`
    IsActive *bool  `json:"is_active,omitempty"`
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    rows, err := h.db.Query(`
        SELECT id, email, role, COALESCE(full_name, '') as full_name, is_active, created_at 
        FROM users ORDER BY id
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var users []UserResponse
    for rows.Next() {
        var u UserResponse
        if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.IsActive, &u.CreatedAt); err != nil {
            continue
        }
        users = append(users, u)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: users})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    userID, _ := strconv.Atoi(id)
    
    if claims.Role != "admin" && claims.UserID != userID {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Access denied"})
        return
    }

    var u UserResponse
    err := h.db.QueryRow(`
        SELECT id, email, role, COALESCE(full_name, '') as full_name, is_active, created_at 
        FROM users WHERE id = $1
    `, id).Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.IsActive, &u.CreatedAt)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "User not found"})
        return
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: u})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    userID, _ := strconv.Atoi(id)
    
    if claims.Role != "admin" && claims.UserID != userID {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Access denied"})
        return
    }

    var req UpdateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.Email != "" {
        _, err := h.db.Exec("UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2", req.Email, id)
        if err != nil {
            h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Email already in use"})
            return
        }
    }

    if req.FullName != "" {
        h.db.Exec("UPDATE users SET full_name = $1, updated_at = NOW() WHERE id = $2", req.FullName, id)
    }

    if req.Password != "" {
        hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
        h.db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hashedPassword), id)
    }

    if claims.Role == "admin" {
        if req.Role != "" {
            h.db.Exec("UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", req.Role, id)
        }
        if req.IsActive != nil {
            h.db.Exec("UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2", *req.IsActive, id)
        }
    }

    h.logger.Info("User updated", "user_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "User updated successfully"})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    result, err := h.db.Exec("DELETE FROM users WHERE id = $1", id)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to delete user"})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "User not found"})
        return
    }

    h.logger.Info("User deleted", "user_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "User deleted successfully"})
}
