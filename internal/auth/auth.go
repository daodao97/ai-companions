package auth

import (
	"context"
	"net/http"
	"strings"
	"companions/internal/conf"

	"github.com/daodao97/xgo/xdb"
	"github.com/daodao97/xgo/xjwt"
	"github.com/gin-gonic/gin"
)

type authContextKey struct{}

// gin middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// remove Bearer prefix
		token = strings.TrimPrefix(token, "Bearer ")

		payload, err := xjwt.VerifyHMacToken(token, conf.Get().JwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		// 将 payload 添加到请求上下文
		ctx := context.WithValue(c.Request.Context(), authContextKey{}, xdb.Record(payload))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func GetAuthFromContext(ctx context.Context) xdb.Record {
	if _ctx, ok := ctx.(*gin.Context); ok {
		return GetAuth(_ctx)
	}
	traceId, _ := ctx.Value(authContextKey{}).(xdb.Record)
	return traceId
}

func GetAuth(c *gin.Context) xdb.Record {
	return GetAuthFromContext(c.Request.Context())
}
