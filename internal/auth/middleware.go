package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gopenid/internal/httpx"
)

const claimsLocalKey = "auth.claims"

// RequireBearer validates the Authorization bearer token (signature, issuer and
// revocation) and stores the resulting claims in the request locals.
func RequireBearer(svc *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenText := BearerToken(c)
		if tokenText == "" {
			return httpx.Error(c, fiber.StatusUnauthorized, "missing bearer token")
		}
		claims, err := svc.Verify(c.Context(), tokenText)
		if err != nil {
			if err == ErrTokenRevoked {
				return httpx.Error(c, fiber.StatusUnauthorized, "token has been revoked")
			}
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
		}
		c.Locals(claimsLocalKey, claims)
		return c.Next()
	}
}

func RequireRole(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, ok := ClaimsFromCtx(c)
		if !ok {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
		}
		if roles, ok := claims["roles"].([]any); ok {
			for _, item := range roles {
				if item == role {
					return c.Next()
				}
			}
		}
		return httpx.Error(c, fiber.StatusForbidden, "insufficient role")
	}
}

// BearerToken extracts the raw token from the Authorization header.
func BearerToken(c fiber.Ctx) string {
	header := c.Get("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if token == header {
		return ""
	}
	return strings.TrimSpace(token)
}

// ClaimsFromCtx returns the validated claims stored by RequireBearer.
func ClaimsFromCtx(c fiber.Ctx) (jwt.MapClaims, bool) {
	claims, ok := c.Locals(claimsLocalKey).(jwt.MapClaims)
	return claims, ok
}

// UserIDFromCtx extracts the authenticated user id from the request locals.
func UserIDFromCtx(c fiber.Ctx) (int64, bool) {
	claims, ok := ClaimsFromCtx(c)
	if !ok {
		return 0, false
	}
	if uid, ok := claims["uid"].(float64); ok {
		return int64(uid), true
	}
	return 0, false
}
