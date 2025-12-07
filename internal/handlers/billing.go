package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/mux"
    "isp-saas.com/platform/internal/middleware"
)

type PlanResponse struct {
    ID             int      `json:"id"`
    Name           string   `json:"name"`
    Description    string   `json:"description"`
    PriceMonthly   float64  `json:"price_monthly"`
    BandwidthLimit *int     `json:"bandwidth_limit_mbps"`
    CacheSizeGB    *int     `json:"cache_size_gb"`
    MaxConnections *int     `json:"max_connections"`
    Features       []string `json:"features"`
    IsActive       bool     `json:"is_active"`
}

type InvoiceResponse struct {
    ID        int     `json:"id"`
    ISPID     int     `json:"isp_id"`
    ISPName   string  `json:"isp_name,omitempty"`
    Amount    float64 `json:"amount"`
    Status    string  `json:"status"`
    DueDate   string  `json:"due_date"`
    PaidAt    *string `json:"paid_at"`
    CreatedAt string  `json:"created_at"`
}

type CreateInvoiceRequest struct {
    ISPID   int     `json:"isp_id"`
    Amount  float64 `json:"amount"`
    DueDays int     `json:"due_days"`
}

func (h *Handler) GetPlans(w http.ResponseWriter, r *http.Request) {
    rows, err := h.db.Query(`
        SELECT id, name, COALESCE(description, ''), price_monthly, bandwidth_limit_mbps, 
               cache_size_gb, max_connections, features, is_active
        FROM plans WHERE is_active = true ORDER BY price_monthly
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var plans []PlanResponse
    for rows.Next() {
        var p PlanResponse
        var featuresJSON []byte
        rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.BandwidthLimit, 
            &p.CacheSizeGB, &p.MaxConnections, &featuresJSON, &p.IsActive)
        json.Unmarshal(featuresJSON, &p.Features)
        plans = append(plans, p)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: plans})
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var p PlanResponse
    var featuresJSON []byte
    err := h.db.QueryRow(`
        SELECT id, name, COALESCE(description, ''), price_monthly, bandwidth_limit_mbps,
               cache_size_gb, max_connections, features, is_active
        FROM plans WHERE id = $1
    `, id).Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.BandwidthLimit,
        &p.CacheSizeGB, &p.MaxConnections, &featuresJSON, &p.IsActive)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "Plan not found"})
        return
    }
    json.Unmarshal(featuresJSON, &p.Features)

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: p})
}

func (h *Handler) GetInvoices(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    
    var query string
    var args []interface{}

    if claims.Role == "admin" || claims.Role == "distributor" {
        query = `
            SELECT inv.id, inv.isp_id, i.name as isp_name, inv.amount, inv.status, 
                   inv.due_date, inv.paid_at, inv.created_at
            FROM invoices inv
            JOIN isps i ON inv.isp_id = i.id
            ORDER BY inv.created_at DESC
        `
    } else {
        query = `
            SELECT inv.id, inv.isp_id, i.name as isp_name, inv.amount, inv.status,
                   inv.due_date, inv.paid_at, inv.created_at
            FROM invoices inv
            JOIN isps i ON inv.isp_id = i.id
            WHERE i.user_id = $1
            ORDER BY inv.created_at DESC
        `
        args = []interface{}{claims.UserID}
    }

    rows, err := h.db.Query(query, args...)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Database error"})
        return
    }
    defer rows.Close()

    var invoices []InvoiceResponse
    for rows.Next() {
        var inv InvoiceResponse
        rows.Scan(&inv.ID, &inv.ISPID, &inv.ISPName, &inv.Amount, &inv.Status,
            &inv.DueDate, &inv.PaidAt, &inv.CreatedAt)
        invoices = append(invoices, inv)
    }

    h.sendJSON(w, http.StatusOK, Response{Success: true, Data: invoices})
}

func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    var req CreateInvoiceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "Invalid request body"})
        return
    }

    if req.ISPID == 0 || req.Amount <= 0 {
        h.sendJSON(w, http.StatusBadRequest, Response{Success: false, Error: "ISP ID and amount are required"})
        return
    }

    if req.DueDays == 0 {
        req.DueDays = 30
    }

    dueDate := time.Now().AddDate(0, 0, req.DueDays)

    var invoiceID int
    err := h.db.QueryRow(`
        INSERT INTO invoices (isp_id, amount, due_date) VALUES ($1, $2, $3) RETURNING id
    `, req.ISPID, req.Amount, dueDate).Scan(&invoiceID)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to create invoice"})
        return
    }

    h.logger.Info("Invoice created", "invoice_id", invoiceID, "isp_id", req.ISPID, "amount", req.Amount)
    h.sendJSON(w, http.StatusCreated, Response{
        Success: true,
        Message: "Invoice created successfully",
        Data: map[string]interface{}{
            "id":       invoiceID,
            "due_date": dueDate.Format("2006-01-02"),
        },
    })
}

func (h *Handler) MarkInvoicePaid(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    _, err := h.db.Exec(`UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1`, id)

    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to update invoice"})
        return
    }

    h.logger.Info("Invoice marked as paid", "invoice_id", id, "by", claims.UserID)
    h.sendJSON(w, http.StatusOK, Response{Success: true, Message: "Invoice marked as paid"})
}

func (h *Handler) CheckOverdueInvoices(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserFromContext(r)
    if claims.Role != "admin" {
        h.sendJSON(w, http.StatusForbidden, Response{Success: false, Error: "Admin access required"})
        return
    }

    result, err := h.db.Exec(`
        UPDATE invoices SET status = 'overdue' 
        WHERE status = 'pending' AND due_date < CURRENT_DATE
    `)
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, Response{Success: false, Error: "Failed to check invoices"})
        return
    }

    rows, _ := result.RowsAffected()

    h.db.Exec(`
        UPDATE isps SET status = 'suspended' 
        WHERE id IN (SELECT DISTINCT isp_id FROM invoices WHERE status = 'overdue') AND status = 'active'
    `)

    h.logger.Info("Overdue invoices checked", "updated", rows)
    h.sendJSON(w, http.StatusOK, Response{
        Success: true,
        Message: "Overdue invoices processed",
        Data:    map[string]int64{"invoices_marked_overdue": rows},
    })
}

func (h *Handler) GenerateInvoicePDF(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var invoiceID int
    var ispID int
    var amount float64
    var status, dueDate, createdAt string

    err := h.db.QueryRow(`
        SELECT inv.id, inv.isp_id, inv.amount, inv.status, inv.due_date, inv.created_at
        FROM invoices inv WHERE inv.id = $1
    `, id).Scan(&invoiceID, &ispID, &amount, &status, &dueDate, &createdAt)

    if err != nil {
        h.sendJSON(w, http.StatusNotFound, Response{Success: false, Error: "Invoice not found"})
        return
    }

    var ispName, serverIP string
    h.db.QueryRow("SELECT name, server_ip FROM isps WHERE id = $1", ispID).Scan(&ispName, &serverIP)

    var planName string
    h.db.QueryRow(`SELECT p.name FROM plans p JOIN isps i ON i.plan_id = p.id WHERE i.id = $1`, ispID).Scan(&planName)
    if planName == "" {
        planName = "Standard"
    }

    html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Invoice #INV-%d</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; color: #333; }
        .header { display: flex; justify-content: space-between; margin-bottom: 40px; border-bottom: 2px solid #3b82f6; padding-bottom: 20px; }
        .logo { font-size: 28px; font-weight: bold; color: #3b82f6; }
        .invoice-info { text-align: right; }
        .invoice-number { font-size: 24px; font-weight: bold; color: #1e293b; }
        .details { display: flex; justify-content: space-between; margin-bottom: 40px; }
        .bill-to, .from { width: 45%%; }
        .section-title { font-weight: bold; color: #666; margin-bottom: 10px; text-transform: uppercase; font-size: 12px; }
        table { width: 100%%; border-collapse: collapse; margin-bottom: 40px; }
        th { background: #3b82f6; color: white; padding: 12px; text-align: left; }
        td { padding: 12px; border-bottom: 1px solid #e2e8f0; }
        .total-row { background: #f8fafc; font-weight: bold; font-size: 18px; }
        .status { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: bold; }
        .status.paid { background: #d1fae5; color: #059669; }
        .status.pending { background: #fef3c7; color: #d97706; }
        .status.overdue { background: #fee2e2; color: #dc2626; }
        .footer { margin-top: 60px; text-align: center; color: #666; font-size: 12px; border-top: 1px solid #e2e8f0; padding-top: 20px; }
        @media print { body { margin: 20px; } }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">🚀 ISP SaaS Platform</div>
        <div class="invoice-info">
            <div class="invoice-number">INVOICE #INV-%d</div>
            <div>Date: %s</div>
            <div>Due: %s</div>
            <div style="margin-top: 10px;"><span class="status %s">%s</span></div>
        </div>
    </div>
    
    <div class="details">
        <div class="bill-to">
            <div class="section-title">Bill To:</div>
            <div style="font-size: 18px; font-weight: bold;">%s</div>
            <div>Server: %s</div>
        </div>
        <div class="from">
            <div class="section-title">From:</div>
            <div style="font-size: 18px; font-weight: bold;">ISP SaaS Platform</div>
            <div>Cache Management Services</div>
        </div>
    </div>
    
    <table>
        <thead>
            <tr>
                <th>Description</th>
                <th>Plan</th>
                <th>Period</th>
                <th style="text-align: right;">Amount</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td>ISP Cache Service - Monthly Subscription</td>
                <td>%s</td>
                <td>1 Month</td>
                <td style="text-align: right;">$%.2f</td>
            </tr>
            <tr class="total-row">
                <td colspan="3" style="text-align: right;">TOTAL:</td>
                <td style="text-align: right;">$%.2f USD</td>
            </tr>
        </tbody>
    </table>
    
    <div class="footer">
        <p><strong>Thank you for your business!</strong></p>
        <p>ISP SaaS Platform - Professional Cache Management Services</p>
        <p style="margin-top: 20px;">To print this invoice, press Ctrl+P (or Cmd+P on Mac)</p>
    </div>
</body>
</html>`, invoiceID, invoiceID, createdAt[:10], dueDate[:10], strings.ToLower(status), strings.ToUpper(status), 
       ispName, serverIP, planName, amount, amount)

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write([]byte(html))
}
