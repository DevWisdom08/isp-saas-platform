#!/bin/bash
# ISP SaaS Platform - API Test Script
# Run: bash test_api.sh

API_URL="http://64.23.151.140"
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "================================================"
echo "   ISP SaaS Platform - API Test"
echo "================================================"
echo ""

# Test 1: Health Check
echo "1. Testing Health Check..."
HEALTH=$(curl -s $API_URL/api/health)
if echo $HEALTH | grep -q "success\":true"; then
    echo -e "${GREEN}✅ Health Check: PASSED${NC}"
else
    echo -e "${RED}❌ Health Check: FAILED${NC}"
fi
echo "   Response: $HEALTH"
echo ""

# Test 2: Login
echo "2. Testing Login..."
LOGIN=$(curl -s -X POST $API_URL/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@ispsaas.com","password":"Admin123456"}')

TOKEN=$(echo $LOGIN | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
    echo -e "${GREEN}✅ Login: PASSED${NC}"
    echo "   Token received: ${TOKEN:0:50}..."
else
    echo -e "${RED}❌ Login: FAILED${NC}"
    echo "   Response: $LOGIN"
fi
echo ""

# Test 3: Get Plans (Public)
echo "3. Testing Get Plans..."
PLANS=$(curl -s $API_URL/api/plans)
if echo $PLANS | grep -q "Basic"; then
    echo -e "${GREEN}✅ Get Plans: PASSED${NC}"
    PLAN_COUNT=$(echo $PLANS | grep -o '"id":' | wc -l)
    echo "   Plans found: $PLAN_COUNT"
else
    echo -e "${RED}❌ Get Plans: FAILED${NC}"
fi
echo ""

# Test 4: Dashboard Stats (Protected)
echo "4. Testing Dashboard Stats..."
STATS=$(curl -s -H "Authorization: Bearer $TOKEN" $API_URL/api/dashboard/stats)
if echo $STATS | grep -q "success\":true"; then
    echo -e "${GREEN}✅ Dashboard Stats: PASSED${NC}"
else
    echo -e "${RED}❌ Dashboard Stats: FAILED${NC}"
fi
echo ""

# Test 5: Get Users (Protected)
echo "5. Testing Get Users..."
USERS=$(curl -s -H "Authorization: Bearer $TOKEN" $API_URL/api/users)
if echo $USERS | grep -q "success\":true"; then
    echo -e "${GREEN}✅ Get Users: PASSED${NC}"
    USER_COUNT=$(echo $USERS | grep -o '"id":' | wc -l)
    echo "   Users found: $USER_COUNT"
else
    echo -e "${RED}❌ Get Users: FAILED${NC}"
fi
echo ""

# Test 6: Get Settings (Protected)
echo "6. Testing Get Settings..."
SETTINGS=$(curl -s -H "Authorization: Bearer $TOKEN" $API_URL/api/settings)
if echo $SETTINGS | grep -q "success\":true"; then
    echo -e "${GREEN}✅ Get Settings: PASSED${NC}"
else
    echo -e "${RED}❌ Get Settings: FAILED${NC}"
fi
echo ""

# Test 7: Get Distributors (Protected)
echo "7. Testing Get Distributors..."
DIST=$(curl -s -H "Authorization: Bearer $TOKEN" $API_URL/api/distributors)
if echo $DIST | grep -q "success\":true"; then
    echo -e "${GREEN}✅ Get Distributors: PASSED${NC}"
else
    echo -e "${RED}❌ Get Distributors: FAILED${NC}"
fi
echo ""

# Test 8: System Logs (Protected)
echo "8. Testing System Logs..."
LOGS=$(curl -s -H "Authorization: Bearer $TOKEN" $API_URL/api/logs)
if echo $LOGS | grep -q "success\":true"; then
    echo -e "${GREEN}✅ System Logs: PASSED${NC}"
else
    echo -e "${RED}❌ System Logs: FAILED${NC}"
fi
echo ""

echo "================================================"
echo "   TEST COMPLETE"
echo "================================================"
echo ""
echo "Admin Login Credentials:"
echo "  Email: admin@ispsaas.com"
echo "  Password: Admin123456"
echo ""
echo "API Base URL: $API_URL/api"
echo ""
