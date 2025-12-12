# ISP SaaS Platform - Complete Installation Guide

**Version:** 1.0.0  
**Last Updated:** December 12, 2025

---

## 📋 TABLE OF CONTENTS

1. [System Requirements](#system-requirements)
2. [Quick Installation](#quick-installation)
3. [Detailed Installation](#detailed-installation)
4. [Configuration](#configuration)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)

---

## 🖥️ SYSTEM REQUIREMENTS

### Minimum Requirements:
- **OS:** Ubuntu 20.04 LTS or newer
- **CPU:** 2 vCPUs
- **RAM:** 4 GB
- **Storage:** 80 GB
- **Network:** Public IP address

### Recommended for Production:
- **OS:** Ubuntu 22.04 LTS
- **CPU:** 4 vCPUs
- **RAM:** 8 GB
- **Storage:** 160 GB SSD
- **Network:** Static IP, 1 Gbps connection

### Software Dependencies (Auto-installed):
- PostgreSQL 14+
- Redis 6+
- Nginx 1.18+
- Apache Traffic Server 9+
- Go 1.21+
- Node.js 20+
- dnsmasq
- apt-cacher-ng

---

## ⚡ QUICK INSTALLATION

### For Fresh Ubuntu Server:

# 1. Update system
sudo apt update && sudo apt upgrade -y

# 2. Clone repository
cd /root
git clone https://github.com/DevWisdom08/isp-saas-platform.git
cd isp-saas-platform

# 3. Run installation script
chmod +x scripts/install-all.sh
sudo ./scripts/install-all.sh

# 4. Configure environment
cp .env.example .env
nano .env  # Edit with your settings

# 5. Start services
sudo systemctl start isp-saas-api
sudo systemctl start trafficserver
sudo systemctl start nginx
sudo systemctl start dnsmasq

# 6. Access dashboard
# Open browser: http://YOUR_SERVER_IP/
# Default login: admin@ispsaas.com / Admin123456!

**Installation time:** ~15-20 minutes

---

## 📦 DETAILED INSTALLATION

### Step 1: System Preparation

# Update package lists
sudo apt update && sudo apt upgrade -y

# Install basic tools
sudo apt install -y git curl wget build-essential ufw

# Configure firewall
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 53/tcp
sudo ufw allow 53/udp
sudo ufw allow 3142/tcp
sudo ufw --force enable### 

Step 2: Install PostgreSQL

sudo apt install -y postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE isp_saas;
CREATE USER devwisdom WITH PASSWORD 'YOUR_SECURE_PASSWORD';
GRANT ALL PRIVILEGES ON DATABASE isp_saas TO devwisdom;
\q
EOF### 

Step 3: Install Redis

sudo apt install -y redis-server

# Configure Redis
sudo sed -i 's/supervised no/supervised systemd/' /etc/redis/redis.conf

# Start Redis
sudo systemctl restart redis
sudo systemctl enable redis### 

Step 4: Install Go

cd /tmp
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version### 

Step 5: Install Node.js

curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Verify
node --version
npm --version### 

Step 6: Install Nginx

sudo apt install -y nginx

# Start Nginx
sudo systemctl start nginx
sudo systemctl enable nginx### 

Step 7: Install Apache Traffic Server

sudo apt install -y trafficserver

# Start ATS
sudo systemctl start trafficserver
sudo systemctl enable trafficserver### 

Step 8: Install dnsmasq & apt-cacher-ng

# Stop systemd-resolved (conflicts with dnsmasq)
sudo systemctl stop systemd-resolved
sudo systemctl disable systemd-resolved

# Install packages
sudo apt install -y dnsmasq apt-cacher-ng

# Start services
sudo systemctl start dnsmasq
sudo systemctl enable dnsmasq
sudo systemctl start apt-cacher-ng
sudo systemctl enable apt-cacher-ng### 

Step 9: Clone and Build Backend

cd /root
git clone https://github.com/DevWisdom08/isp-saas-platform.git
cd isp-saas-platform

# Create .env file
cat > .env << 'ENVEOF'
DB_HOST=localhost
DB_PORT=5432
DB_USER=devwisdom
DB_PASSWORD=YOUR_SECURE_PASSWORD
DB_NAME=isp_saas
PORT=8080
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
REDIS_HOST=localhost
REDIS_PORT=6379
ENVEOF

# Build backend
go build -o isp-saas-api ./cmd/api/

# Create systemd service
sudo tee /etc/systemd/system/isp-saas-api.service > /dev/null << 'SERVICEEOF'
[Unit]
Description=ISP SaaS Platform API
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/isp-saas-platform
EnvironmentFile=/root/isp-saas-platform/.env
ExecStart=/root/isp-saas-platform/isp-saas-api
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
SERVICEEOF

# Start service
sudo systemctl daemon-reload
sudo systemctl start isp-saas-api
sudo systemctl enable isp-saas-api### 

Step 10: Build and Deploy Frontend

cd /root/isp-saas-platform
cd isp-saas-frontend

# Install dependencies
npm install

# Build for production
npm run build

# Deploy to web root
sudo mkdir -p /var/www/isp-saas
sudo cp -r dist/* /var/www/isp-saas/### 

Step 11: Configure Nginx

sudo tee /etc/nginx/sites-available/isp-saas-api > /dev/null << 'NGINXEOF'
server {
    listen 80;
    server_name _;

    # API proxy
    location /api/ {
        proxy_pass http://localhost:8080/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_cache_bypass $http_upgrade;
    }

    # Static files
    location /static/ {
        alias /var/www/isp-saas/;
        autoindex on;
    }

    # Frontend SPA
    location / {
        root /var/www/isp-saas;
        try_files $uri $uri/ /index.html;
    }

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
}
NGINXEOF

# Enable site
sudo ln -sf /etc/nginx/sites-available/isp-saas-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx### Step 12: Configure Traffic Server

See `docs/TRAFFIC_SERVER_CONFIG.md` for detailed ATS configuration.

---

## ⚙️ CONFIGURATION

### Database Configuration

Edit `/root/isp-saas-platform/.env`:

DB_HOST=localhost
DB_PORT=5432
DB_USER=devwisdom
DB_PASSWORD=your_secure_password_here
DB_NAME=isp_saas### API Configuration

PORT=8080
JWT_SECRET=change-this-to-a-random-secure-string### Redis Configuration

Default configuration works for most cases. Edit `/etc/redis/redis.conf` for custom settings.

---

## ✅ VERIFICATION

### Check All Services

# Check service status
sudo systemctl status isp-saas-api
sudo systemctl status postgresql
sudo systemctl status redis
sudo systemctl status nginx
sudo systemctl status trafficserver
sudo systemctl status dnsmasq
sudo systemctl status apt-cacher-ng

# Check API health
curl http://localhost:8080/api/health

# Check frontend
curl -I http://localhost/

# Check logs
sudo journalctl -u isp-saas-api -n 50 --no-pager### Test Login

1. Open browser: `http://YOUR_SERVER_IP/`
2. Login with:
   - Email: `admin@ispsaas.com`
   - Password: `Admin123456!` (change immediately!)
3. You should see the dashboard

---

## 🔧 TROUBLESHOOTING

### API Won't Start

# Check logs
sudo journalctl -u isp-saas-api -n 100 --no-pager

# Check database connection
sudo -u postgres psql -d isp_saas -c "SELECT version();"

# Check Redis
redis-cli ping### Database Connection Failed

# Check PostgreSQL is running
sudo systemctl status postgresql

# Check password
sudo -u postgres psql
\l  # List databases
\du # List users### Frontend Shows 502 Bad Gateway

# Check API is running
curl http://localhost:8080/api/health

# Check Nginx configuration
sudo nginx -t

# Check Nginx error log
sudo tail -50 /var/log/nginx/error.log### Port Already in Use

# Find process using port 8080
sudo lsof -i :8080

# Kill process if needed
sudo kill -9 PID---

## 📞 SUPPORT

- **Documentation:** `/root/isp-saas-platform/docs/`
- **GitHub Issues:** https://github.com/YOUR_USERNAME/isp-saas-platform/issues
- **Email:** support@yourcompany.com

---

**Installation Complete!** 🎉

Next Steps:
1. Change default admin password
2. Configure ISPs and distributors
3. Deploy agents to customer servers
4. Monitor cache performance

