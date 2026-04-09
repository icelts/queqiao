package handler

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func requireIdempotencyKey(c *gin.Context) bool {
	if strings.TrimSpace(c.GetHeader("Idempotency-Key")) != "" {
		return true
	}
	response.ErrorFrom(c, service.ErrIdempotencyKeyRequired)
	return false
}

func markOrderFailedOnPaymentInitialization(
	ctx context.Context,
	referralService *service.ReferralService,
	orderNo string,
	channel string,
	initErr error,
) error {
	if referralService == nil || strings.TrimSpace(orderNo) == "" {
		return initErr
	}

	note := "payment initialization failed"
	if initErr != nil {
		message := strings.TrimSpace(infraerrors.Message(initErr))
		if message == "" {
			message = strings.TrimSpace(initErr.Error())
		}
		if message != "" {
			note += ": " + message
		}
	}

	if _, markErr := referralService.MarkRechargeOrderFailed(ctx, orderNo, channel, "", "", "", note); markErr != nil {
		return infraerrors.InternalServer(
			"PAYMENT_INIT_CLEANUP_FAILED",
			"payment initialization failed and order cleanup failed",
		).WithCause(markErr).WithMetadata(map[string]string{
			"order_no": strings.TrimSpace(orderNo),
			"channel":  strings.TrimSpace(channel),
		})
	}

	return initErr
}
