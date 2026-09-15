// payment-consumer\internal\infra\paymentgateway\erros.go
package paymentgateway

import (
	"errors"
	"net"

	"github.com/mercadopago/sdk-go/pkg/mperror"
)

func IsRetryableGatewayError(err error) bool {
	if err == nil {
		return false
	}

	var mpErr *mperror.ResponseError
	if errors.As(err, &mpErr) {
		switch mpErr.StatusCode {
		case 429, 423, 424:
			return true
		}
		return mpErr.StatusCode >= 500
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}
