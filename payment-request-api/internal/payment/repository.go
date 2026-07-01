package payment

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbbridge "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/database/bridge"
)

type PaymentRepositoryDB struct {
	queries *dbbridge.Queries
}

func NewPaymentRepositoryDB(pool *pgxpool.Pool) (*PaymentRepositoryDB, error) {
	if pool == nil {
		return nil, errors.New("nil db pool")
	}
	return &PaymentRepositoryDB{queries: dbbridge.New(pool)}, nil
}

func (r *PaymentRepositoryDB) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	row, err := r.queries.GetPaymentStatusAndClientSecret(ctx, parseStringToUUID(paymentUUID))
	log.Printf("Fetching payment status for payment: %s, result: %v", paymentUUID, row)
	if err != nil {
		return PaymentStatusResponse{}, err
	}

	return PaymentStatusResponse{
		Status:       row.Status,
		ClientSecret: row.StripeClientSecret,
	}, nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			// conflict happened and INSERT did nothing; fetch existing row
			existing, err2 := r.queries.GetPaymentRequestByIdempotencyKey(ctx, req.IdempotencyKey)
			if err2 != nil {
				return CreatePaymentResponse{}, err2
			}
			return CreatePaymentResponse{
				PaymentUUID:   existing.Uuid,
				PaymentMethod: existing.PaymentMethod,
				Status:        existing.Status,
				CreatedAt:     existing.CreatedAt.Time,
				UpdatedAt:     existing.UpdatedAt.Time,
			}, nil
		}
		return CreatePaymentResponse{}, err
	}

	return CreatePaymentResponse{
		PaymentUUID:   row.Uuid,
		PaymentMethod: row.PaymentMethod,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}
