package billing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TaxTypeNone      = "none"
	TaxTypeVAT       = "vat"
	TaxTypeGST       = "gst"
	TaxTypeSalesTax  = "sales_tax"
	PriceDisplayInc  = "inclusive"
	PriceDisplayExc  = "exclusive"
)

// OrgTaxSettings is per-org tax configuration (plan 15.13).
type OrgTaxSettings struct {
	OrgID                     uuid.UUID
	Enabled                   bool
	RegisteredJurisdictions   []string
	DefaultTaxCategory        string
	PriceDisplay              string
	FilingMode                string
	RecordRetentionYears      int
	SellerName                string
	SellerAddress             string
	SellerTaxID               string
	UpdatedAt                 time.Time
}

// TaxFields captures tax metadata stored on an entitlement.
type TaxFields struct {
	SubtotalCents            int
	TaxAmountCents           int
	TaxRate                  *float64
	TaxJurisdiction          string
	TaxType                  string
	TaxInclusive             bool
	CustomerCountry          string
	CustomerRegion           string
	CustomerTaxIDEnc         string
	ReverseCharge            bool
	StripeTaxCalculationID   string
	InvoiceID                *uuid.UUID
}

// TaxInvoice is a tax-compliant invoice record.
type TaxInvoice struct {
	ID             uuid.UUID
	EntitlementID  uuid.UUID
	InvoiceNumber  string
	PDFStorageKey  string
	IssuedAt       time.Time
	CreditedBy     *uuid.UUID
}

// TaxReportRow aggregates tax collected for a jurisdiction and period.
type TaxReportRow struct {
	Jurisdiction      string
	TaxType           string
	TransactionCount  int
	TaxCollectedCents int64
	SubtotalCents     int64
}

