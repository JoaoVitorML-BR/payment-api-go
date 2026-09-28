// payment-consumer/worker/payment_requested_processor_test.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mercadopago/sdk-go/pkg/mperror"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ---------------------------------------------------------------------------
// fakeGateway — configurable in-memory mock of paymentgateway.Gateway
// ---------------------------------------------------------------------------

type fakeGateway struct {
	mu          sync.Mutex
	createCalls int
	onCreate    func(ctx context.Context, input paymentgateway.CreatePaymentInput, callNum int) (*paymentgateway.PaymentResult, error)
}

func (f *fakeGateway) CreatePayment(ctx context.Context, input paymentgateway.CreatePaymentInput) (*paymentgateway.PaymentResult, error) {
	f.mu.Lock()
	f.createCalls++
	callNum := f.createCalls
	f.mu.Unlock()

	if f.onCreate != nil {
		return f.onCreate(ctx, input, callNum)
	}
	return &paymentgateway.PaymentResult{
		GatewayPaymentID: "gw-test-123",
		Status:           paymentgateway.StatusPending,
		RawStatus:        "pending",
		AmountCents:      input.AmountCents,
		Currency:         input.Currency,
	}, nil
}

func (f *fakeGateway) GetPayment(_ context.Context, _ string) (*paymentgateway.PaymentResult, error) {
	return nil, nil
}

func (f *fakeGateway) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls
}

func retryableErr() error {
	return &mperror.ResponseError{StatusCode: 500, Message: "Internal Server Error"}
}

func nonRetryableErr() error {
	return &mperror.ResponseError{StatusCode: 422, Message: "Invalid Tax ID"}
}

func TestIsRetryableGatewayError(t *testing.T) {
	rErr := retryableErr()
	if !paymentgateway.IsRetryableGatewayError(rErr) {
		t.Errorf("expected retryableErr (500) to be retryable")
	}
	nrErr := nonRetryableErr()
	if paymentgateway.IsRetryableGatewayError(nrErr) {
		t.Errorf("expected nonRetryableErr (422) to NOT be retryable")
	}
}

// ---------------------------------------------------------------------------
// fakePaymentQueries — in-memory mock of the PaymentQueries interface
// ---------------------------------------------------------------------------

type fakePaymentQueries struct {
	mu                   sync.Mutex
	latestAttemptByUUID  map[string]*bridge.GetLatestPaymentAttemptRow
	savedAttempts        []bridge.UpdatePaymentAttemptParams
	paymentRequestStatus map[string]string
	markedFailedUUIDs    []string
	successUpdates       []bridge.UpdatePaymentRequestSuccessParams
}

func newFakePaymentQueries() *fakePaymentQueries {
	return &fakePaymentQueries{
		latestAttemptByUUID:  make(map[string]*bridge.GetLatestPaymentAttemptRow),
		paymentRequestStatus: make(map[string]string),
	}
}

// seedExistingAttempt pre-populates a latest attempt so Handle sees it from the start.
func (f *fakePaymentQueries) seedExistingAttempt(uuidKey, status string, attemptNumber int32, gwIDValid bool, gwID string) {
	row := &bridge.GetLatestPaymentAttemptRow{
		AttemptNumber: attemptNumber,
		Status:        status,
	}
	if gwIDValid {
		row.GatewayPaymentID = pgtype.Text{String: gwID, Valid: true}
	}
	f.latestAttemptByUUID[uuidKey] = row
}

func (f *fakePaymentQueries) GetLatestPaymentAttempt(_ context.Context, paymentRequestUuid pgtype.UUID) (bridge.GetLatestPaymentAttemptRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := pgUUIDToStr(paymentRequestUuid)
	row, ok := f.latestAttemptByUUID[key]
	if !ok || row == nil {
		return bridge.GetLatestPaymentAttemptRow{}, pgx.ErrNoRows
	}
	return *row, nil
}

func (f *fakePaymentQueries) UpdatePaymentAttempt(_ context.Context, arg bridge.UpdatePaymentAttemptParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.savedAttempts = append(f.savedAttempts, arg)

	// Reflect saved state so the next GetLatestPaymentAttempt call sees it.
	key := pgUUIDToStr(arg.PaymentRequestUuid)
	f.latestAttemptByUUID[key] = &bridge.GetLatestPaymentAttemptRow{
		PaymentRequestUuid: arg.PaymentRequestUuid,
		GatewayPaymentID:   arg.GatewayPaymentID,
		AttemptNumber:      arg.AttemptNumber,
		Currency:           arg.Currency,
		Status:             arg.Status,
		ErrorCode:          arg.ErrorCode,
		ErrorMessage:       arg.ErrorMessage,
		Response:           arg.Response,
	}
	return nil
}

