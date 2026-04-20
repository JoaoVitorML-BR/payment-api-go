package server

import (
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/gin-gonic/gin"
)

func Run(cfg *config.Config, router *gin.Engine) error {
	return router.Run(":" + cfg.Port)
}
