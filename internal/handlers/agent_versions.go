package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type AgentVersion struct {
	ID           int       `json:"id"`
	Version      string    `json:"version"`
	DownloadURL  string    `json:"download_url"`
	Checksum     string    `json:"checksum"`
	ReleaseNotes string    `json:"release_notes"`
	IsStable     bool      `json:"is_stable"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetLatestAgentVersion returns the latest agent version
func (h *Handler) GetLatestAgentVersion(w http.ResponseWriter, r *http.Request) {
	var version AgentVersion
	
	query := `
		SELECT id, version, download_url, COALESCE(checksum, ''), COALESCE(release_notes, ''), COALESCE(is_stable, false), created_at
		FROM agent_versions
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	err := h.db.QueryRow(query).Scan(
		&version.ID,
		&version.Version,
		&version.DownloadURL,
		&version.Checksum,
		&version.ReleaseNotes,
		&version.IsStable,
		&version.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		h.sendJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   "No agent version available",
		})
		return
	}
	
	if err != nil {
		h.logger.Error("Failed to get agent version", map[string]interface{}{"error": err.Error()})
		h.sendJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to get agent version",
		})
		return
	}
	
	h.sendJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    version,
	})
}

// CreateAgentVersion creates a new agent version (admin only)
func (h *Handler) CreateAgentVersion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version      string `json:"version"`
		DownloadURL  string `json:"download_url"`
		Checksum     string `json:"checksum"`
		ReleaseNotes string `json:"release_notes"`
		IsStable     bool   `json:"is_stable"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	userID := r.Context().Value("user_id").(int)
	
	query := `
		INSERT INTO agent_versions (version, download_url, checksum, release_notes, is_stable)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	
	var versionID int
	var createdAt time.Time
	
	err := h.db.QueryRow(query, req.Version, req.DownloadURL, req.Checksum, req.ReleaseNotes, req.IsStable).Scan(&versionID, &createdAt)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create agent version",
		})
		return
	}
	
	h.logger.Info("Agent version created", map[string]interface{}{
		"by":         userID,
		"version_id": versionID,
		"version":    req.Version,
	})
	
	h.sendJSON(w, http.StatusCreated, Response{
		Success: true,
		Data: map[string]interface{}{
			"id":         versionID,
			"version":    req.Version,
			"created_at": createdAt,
		},
	})
}

// GetAgentVersions returns all agent versions (admin only)
func (h *Handler) GetAgentVersions(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, version, download_url, COALESCE(checksum, ''), COALESCE(release_notes, ''), COALESCE(is_stable, false), created_at
		FROM agent_versions
		ORDER BY created_at DESC
	`
	
	rows, err := h.db.Query(query)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to get agent versions",
		})
		return
	}
	defer rows.Close()
	
	versions := []AgentVersion{}
	
	for rows.Next() {
		var v AgentVersion
		
		err := rows.Scan(&v.ID, &v.Version, &v.DownloadURL, &v.Checksum, &v.ReleaseNotes, &v.IsStable, &v.CreatedAt)
		if err != nil {
			continue
		}
		
		versions = append(versions, v)
	}
	
	h.sendJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    versions,
	})
}
