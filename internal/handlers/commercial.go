package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetISPCommercialStats returns commercial metrics for an ISP
func (h *Handler) GetISPCommercialStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ispID, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid ISP ID"})
		return
	}

	// Get ISP data with commercial fields
	var isp struct {
		ID                 int     `json:"id"`
		Name               string  `json:"name"`
		CostPerMbps        float64 `json:"cost_per_mbps"`
		PeakTrafficMbps    int     `json:"peak_traffic_mbps"`
		MonthlyBandwidthGB int     `json:"monthly_bandwidth_gb"`
	}

	query := `SELECT id, name, cost_per_mbps, peak_traffic_mbps, monthly_bandwidth_gb 
	          FROM isps WHERE id = $1`
	err = h.db.QueryRow(query, ispID).Scan(
		&isp.ID, &isp.Name, &isp.CostPerMbps, &isp.PeakTrafficMbps, &isp.MonthlyBandwidthGB,
	)
	if err != nil {
		h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "ISP not found"})
		return
	}

	// Get latest telemetry for cache performance
	var telemetry struct {
		CacheHits        int64   `json:"cache_hits"`
		CacheMisses      int64   `json:"cache_misses"`
		BandwidthSavedMB float64 `json:"bandwidth_saved_mb"`
	}

	telemetryQuery := `
		SELECT 
			COALESCE(SUM(cache_hits), 0) as cache_hits,
			COALESCE(SUM(cache_misses), 0) as cache_misses,
			COALESCE(SUM(bandwidth_saved_mb), 0) as bandwidth_saved_mb
		FROM telemetry 
		WHERE isp_id = $1
		AND created_at >= NOW() - INTERVAL '30 days'
	`
	err = h.db.QueryRow(telemetryQuery, ispID).Scan(
		&telemetry.CacheHits, &telemetry.CacheMisses, &telemetry.BandwidthSavedMB,
	)
	if err != nil {
		// No telemetry yet - use zeros
		telemetry.CacheHits = 0
		telemetry.CacheMisses = 0
		telemetry.BandwidthSavedMB = 0
	}

	// Calculate commercial metrics
	totalRequests := telemetry.CacheHits + telemetry.CacheMisses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(telemetry.CacheHits) / float64(totalRequests) * 100
	}

	// Calculate bandwidth saved in Mbps (extrapolated to monthly average)
	// We take last 30 days of data and calculate average Mbps
	bandwidthSavedMbps := 0.0
	monthlySavedGB := 0.0
	if telemetry.BandwidthSavedMB > 0 {
		// Convert MB to GB
		monthlySavedGB = telemetry.BandwidthSavedMB / 1024.0
		// Convert to Mbps: (GB * 8 bits * 1024 MB) / (30 days * 24 hours * 3600 seconds)
		bandwidthSavedMbps = (monthlySavedGB * 8 * 1024) / (30 * 24 * 3600)
	}

	// Calculate traffic metrics
	peakTrafficWithCache := float64(isp.PeakTrafficMbps)
	peakTrafficWithoutCache := peakTrafficWithCache

	// If we have monthly bandwidth baseline, calculate reduction
	if isp.MonthlyBandwidthGB > 0 && hitRate > 0 {
		// Estimate savings: monthly_bandwidth * hit_rate%
		estimatedSavingsGB := float64(isp.MonthlyBandwidthGB) * (hitRate / 100)
		// Convert to Mbps
		bandwidthSavedMbps = (estimatedSavingsGB * 8 * 1024) / (30 * 24 * 3600)
		peakTrafficWithoutCache = float64(isp.PeakTrafficMbps)
		peakTrafficWithCache = peakTrafficWithoutCache * (1 - hitRate/100)
	}

	// Calculate USD savings
	monthlySavingsUSD := bandwidthSavedMbps * isp.CostPerMbps
	annualSavingsUSD := monthlySavingsUSD * 12

	// Calculate ROI
	systemCost := 2000.0
	roiMonths := 0.0
	if monthlySavingsUSD > 0 {
		roiMonths = systemCost / monthlySavingsUSD
	}

	// Build response
	response := map[string]interface{}{
		"isp_id":   isp.ID,
		"isp_name": isp.Name,
		"config": map[string]interface{}{
			"cost_per_mbps":         isp.CostPerMbps,
			"peak_traffic_baseline": isp.PeakTrafficMbps,
		},
		"performance": map[string]interface{}{
			"cache_hits":         telemetry.CacheHits,
			"cache_misses":       telemetry.CacheMisses,
			"total_requests":     totalRequests,
			"hit_rate_percent":   hitRate,
			"bandwidth_saved_mb": telemetry.BandwidthSavedMB,
		},
		"traffic": map[string]interface{}{
			"peak_without_cache_mbps": peakTrafficWithoutCache,
			"peak_with_cache_mbps":    peakTrafficWithCache,
			"bandwidth_saved_mbps":    bandwidthSavedMbps,
			"reduction_percent":       (bandwidthSavedMbps / peakTrafficWithoutCache) * 100,
		},
		"savings": map[string]interface{}{
			"monthly_usd":  monthlySavingsUSD,
			"annual_usd":   annualSavingsUSD,
			"roi_months":   roiMonths,
			"system_cost":  systemCost,
			"payback_days": roiMonths * 30,
		},
	}

	h.sendJSON(w, http.StatusOK, Response{Success: true, Data: response})
}

// UpdateISPCommercialConfig updates commercial configuration for an ISP
func (h *Handler) UpdateISPCommercialConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ispID, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid ISP ID"})
		return
	}

	var input struct {
		CostPerMbps     *float64 `json:"cost_per_mbps"`
		PeakTrafficMbps *int     `json:"peak_traffic_mbps"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid input"})
		return
	}

	// Update ISP commercial config
	query := `UPDATE isps SET `
	args := []interface{}{}
	argCount := 1

	if input.CostPerMbps != nil {
		if argCount > 1 {
			query += ", "
		}
		query += "cost_per_mbps = $" + strconv.Itoa(argCount)
		args = append(args, *input.CostPerMbps)
		argCount++
	}

	if input.PeakTrafficMbps != nil {
		if argCount > 1 {
			query += ", "
		}
		query += "peak_traffic_mbps = $" + strconv.Itoa(argCount)
		args = append(args, *input.PeakTrafficMbps)
		argCount++
	}

	query += " WHERE id = $" + strconv.Itoa(argCount)
	args = append(args, ispID)

	_, err = h.db.Exec(query, args...)
	if err != nil {
		h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to update configuration"})
		return
	}

	h.sendJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Commercial configuration updated",
	})
}
