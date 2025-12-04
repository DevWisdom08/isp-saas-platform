-- ISP SaaS Platform - Initial Schema

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'distributor', 'isp')),
    full_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price_monthly DECIMAL(10,2) NOT NULL,
    bandwidth_limit_mbps INTEGER,
    cache_size_gb INTEGER,
    max_connections INTEGER,
    features JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS isps (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    server_ip INET NOT NULL,
    hw_id VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'inactive')),
    plan_id INTEGER REFERENCES plans(id),
    cache_size_gb INTEGER DEFAULT 10,
    bandwidth_limit_mbps INTEGER DEFAULT 1000,
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS licenses (
    id SERIAL PRIMARY KEY,
    isp_id INTEGER REFERENCES isps(id) ON DELETE CASCADE,
    license_key VARCHAR(255) UNIQUE NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT true,
    modules JSONB DEFAULT '["cache", "https", "monitoring"]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telemetry (
    id SERIAL PRIMARY KEY,
    isp_id INTEGER REFERENCES isps(id) ON DELETE CASCADE,
    cache_hits BIGINT DEFAULT 0,
    cache_misses BIGINT DEFAULT 0,
    bandwidth_saved_mb BIGINT DEFAULT 0,
    total_requests BIGINT DEFAULT 0,
    cache_size_used_mb INTEGER DEFAULT 0,
    cpu_usage DECIMAL(5,2),
    memory_usage DECIMAL(5,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoices (
    id SERIAL PRIMARY KEY,
    isp_id INTEGER REFERENCES isps(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'overdue', 'cancelled')),
    due_date DATE NOT NULL,
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS system_logs (
    id SERIAL PRIMARY KEY,
    level VARCHAR(20) NOT NULL,
    source VARCHAR(100),
    message TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Default plans
INSERT INTO plans (name, description, price_monthly, bandwidth_limit_mbps, cache_size_gb, max_connections, features) VALUES
('Basic', 'Para ISPs pequeños', 49.99, 1000, 10, 100, '["cache", "basic_stats"]'),
('Professional', 'Para ISPs medianos', 99.99, 5000, 50, 500, '["cache", "advanced_stats", "https_acceleration"]'),
('Enterprise', 'Para grandes redes', 199.99, 10000, 100, 2000, '["cache", "advanced_stats", "https_acceleration", "priority_support", "custom_rules"]')
ON CONFLICT DO NOTHING;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_isps_status ON isps(status);
CREATE INDEX IF NOT EXISTS idx_isps_hw_id ON isps(hw_id);
CREATE INDEX IF NOT EXISTS idx_licenses_key ON licenses(license_key);
CREATE INDEX IF NOT EXISTS idx_licenses_expires ON licenses(expires_at);
CREATE INDEX IF NOT EXISTS idx_telemetry_isp_created ON telemetry(isp_id, created_at);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_system_logs_created ON system_logs(created_at);
