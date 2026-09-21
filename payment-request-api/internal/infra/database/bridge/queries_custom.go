// payment-request-api\internal\infra\database\bridge\queries_custom.go
package bridge

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
