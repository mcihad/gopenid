package auth

import (
	"strings"

	"gopenid/internal/config"
	"gopenid/internal/httpx"
	"gopenid/internal/keys"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func RequireBearer(cfg config.Config, keyManager *keys.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		tokenText := strings.TrimPrefix(header, "Bearer ")
		if tokenText == header || tokenText == "" {
			return httpx.Error(c, fiber.StatusUnauthorized, "missing bearer token")
		}
		token, err := jwt.Parse(tokenText, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return keyManager.PublicKey(), nil
		}, jwt.WithIssuer(cfg.Issuer))
		if err != nil || !token.Valid {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
		}
		return c.Next()
	}
}
