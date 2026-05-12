package payment

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbbridge "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/database/bridge"
)

type PaymentRepositoryDB struct {
	queries *dbbridge.Queries
}

func NewPaymentRepositoryDB(pool *pgxpool.Pool) *PaymentRepositoryDB {
	return &PaymentRepositoryDB{queries: dbbridge.New(pool)}
}

func (r *PaymentRepositoryDB) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	// Convert string to pgtype.Text (pgx native null type)
	merchantRef := pgtype.Text{
		String: req.MerchantReference,
		Valid:  req.MerchantReference != "",
	}

	params := dbbridge.CreatePaymentRequestParams{
		IdempotencyKey:        req.IdempotencyKey,
		MerchantReference:     merchantRef,
		AmountCents:           req.AmountCents,
		Currency:              req.Currency,
		PaymentMethod:         req.PaymentMethod,
		Status:                "pending",
		FailureCode:           pgtype.Text{Valid: false},
		FailureMessage:        pgtype.Text{Valid: false},
		StripePaymentIntentID: pgtype.Text{Valid: false},
	}

	row, err := r.queries.CreatePaymentRequest(ctx, params)
	if err != nil {
		return CreatePaymentResponse{}, err
	}

	return CreatePaymentResponse{
		ID:            row.ID,
		PaymentMethod: row.PaymentMethod,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}
