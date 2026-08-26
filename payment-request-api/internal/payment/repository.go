// payment-request-api\internal\payment\repository.go
package payment

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbbridge "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/database/bridge"
)

type PaymentRepositoryDB struct {
	queries *dbbridge.Queries
}

func (r *PaymentRepositoryDB) UpdatePaymentStatus(ctx context.Context, paymentUUID string, status string, amountCents int64) error {
	parsedUUID := parseStringToUUID(paymentUUID)

	params := dbbridge.UpdatePaymentStatusParams{
		Status:      status,
		AmountCents: amountCents,
		Uuid:        parsedUUID,
	}

	err := r.queries.UpdatePaymentStatus(ctx, params)
	if err != nil {
		return err
	}
	return nil
}

func NewPaymentRepositoryDB(pool *pgxpool.Pool) (*PaymentRepositoryDB, error) {
	if pool == nil {
		return nil, errors.New("nil db pool")
	}
	return &PaymentRepositoryDB{queries: dbbridge.New(pool)}, nil
}

func (r *PaymentRepositoryDB) GetPaymentClientSecret(ctx context.Context, paymentUUID string) (PaymentStatusResponse, error) {
	parsedUUID := parseStringToUUID(paymentUUID)

	row, err := r.queries.GetPaymentClientSecret(ctx, parsedUUID)
	log.Printf("Fetching payment status for payment: %s, result: %v", paymentUUID, row)
	if err != nil {
		return PaymentStatusResponse{}, err
	}

	var expirationAt *time.Time
	if row.PixExpirationAt.Valid {
		exp := row.PixExpirationAt.Time
		expirationAt = &exp
	}

	return PaymentStatusResponse{
		Status:           row.Status,
		ClientSecret:     row.StripeClientSecret.String,
		Gateway:          row.Gateway,
		GatewayPaymentID: row.GatewayPaymentID.String,
		PixQRCode:        row.PixQrCode.String,
		PixQRCodeBase64:  row.PixQrCodeBase64.String,
		PixExpirationAt:  expirationAt,
	}, nil
}

func (r *PaymentRepositoryDB) GetPaymentRequestByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (PaymentGatewayValidationData, error) {
	row, err := r.queries.GetPaymentRequestByGatewayPaymentID(ctx, strings.TrimSpace(gatewayPaymentID))
	if err != nil {
		return PaymentGatewayValidationData{}, err
	}

	return PaymentGatewayValidationData{
		PaymentUUID:      row.Uuid,
		ExpectedAmount:   row.AmountCents,
		ExpectedCurrency: row.Currency,
		CurrentStatus:    row.Status,
	}, nil
}

func (r *PaymentRepositoryDB) UpdatePaymentStatusByGatewayPaymentID(
	ctx context.Context,
	gatewayPaymentID string,
	status string,
) (int64, error) {

	params := dbbridge.UpdatePaymentStatusByGatewayPaymentIDParams{
		Status:           status,
		GatewayPaymentID: strings.TrimSpace(gatewayPaymentID),
	}

	rowsAffected, err := r.queries.UpdatePaymentStatusByGatewayPaymentID(
		ctx,
		&params,
	)

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
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
		Gateway:               "mercado_pago",
		GatewayPaymentID:      pgtype.Text{Valid: false},
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