func (f *fakePaymentQueries) UpdatePaymentRequestFailed(_ context.Context, uuid pgtype.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := pgUUIDToStr(uuid)
	f.paymentRequestStatus[key] = "failed"
	f.markedFailedUUIDs = append(f.markedFailedUUIDs, key)
	return nil
}

func (f *fakePaymentQueries) UpdatePaymentRequestSuccess(_ context.Context, arg bridge.UpdatePaymentRequestSuccessParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := pgUUIDToStr(arg.Uuid)
	f.paymentRequestStatus[key] = arg.Status
	f.successUpdates = append(f.successUpdates, arg)
	return nil
}

func (f *fakePaymentQueries) attemptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.savedAttempts)
}

func (f *fakePaymentQueries) attemptAt(i int) bridge.UpdatePaymentAttemptParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.savedAttempts[i]
}

func (f *fakePaymentQueries) reqStatus(uuidKey string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.paymentRequestStatus[uuidKey]
}

func (f *fakePaymentQueries) markFailedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.markedFailedUUIDs)
}

// pgUUIDToStr converts a pgtype.UUID to its canonical hyphenated form.
func pgUUIDToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------------------------------------------------------------------------
// Test helpers & Setup
// ---------------------------------------------------------------------------

const testPaymentUUID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

func makeDelivery(paymentID string) amqp.Delivery {
	msg := paymentRequestedMessage{
		EventName:      "payment.requested.v1",
		PaymentID:      paymentID,
		IdempotencyKey: "idemp-" + paymentID,
		AmountCents:    5000,
		Currency:       "BRL",
		PaymentMethod:  "pix",
	}
	body, _ := json.Marshal(msg)
	return amqp.Delivery{Body: body}
}

func testConfig() *config.Config {
	return &config.Config{MercadoPagoWebhookURL: "https://example.com/webhook"}
}

