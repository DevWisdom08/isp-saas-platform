-- Top cached sites/applications tracking

CREATE TABLE IF NOT EXISTS cached_sites (
    id SERIAL PRIMARY KEY,
    isp_id INTEGER REFERENCES isps(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL,
    hits BIGINT DEFAULT 0,
    bandwidth_saved_mb BIGINT DEFAULT 0,
    last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Unique constraint per ISP and domain
CREATE UNIQUE INDEX IF NOT EXISTS idx_cached_sites_isp_domain ON cached_sites(isp_id, domain);

-- Index for top sites queries
CREATE INDEX IF NOT EXISTS idx_cached_sites_hits ON cached_sites(hits DESC);
CREATE INDEX IF NOT EXISTS idx_cached_sites_isp ON cached_sites(isp_id);

-- Popular applications categories
CREATE TABLE IF NOT EXISTS app_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(50),
    domains TEXT[], -- Array of domain patterns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default app categories
INSERT INTO app_categories (name, icon, domains) VALUES
('YouTube', '🎬', ARRAY['youtube.com', 'googlevideo.com', 'ytimg.com']),
('Netflix', '🎥', ARRAY['netflix.com', 'nflxvideo.net']),
('Facebook', '📘', ARRAY['facebook.com', 'fbcdn.net', 'fb.com']),
('Instagram', '📷', ARRAY['instagram.com', 'cdninstagram.com']),
('TikTok', '🎵', ARRAY['tiktok.com', 'tiktokcdn.com']),
('WhatsApp', '💬', ARRAY['whatsapp.com', 'whatsapp.net']),
('Google', '🔍', ARRAY['google.com', 'googleapis.com', 'gstatic.com']),
('Windows Update', '🪟', ARRAY['windowsupdate.com', 'microsoft.com']),
('Steam', '🎮', ARRAY['steampowered.com', 'steamcontent.com']),
('Spotify', '🎧', ARRAY['spotify.com', 'scdn.co'])
ON CONFLICT DO NOTHING;
