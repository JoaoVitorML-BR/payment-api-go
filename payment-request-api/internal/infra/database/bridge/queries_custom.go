// payment-request-api\internal\infra\database\bridge\queries_custom.go
package bridge

import (
	"context"
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