// GetOrgTaxSettings loads org tax settings or defaults when missing.
func GetOrgTaxSettings(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*OrgTaxSettings, error) {
	var s OrgTaxSettings
	var jurisdictionsJSON []byte
	err := pool.QueryRow(ctx, `
SELECT org_id, enabled, registered_jurisdictions_json, default_tax_category,
       price_display, filing_mode, record_retention_years,
       COALESCE(seller_name, ''), COALESCE(seller_address, ''), COALESCE(seller_tax_id, ''),
       updated_at
FROM billing.org_tax_settings
WHERE org_id = $1
`, orgID).Scan(
		&s.OrgID, &s.Enabled, &jurisdictionsJSON, &s.DefaultTaxCategory,
		&s.PriceDisplay, &s.FilingMode, &s.RecordRetentionYears,
		&s.SellerName, &s.SellerAddress, &s.SellerTaxID, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &OrgTaxSettings{
			OrgID:                orgID,
			DefaultTaxCategory:   "txcd_99999999",
			PriceDisplay:         PriceDisplayExc,
			FilingMode:           "manual",
			RecordRetentionYears: 7,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(jurisdictionsJSON) > 0 {
		_ = json.Unmarshal(jurisdictionsJSON, &s.RegisteredJurisdictions)
	}
	return &s, nil
}

// UpsertOrgTaxSettings saves org tax configuration.
func UpsertOrgTaxSettings(ctx context.Context, pool *pgxpool.Pool, s OrgTaxSettings) error {
	jurisdictionsJSON, err := json.Marshal(s.RegisteredJurisdictions)
	if err != nil {
		return err
	}
	if s.DefaultTaxCategory == "" {
		s.DefaultTaxCategory = "txcd_99999999"
	}
	if s.PriceDisplay == "" {
		s.PriceDisplay = PriceDisplayExc
	}
	if s.FilingMode == "" {
		s.FilingMode = "manual"
	}
	if s.RecordRetentionYears < 1 {
		s.RecordRetentionYears = 7
	}
	_, err = pool.Exec(ctx, `
INSERT INTO billing.org_tax_settings (
    org_id, enabled, registered_jurisdictions_json, default_tax_category,
    price_display, filing_mode, record_retention_years,
    seller_name, seller_address, seller_tax_id, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (org_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    registered_jurisdictions_json = EXCLUDED.registered_jurisdictions_json,
    default_tax_category = EXCLUDED.default_tax_category,
    price_display = EXCLUDED.price_display,
    filing_mode = EXCLUDED.filing_mode,
    record_retention_years = EXCLUDED.record_retention_years,
    seller_name = EXCLUDED.seller_name,
    seller_address = EXCLUDED.seller_address,
    seller_tax_id = EXCLUDED.seller_tax_id,
    updated_at = NOW()
`, s.OrgID, s.Enabled, jurisdictionsJSON, s.DefaultTaxCategory,
		s.PriceDisplay, s.FilingMode, s.RecordRetentionYears,
		s.SellerName, s.SellerAddress, s.SellerTaxID)
	return err
}

// UpdateEntitlementTax persists tax fields on an existing entitlement.
func UpdateEntitlementTax(ctx context.Context, pool *pgxpool.Pool, stripeEventID string, tax TaxFields) error {
	_, err := pool.Exec(ctx, `
UPDATE billing.user_entitlements SET
    subtotal_cents = $2,
    tax_amount_cents = $3,
    tax_rate = $4,
    tax_jurisdiction = NULLIF($5, ''),
    tax_type = $6,
    tax_inclusive = $7,
    customer_country = NULLIF($8, ''),
    customer_region = NULLIF($9, ''),
    customer_tax_id_enc = NULLIF($10, ''),
    reverse_charge = $11,
    stripe_tax_calculation_id = NULLIF($12, ''),
    invoice_id = $13
WHERE stripe_event_id = $1
`, stripeEventID,
		tax.SubtotalCents, tax.TaxAmountCents, tax.TaxRate, tax.TaxJurisdiction, tax.TaxType,
		tax.TaxInclusive, tax.CustomerCountry, tax.CustomerRegion, tax.CustomerTaxIDEnc,
		tax.ReverseCharge, tax.StripeTaxCalculationID, tax.InvoiceID)
	return err
}

// CreateTaxInvoice inserts a tax invoice and links it to the entitlement.
func CreateTaxInvoice(ctx context.Context, pool *pgxpool.Pool, entitlementID uuid.UUID, invoiceNumber, pdfKey string) (*TaxInvoice, error) {
	var inv TaxInvoice
	err := pool.QueryRow(ctx, `
INSERT INTO billing.tax_invoices (entitlement_id, invoice_number, pdf_storage_key)
VALUES ($1, $2, NULLIF($3, ''))
RETURNING id, entitlement_id, invoice_number, COALESCE(pdf_storage_key, ''), issued_at, credited_by
`, entitlementID, invoiceNumber, pdfKey).Scan(
		&inv.ID, &inv.EntitlementID, &inv.InvoiceNumber, &inv.PDFStorageKey, &inv.IssuedAt, &inv.CreditedBy,
	)
	if err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `UPDATE billing.user_entitlements SET invoice_id = $2 WHERE id = $1`, entitlementID, inv.ID)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetTaxInvoiceByID loads an invoice by id.
func GetTaxInvoiceByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*TaxInvoice, error) {
	var inv TaxInvoice
	err := pool.QueryRow(ctx, `
SELECT id, entitlement_id, invoice_number, COALESCE(pdf_storage_key, ''), issued_at, credited_by
FROM billing.tax_invoices WHERE id = $1
`, id).Scan(&inv.ID, &inv.EntitlementID, &inv.InvoiceNumber, &inv.PDFStorageKey, &inv.IssuedAt, &inv.CreditedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// EntitlementWithTax loads entitlement + tax fields for invoice generation.
type EntitlementWithTax struct {
	Entitlement
	SubtotalCents          int
	TaxAmountCents         int
	TaxRate                *float64
	TaxJurisdiction        string
	TaxType                string
	TaxInclusive           bool
	CustomerCountry        string
	CustomerRegion         string
	CustomerTaxIDEnc       string
	ReverseCharge          bool
	StripeTaxCalculationID string
	InvoiceID              *uuid.UUID
	UserEmail              string
}

// GetEntitlementWithTax loads entitlement tax details by id.
func GetEntitlementWithTax(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*EntitlementWithTax, error) {
	var e EntitlementWithTax
	var courseID *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT e.id, e.user_id, e.entitlement_type, e.course_id, e.stripe_event_id, e.stripe_invoice_id,
       e.amount_paid_cents, e.currency, e.valid_from, e.valid_until, e.status, e.created_at,
       e.subtotal_cents, e.tax_amount_cents, e.tax_rate, COALESCE(e.tax_jurisdiction, ''),
       e.tax_type, e.tax_inclusive, COALESCE(e.customer_country, ''), COALESCE(e.customer_region, ''),
       COALESCE(e.customer_tax_id_enc, ''), e.reverse_charge, COALESCE(e.stripe_tax_calculation_id, ''),
       e.invoice_id, u.email
FROM billing.user_entitlements e
JOIN "user".users u ON u.id = e.user_id
WHERE e.id = $1
`, id).Scan(
		&e.ID, &e.UserID, &e.EntitlementType, &courseID, &e.StripeEventID, &e.StripeInvoiceID,
		&e.AmountPaidCents, &e.Currency, &e.ValidFrom, &e.ValidUntil, &e.Status, &e.CreatedAt,
		&e.SubtotalCents, &e.TaxAmountCents, &e.TaxRate, &e.TaxJurisdiction,
		&e.TaxType, &e.TaxInclusive, &e.CustomerCountry, &e.CustomerRegion,
		&e.CustomerTaxIDEnc, &e.ReverseCharge, &e.StripeTaxCalculationID, &e.InvoiceID, &e.UserEmail,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.CourseID = courseID
	return &e, nil
}

// TaxReport generates aggregated tax totals for a period and optional jurisdiction.
func TaxReport(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, to time.Time, jurisdiction string) ([]TaxReportRow, error) {
	query := `
SELECT COALESCE(e.tax_jurisdiction, 'unknown') AS jurisdiction,
       e.tax_type,
       COUNT(*)::int AS transaction_count,
       COALESCE(SUM(e.tax_amount_cents), 0)::bigint AS tax_collected_cents,
       COALESCE(SUM(e.subtotal_cents), 0)::bigint AS subtotal_cents
FROM billing.user_entitlements e
LEFT JOIN course.courses c ON c.id = e.course_id
WHERE e.tax_type <> 'none'
  AND e.status = 'active'
  AND e.created_at >= $1
  AND e.created_at < $2
  AND (c.org_id = $3 OR e.entitlement_type LIKE 'subscription%')
`
	args := []any{from, to, orgID}
	if jurisdiction != "" {
		query += ` AND e.tax_jurisdiction = $4`
		args = append(args, jurisdiction)
	}
	query += ` GROUP BY e.tax_jurisdiction, e.tax_type ORDER BY e.tax_jurisdiction, e.tax_type`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaxReportRow
	for rows.Next() {
		var r TaxReportRow
		if err := rows.Scan(&r.Jurisdiction, &r.TaxType, &r.TransactionCount, &r.TaxCollectedCents, &r.SubtotalCents); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReverseEntitlementTaxByCourse marks the latest course entitlement refunded and issues a credit note.
func ReverseEntitlementTaxByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*uuid.UUID, error) {
	var entID uuid.UUID
	var taxAmount int
	err := pool.QueryRow(ctx, `
SELECT id, tax_amount_cents FROM billing.user_entitlements
WHERE course_id = $1 AND status = 'active' AND entitlement_type = 'course_purchase'
ORDER BY created_at DESC
LIMIT 1
`, courseID).Scan(&entID, &taxAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `
UPDATE billing.user_entitlements SET status = 'refunded' WHERE id = $1
`, entID)
	if err != nil {
		return nil, err
	}
	if taxAmount <= 0 {
		return nil, nil
	}
	var creditID uuid.UUID
	creditNumber := "CR-" + entID.String()[:8]
	err = pool.QueryRow(ctx, `
INSERT INTO billing.tax_invoices (entitlement_id, invoice_number, credited_by)
SELECT $1, $2, invoice_id FROM billing.user_entitlements WHERE id = $1
RETURNING id
`, entID, creditNumber).Scan(&creditID)
	if err != nil {
		return nil, err
	}
	return &creditID, nil
}