func uuidKey() string {
	var u pgtype.UUID
	_ = u.Scan(testPaymentUUID)
	return pgUUIDToStr(u)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// a) TestHandle_SuccessOnFirstAttempt: gateway succeeds on 1st call.
func TestHandle_SuccessOnFirstAttempt(t *testing.T) {
	db := newFakePaymentQueries()
	gw := &fakeGateway{}

	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	err := p.Handle(context.Background(), d)
	if err != nil {
		t.Errorf("esperava Handle retornar nil, veio: %v", err)
	}

	if gw.CallCount() != 1 {
		t.Errorf("esperava CreatePayment chamado 1 vez, veio %d", gw.CallCount())
	}

	if db.attemptCount() != 1 {
		t.Errorf("esperava 1 payment_attempt salvo, veio %d", db.attemptCount())
	}

	saved := db.attemptAt(0)
	if saved.AttemptNumber != 1 {
		t.Errorf("esperava attempt_number = 1, veio %d", saved.AttemptNumber)
	}
	if saved.Status == "failed" {
		t.Errorf("esperava status do attempt diferente de 'failed', veio %q", saved.Status)
	}
	if !saved.GatewayPaymentID.Valid {
		t.Errorf("esperava GatewayPaymentID válido (não-vazio) para tentativa de sucesso")
	}

	reqStat := db.reqStatus(uuidKey())
	if reqStat == "failed" || reqStat == "" {
		t.Errorf("esperava status de payment_requests indicando sucesso/pending (não 'failed' ou vazio), veio %q", reqStat)
	}

	if db.markFailedCount() != 0 {
		t.Errorf("não esperava UpdatePaymentRequestFailed ser chamado, foi chamado %d vez(es)", db.markFailedCount())
	}
}

// b) TestHandle_RetryableErrorThenSuccess: gateway falha 2x com erro retryable e sucede na 3ª.
func TestHandle_RetryableErrorThenSuccess(t *testing.T) {
	db := newFakePaymentQueries()
	gw := &fakeGateway{
		onCreate: func(ctx context.Context, input paymentgateway.CreatePaymentInput, callNum int) (*paymentgateway.PaymentResult, error) {
			if callNum <= 2 {
				return nil, retryableErr()
			}
			return &paymentgateway.PaymentResult{
				GatewayPaymentID: "gw-success-456",
				Status:           paymentgateway.StatusPending,
				RawStatus:        "pending",
				AmountCents:      input.AmountCents,
				Currency:         input.Currency,
			}, nil
		},
	}

	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	// Chamada 1: deve retornar erro (retryable → RabbitMQ vai reentregar)
	err1 := p.Handle(context.Background(), d)
	if err1 == nil {
		t.Error("esperava Handle (chamada 1) retornar erro não-nil (retryable), veio nil")
	}

	// Chamada 2: deve retornar erro (retryable novamente)
	err2 := p.Handle(context.Background(), d)
	if err2 == nil {
		t.Error("esperava Handle (chamada 2) retornar erro não-nil (retryable), veio nil")
	}

	// Chamada 3: gateway sucede
	err3 := p.Handle(context.Background(), d)
	if err3 != nil {
		t.Errorf("esperava Handle (chamada 3) retornar nil, veio: %v", err3)
	}

	if gw.CallCount() != 3 {
		t.Errorf("esperava CreatePayment chamado 3 vezes, veio %d", gw.CallCount())
	}

	if db.attemptCount() != 3 {
		t.Errorf("esperava 3 payment_attempts salvos, veio %d", db.attemptCount())
	}

	for i := 0; i < 3; i++ {
		a := db.attemptAt(i)
		expectedAttemptNumber := int32(i + 1)
		if a.AttemptNumber != expectedAttemptNumber {
			t.Errorf("attempt[%d]: esperava attempt_number = %d, veio %d", i, expectedAttemptNumber, a.AttemptNumber)
		}
	}

	if db.attemptAt(0).Status != "failed" || db.attemptAt(1).Status != "failed" {
		t.Errorf("esperava que as 2 primeiras tentativas fossem 'failed', vieram %q e %q",
			db.attemptAt(0).Status, db.attemptAt(1).Status)
	}
	if db.attemptAt(2).Status == "failed" {
		t.Errorf("esperava que a 3ª tentativa tivesse status de sucesso (não 'failed'), veio %q", db.attemptAt(2).Status)
	}

	reqStat := db.reqStatus(uuidKey())
	if reqStat == "failed" {
		t.Errorf("esperava status de payment_requests diferente de 'failed' após 3ª chamada com sucesso, veio %q", reqStat)
	}
	if db.markFailedCount() != 0 {
		t.Errorf("não esperava UpdatePaymentRequestFailed ser chamado, foi chamado %d vez(es)", db.markFailedCount())
	}
}

// c) TestHandle_ExhaustsRetriesAndMarksFailed: gateway falha sempre com erro retryable.
func TestHandle_ExhaustsRetriesAndMarksFailed(t *testing.T) {
	db := newFakePaymentQueries()
	gw := &fakeGateway{
		onCreate: func(ctx context.Context, input paymentgateway.CreatePaymentInput, callNum int) (*paymentgateway.PaymentResult, error) {
			return nil, retryableErr()
		},
	}

	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	// Chamadas 1 e 2: retornam erro (retryable)
	err1 := p.Handle(context.Background(), d)
	if err1 == nil {
		t.Error("esperava Handle (chamada 1) retornar erro, veio nil")
	}

	err2 := p.Handle(context.Background(), d)
	if err2 == nil {
		t.Error("esperava Handle (chamada 2) retornar erro, veio nil")
	}

	// Chamada 3 (tentativa 3 = maxCreatePaymentAttempts): retorna nil (ACK) e marca failed
	err3 := p.Handle(context.Background(), d)
	if err3 != nil {
		t.Errorf("esperava Handle (chamada 3) retornar nil (ACK), veio: %v", err3)
	}

	if gw.CallCount() != 3 {
		t.Errorf("esperava CreatePayment chamado exatamente 3 vezes, veio %d", gw.CallCount())
	}

	if db.attemptCount() != 3 {
		t.Errorf("esperava 3 payment_attempts salvos, veio %d", db.attemptCount())
	}

	for i := 0; i < 3; i++ {
		a := db.attemptAt(i)
		expectedNum := int32(i + 1)
		if a.AttemptNumber != expectedNum {
			t.Errorf("attempt[%d]: esperava attempt_number = %d, veio %d", i, expectedNum, a.AttemptNumber)
		}
		if a.Status != "failed" {
			t.Errorf("attempt[%d]: esperava status = 'failed', veio %q", i, a.Status)
		}
	}

	if db.reqStatus(uuidKey()) != "failed" {
		t.Errorf("esperava status de payment_requests = 'failed' após esgotar tentativas, veio %q", db.reqStatus(uuidKey()))
	}
	if db.markFailedCount() != 1 {
		t.Errorf("esperava UpdatePaymentRequestFailed chamado exatamente 1 vez, veio %d", db.markFailedCount())
	}
}

// d) TestHandle_NonRetryableErrorFailsImmediately: gateway falha na 1ª chamada com erro NÃO retryable.
// Nota de comportamento do código real:
//   - Handle retorna nil (ACK) já na 1ª tentativa e marca payment_requests.status = 'failed' imediatamente.
//   - Em produção, como retornou nil, o RabbitMQ dá ACK e a mensagem NÃO é reentregada.
func TestHandle_NonRetryableErrorFailsImmediately(t *testing.T) {
	db := newFakePaymentQueries()
	gw := &fakeGateway{
		onCreate: func(ctx context.Context, input paymentgateway.CreatePaymentInput, callNum int) (*paymentgateway.PaymentResult, error) {
			return nil, nonRetryableErr()
		},
	}

	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	err := p.Handle(context.Background(), d)
	if err != nil {
		t.Errorf("esperava Handle retornar nil (ACK, sem retry) para erro não-retryable, veio: %v", err)
	}

	if gw.CallCount() != 1 {
		t.Errorf("esperava CreatePayment chamado exatamente 1 vez, veio %d", gw.CallCount())
	}

	if db.attemptCount() != 1 {
		t.Errorf("esperava 1 payment_attempt salvo, veio %d", db.attemptCount())
	}

	attempt := db.attemptAt(0)
	if attempt.AttemptNumber != 1 {
		t.Errorf("esperava attempt_number = 1, veio %d", attempt.AttemptNumber)
	}
	if attempt.Status != "failed" {
		t.Errorf("esperava status do attempt = 'failed', veio %q", attempt.Status)
	}

	if db.reqStatus(uuidKey()) != "failed" {
		t.Errorf("esperava status de payment_requests = 'failed', veio %q", db.reqStatus(uuidKey()))
	}
	if db.markFailedCount() != 1 {
		t.Errorf("esperava UpdatePaymentRequestFailed chamado 1 vez, veio %d", db.markFailedCount())
	}
}

// e) TestHandle_DoesNotCallGatewayAfterExhausted: já existe um attempt com status='failed' e attempt_number=3.
// Simula a mensagem voltando para a fila após as tentativas terem sido esgotadas em execução anterior.
func TestHandle_DoesNotCallGatewayAfterExhausted(t *testing.T) {
	db := newFakePaymentQueries()
	// Pré-popula attempt_number = 3 com status failed (maxCreatePaymentAttempts esgotado)
	db.seedExistingAttempt(uuidKey(), "failed", int32(maxCreatePaymentAttempts), false, "")

	gw := &fakeGateway{}
	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	err := p.Handle(context.Background(), d)
	if err != nil {
		t.Errorf("esperava Handle retornar nil, veio: %v", err)
	}

	if gw.CallCount() != 0 {
		t.Errorf("esperava CreatePayment NÃO ser chamado (tentativas já esgotadas), mas foi chamado %d vez(es)", gw.CallCount())
	}

	if db.reqStatus(uuidKey()) != "failed" {
		t.Errorf("esperava status de payment_requests = 'failed', veio %q", db.reqStatus(uuidKey()))
	}
	if db.markFailedCount() != 1 {
		t.Errorf("esperava UpdatePaymentRequestFailed chamado 1 vez (para fechar o payment_request), veio %d", db.markFailedCount())
	}
}

// f) TestHandle_SkipsWhenAlreadySucceeded: já existe um attempt com gateway_payment_id válido e status != 'failed'.
// Simula idempotência quando mensagem duplicada chega após pagamento já ter sido iniciado/aprovado.
func TestHandle_SkipsWhenAlreadySucceeded(t *testing.T) {
	db := newFakePaymentQueries()
	// Pré-popula tentativa existente com gateway_payment_id válido e status non-failed
	db.seedExistingAttempt(uuidKey(), "pending", 1, true, "gw-already-created")

	gw := &fakeGateway{}
	p := NewPaymentRequestedProcessor(db, gw, testConfig())
	d := makeDelivery(testPaymentUUID)

	err := p.Handle(context.Background(), d)
	if err != nil {
		t.Errorf("esperava Handle retornar nil (ignora mensagem duplicada), veio: %v", err)
	}

	if gw.CallCount() != 0 {
		t.Errorf("esperava CreatePayment NÃO ser chamado (já possui attempt de sucesso/pending), mas foi chamado %d vez(es)", gw.CallCount())
	}

	if db.attemptCount() != 0 {
		t.Errorf("não esperava novos payment_attempts salvos, vieram %d", db.attemptCount())
	}
}

// g) TestHandle_MalformedMessageIsDiscarded: JSON inválido no body da mensagem.
func TestHandle_MalformedMessageIsDiscarded(t *testing.T) {
	db := newFakePaymentQueries()
	gw := &fakeGateway{}
	p := NewPaymentRequestedProcessor(db, gw, testConfig())

	d := amqp.Delivery{Body: []byte("invalid json payload {")}

	err := p.Handle(context.Background(), d)
	if err != nil {
		t.Errorf("esperava Handle retornar nil (descartar mensagem malformada sem reentregar), veio: %v", err)
	}

	if gw.CallCount() != 0 {
		t.Errorf("esperava CreatePayment NÃO ser chamado para mensagem malformada, mas foi chamado %d vez(es)", gw.CallCount())
	}

	if db.attemptCount() != 0 {
		t.Errorf("não esperava payment_attempts salvos para mensagem malformada, veio %d", db.attemptCount())
	}
}
