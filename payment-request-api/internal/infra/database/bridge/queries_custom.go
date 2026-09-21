// payment-request-api\internal\infra\database\bridge\queries_custom.go
package bridge

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrRefundReservationUnavailable = errors.New("refund reservation unavailable")

type RefundPaymentInfoRow struct {
	GatewayPaymentID string
	AmountCents      int64
	Status           string
	SellerID         string
}

func (q *Queries) GetRefundPaymentInfo(ctx context.Context, paymentID string) (RefundPaymentInfoRow, error) {
	const query = `SELECT gateway_payment_id, amount_cents, status, COALESCE(seller_id, '') FROM payment_requests WHERE uuid = $1::uuid`
	var row RefundPaymentInfoRow
	err := q.db.QueryRow(ctx, query, paymentID).Scan(&row.GatewayPaymentID, &row.AmountCents, &row.Status, &row.SellerID)
	return row, err
}

type RefundRecordRow struct {
	PaymentID       string
	IdempotencyKey  string
	AmountCents     int64
	Status          string
	GatewayRefundID string
}

func (q *Queries) ListProcessingRefunds(ctx context.Context, limit int32) ([]RefundRecordRow, error) {
	const query = `SELECT payment_request_uuid::text, idempotency_key, amount_cents, status, COALESCE(gateway_refund_id, '') FROM payment_refunds WHERE status = 'processing' ORDER BY updated_at ASC LIMIT $1`
	rows, err := q.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RefundRecordRow
	for rows.Next() {
		var row RefundRecordRow
		if err := rows.Scan(&row.PaymentID, &row.IdempotencyKey, &row.AmountCents, &row.Status, &row.GatewayRefundID); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (q *Queries) ReserveRefund(ctx context.Context, paymentID, idempotencyKey string, amountCents int64, reason string) (RefundRecordRow, error) {
	const existing = `SELECT payment_request_uuid::text, idempotency_key, amount_cents, status, COALESCE(gateway_refund_id, '') FROM payment_refunds WHERE idempotency_key = $1`
	var record RefundRecordRow
	if err := q.db.QueryRow(ctx, existing, idempotencyKey).Scan(&record.PaymentID, &record.IdempotencyKey, &record.AmountCents, &record.Status, &record.GatewayRefundID); err == nil {
		return record, nil
	}

	const insert = `
WITH totals AS (
  SELECT COALESCE(SUM(amount_cents) FILTER (WHERE status IN ('requested', 'processing', 'succeeded')), 0)::bigint AS refunded
  FROM payment_refunds WHERE payment_request_uuid = $1::uuid
), payment AS (
	SELECT amount_cents, status FROM payment_requests WHERE uuid = $1::uuid FOR UPDATE
)
INSERT INTO payment_refunds (payment_request_uuid, idempotency_key, amount_cents, reason, status)
SELECT $1::uuid, $2, $3, $4, 'processing'
FROM totals, payment
WHERE payment.status IN ('succeeded', 'partially_refunded')
  AND $3 > 0 AND totals.refunded + $3 <= payment.amount_cents
RETURNING payment_request_uuid::text, idempotency_key, amount_cents, status, ''`
	if err := q.db.QueryRow(ctx, insert, paymentID, idempotencyKey, amountCents, reason).Scan(&record.PaymentID, &record.IdempotencyKey, &record.AmountCents, &record.Status, &record.GatewayRefundID); err != nil {
		return RefundRecordRow{}, ErrRefundReservationUnavailable
	}
	return record, nil
}

func (q *Queries) MarkRefundSucceeded(ctx context.Context, idempotencyKey, gatewayRefundID string) error {
	const query = `
WITH updated AS (
  UPDATE payment_refunds SET status = 'succeeded', gateway_refund_id = $2, updated_at = NOW()
  WHERE idempotency_key = $1 RETURNING payment_request_uuid
), totals AS (
  SELECT u.payment_request_uuid, p.amount_cents, COALESCE(SUM(r.amount_cents) FILTER (WHERE r.status = 'succeeded'), 0)::bigint AS refunded
  FROM updated u JOIN payment_requests p ON p.uuid = u.payment_request_uuid
  JOIN payment_refunds r ON r.payment_request_uuid = u.payment_request_uuid
  GROUP BY u.payment_request_uuid, p.amount_cents
)
UPDATE payment_requests p SET status = CASE WHEN t.refunded >= t.amount_cents THEN 'refunded' ELSE 'partially_refunded' END, updated_at = NOW()
FROM totals t WHERE p.uuid = t.payment_request_uuid`
	_, err := q.db.Exec(ctx, query, idempotencyKey, gatewayRefundID)
	return err
}

func (q *Queries) MarkRefundFailed(ctx context.Context, idempotencyKey, code, message string) error {
	const query = `UPDATE payment_refunds SET status = 'failed', error_code = $2, error_message = $3, updated_at = NOW() WHERE idempotency_key = $1`
	_, err := q.db.Exec(ctx, query, idempotencyKey, code, message)
	return err
}

const getPaymentRequestByGatewayPaymentID = `-- name: GetPaymentRequestByGatewayPaymentID :one
SELECT uuid::text AS uuid, amount_cents, currency, status
FROM payment_requests
WHERE gateway_payment_id = $1
LIMIT 1
`

type GetPaymentRequestByGatewayPaymentIDRow struct {
	Uuid        string
	AmountCents int64
	Currency    string
	Status      string
}

func (q *Queries) GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (GetPaymentRequestByGatewayPaymentIDRow, error) {
	row := q.db.QueryRow(ctx, getPaymentRequestByGatewayPaymentID, gatewayPaymentID)
	var i GetPaymentRequestByGatewayPaymentIDRow
	err := row.Scan(
		&i.Uuid,
		&i.AmountCents,
		&i.Currency,
		&i.Status,
	)
	return i, err
}

const updatePaymentStatusByGatewayPaymentID = `-- name: UpdatePaymentStatusByGatewayPaymentID :execrows
  UPDATE payment_requests
  	SET 
		status = $1, 
		updated_at = NOW()
  	WHERE gateway_payment_id = $2
		AND status NOT IN ('succeeded', 'failed', 'canceled')
`

type UpdatePaymentStatusByGatewayPaymentIDParams struct {
	Status           string
	GatewayPaymentID string
}

func (q *Queries) UpdatePaymentStatusByGatewayPaymentID(
	ctx context.Context,
	arg *UpdatePaymentStatusByGatewayPaymentIDParams,
) (int64, error) {

	result, err := q.db.Exec(
		ctx,
		updatePaymentStatusByGatewayPaymentID,
		arg.Status,
		arg.GatewayPaymentID,
	)

	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

const updatePaymentStatusByUUID = `-- name: UpdatePaymentStatusByUUID :execrows
  UPDATE payment_requests
  	SET 
		status = $1, 
		updated_at = NOW()
	  WHERE uuid = $2::uuid
		AND status NOT IN ('failed', 'canceled', 'refunded', 'partially_refunded')
`

type UpdatePaymentStatusByUUIDParams struct {
	Status      string
	AmountCents int64
	Uuid        pgtype.UUID
}

func (q *Queries) UpdatePaymentStatusByUUID(
	ctx context.Context,
	arg *UpdatePaymentStatusByUUIDParams,
) (int64, error) {
	result, err := q.db.Exec(
		ctx,
		updatePaymentStatusByUUID,
		arg.Status,
		arg.Uuid,
	)

	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

const getPendingPaymentsForReconciliation = `-- name: GetPendingPaymentsForReconciliation :many
SELECT gateway_payment_id, uuid::text AS uuid
FROM payment_requests
WHERE status = 'pending'
  AND gateway_payment_id IS NOT NULL
  AND gateway_payment_id != ''
  AND updated_at <= $1
ORDER BY updated_at ASC
LIMIT $2
`

type GetPendingPaymentsForReconciliationRow struct {
	GatewayPaymentID string
	Uuid             string
}

func (q *Queries) GetPendingPaymentsForReconciliation(ctx context.Context, maxUpdatedAt time.Time, limit int32) ([]GetPendingPaymentsForReconciliationRow, error) {
	rows, err := q.db.Query(ctx, getPendingPaymentsForReconciliation, maxUpdatedAt, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GetPendingPaymentsForReconciliationRow
	for rows.Next() {
		var i GetPendingPaymentsForReconciliationRow
		var gwID *string
		if err := rows.Scan(&gwID, &i.Uuid); err != nil {
			return nil, err
		}
		if gwID != nil {
			i.GatewayPaymentID = *gwID
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
