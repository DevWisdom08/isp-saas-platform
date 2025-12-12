#!/bin/bash
# ISP SaaS Platform - Automatic Installer
# Version: 1.0.0

set -e  # Exit on any error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SAAS_API_URL="http://64.23.151.140:8080/api"
AGENT_DOWNLOAD_URL="http://64.23.151.140/static/isp-agent"
INSTALL_DIR="/opt/isp-agent"

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   ISP SaaS Platform - Automatic Installer v1.0.0      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
   echo -e "${RED}✗ Please run as root (use sudo)${NC}"
   exit 1
fi

echo -e "${GREEN}✓${NC} Running as root"

# Step 1: Collect installation information
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 1: Installation Information${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

read -p "Enter License Key: " LICENSE_KEY
read -p "Enter ISP Name: " ISP_NAME
read -p "Enter Contact Email: " CONTACT_EMAIL

echo ""
echo "Installation Details:"
echo "  License Key: $LICENSE_KEY"
echo "  ISP Name: $ISP_NAME"
echo "  Contact Email: $CONTACT_EMAIL"
echo ""
read -p "Continue with installation? (y/n): " CONFIRM

if [ "$CONFIRM" != "y" ]; then
    echo -e "${RED}Installation cancelled${NC}"
    exit 0
fi

# Step 2: System Update
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 2: System Update${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

apt-get update -qq
echo -e "${GREEN}✓${NC} System updated"

# Step 3: Install Dependencies
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 3: Installing Dependencies${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

apt-get install -y -qq curl wget nginx trafficserver apt-cacher-ng dnsmasq > /dev/null 2>&1
echo -e "${GREEN}✓${NC} Dependencies installed"

# Step 4: Create Installation Directory
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 4: Creating Installation Directory${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

mkdir -p $INSTALL_DIR
cd $INSTALL_DIR
echo -e "${GREEN}✓${NC} Created $INSTALL_DIR"

# Step 5: Download Agent
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 5: Downloading ISP Agent${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

wget -q --show-progress "$AGENT_DOWNLOAD_URL" -O isp-agent
chmod +x isp-agent
echo -e "${GREEN}✓${NC} Agent downloaded"

# Step 6: Validate License
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 6: Validating License${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Generate Hardware ID
HWID=$(./isp-agent -hwid)
echo "Hardware ID: $HWID"

# Validate license with API
VALIDATION=$(curl -s -X POST "$SAAS_API_URL/licenses/validate" \
  -H "Content-Type: application/json" \
  -d "{\"license_key\":\"$LICENSE_KEY\",\"hardware_id\":\"$HWID\"}")

if echo "$VALIDATION" | grep -q '"valid":true'; then
    echo -e "${GREEN}✓${NC} License validated successfully"
else
    echo -e "${RED}✗ License validation failed${NC}"
    echo "Response: $VALIDATION"
    exit 1
fi

# Step 7: Configure Agent
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 7: Configuring Agent${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat > $INSTALL_DIR/config.json << EOF
{
  "saas_api_url": "$SAAS_API_URL",
  "license_key": "$LICENSE_KEY",
  "hardware_id": "$HWID",
  "isp_name": "$ISP_NAME",
  "contact_email": "$CONTACT_EMAIL",
  "telemetry_interval": 300,
  "nginx_cache_path": "/var/cache/nginx"
}
EOF

echo -e "${GREEN}✓${NC} Agent configured"

# Step 8: Install systemd service
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 8: Installing systemd Service${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cat > /etc/systemd/system/isp-agent.service << EOF
[Unit]
Description=ISP SaaS Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/isp-agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable isp-agent
echo -e "${GREEN}✓${NC} Service installed"

# Step 9: Start Services
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 9: Starting Services${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

systemctl start isp-agent
systemctl start trafficserver
systemctl start nginx
echo -e "${GREEN}✓${NC} Services started"

# Step 10: Health Check
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Step 10: Health Check${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

sleep 3

if systemctl is-active --quiet isp-agent; then
    echo -e "${GREEN}✓${NC} ISP Agent is running"
else
    echo -e "${RED}✗${NC} ISP Agent failed to start"
fi

if systemctl is-active --quiet trafficserver; then
    echo -e "${GREEN}✓${NC} Traffic Server is running"
else
    echo -e "${RED}✗${NC} Traffic Server failed to start"
fi

if systemctl is-active --quiet nginx; then
    echo -e "${GREEN}✓${NC} Nginx is running"
else
    echo -e "${RED}✗${NC} Nginx failed to start"
fi

# Final Summary
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║         Installation Completed Successfully!           ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Installation Details:"
echo "  - Installation Directory: $INSTALL_DIR"
echo "  - License Key: $LICENSE_KEY"
echo "  - Hardware ID: $HWID"
echo "  - ISP Name: $ISP_NAME"
echo ""
echo "Service Status:"
systemctl status isp-agent --no-pager -l | head -3
echo ""
echo "Useful Commands:"
echo "  - View agent logs: journalctl -u isp-agent -f"
echo "  - Restart agent: systemctl restart isp-agent"
echo "  - Agent status: systemctl status isp-agent"
echo ""
echo -e "${GREEN}✓ Your ISP caching system is now operational!${NC}"
