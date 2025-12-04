package middleware

import (
    "context"
    "net/http"
    "os"
    "strings"

    "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

type Claims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, `{"success":false,"error":"Authorization header required"}`, http.StatusUnauthorized)
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            http.Error(w, `{"success":false,"error":"Bearer token required"}`, http.StatusUnauthorized)
            return
        }

        secret := os.Getenv("JWT_SECRET")
        if secret == "" {
            secret = "your-super-secret-key-change-in-production"
        }

        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })

        if err != nil || !token.Valid {
            http.Error(w, `{"success":false,"error":"Invalid token"}`, http.StatusUnauthorized)
            return
        }

        claims, ok := token.Claims.(*Claims)
        if !ok {
            http.Error(w, `{"success":false,"error":"Invalid token claims"}`, http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), UserContextKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetUserFromContext(r *http.Request) *Claims {
    claims, ok := r.Context().Value(UserContextKey).(*Claims)
    if !ok {
        return nil
    }
    return claims
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := GetUserFromContext(r)
            if claims == nil {
                http.Error(w, `{"success":false,"error":"Unauthorized"}`, http.StatusUnauthorized)
                return
            }

            for _, role := range roles {
                if claims.Role == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            http.Error(w, `{"success":false,"error":"Insufficient permissions"}`, http.StatusForbidden)
        })
    }
}
