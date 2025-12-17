-- Add commercial tracking fields to ISPs
ALTER TABLE isps ADD COLUMN IF NOT EXISTS cost_per_mbps DECIMAL(10,2) DEFAULT 0;
ALTER TABLE isps ADD COLUMN IF NOT EXISTS peak_traffic_mbps INTEGER DEFAULT 0;
ALTER TABLE isps ADD COLUMN IF NOT EXISTS monthly_bandwidth_gb INTEGER DEFAULT 0;

-- Add calculated savings to telemetry
ALTER TABLE telemetry ADD COLUMN IF NOT EXISTS bandwidth_saved_mbps DECIMAL(10,2) DEFAULT 0;
ALTER TABLE telemetry ADD COLUMN IF NOT EXISTS usd_savings_calculated DECIMAL(10,2) DEFAULT 0;

-- Comments
COMMENT ON COLUMN isps.cost_per_mbps IS 'Cost per Mbps from ISPs international provider (USD)';
COMMENT ON COLUMN isps.peak_traffic_mbps IS 'Peak traffic WITHOUT cache (Mbps) - baseline';
COMMENT ON COLUMN isps.monthly_bandwidth_gb IS 'Total monthly bandwidth used (GB)';
