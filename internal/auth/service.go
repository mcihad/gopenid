package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/keys"
	"gopenid/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db   *store.Store
	cfg  config.Config
	keys *keys.Manager
}

func New(db *store.Store, cfg config.Config, keyManager *keys.Manager) *Service {
	return &Service{db: db, cfg: cfg, keys: keyManager}
}

func (s *Service) Authenticate(email, password string) (domain.User, error) {
	user, err := s.db.GetUserByEmail(context.Background(), email)
	if err != nil {
		return user, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return user, errors.New("invalid credentials")
	}
	return user, nil
}

func (s *Service) AccessToken(user domain.User, clientID string, scope string) (string, error) {
	return s.token(user, clientID, scope, "", "access")
}

func (s *Service) IDToken(user domain.User, clientID string, scope string, nonce string) (string, error) {
	return s.token(user, clientID, scope, nonce, "id")
}

func (s *Service) token(user domain.User, clientID string, scope string, nonce string, tokenUse string) (string, error) {
	now := time.Now()
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}
	aud := "gopenid"
	if clientID != "" {
		aud = clientID
	}
	claims := jwt.MapClaims{
		"iss":       s.cfg.Issuer,
		"sub":       user.Email,
		"aud":       aud,
		"iat":       now.Unix(),
		"exp":       now.Add(s.cfg.TokenTTL).Unix(),
		"email":     user.Email,
		"name":      user.Name,
		"roles":     roles,
		"uid":       user.ID,
		"token_use": tokenUse,
	}
	if user.DepartmentID != nil {
		claims["department"] = user.Department.Name
	}
	if scope != "" {
		claims["scope"] = scope
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}

	if scopeContains(scope, "client_roles") {
		clientRoles := user.ClientRoles
		if len(clientRoles) > 0 {
			resourceAccess := make(map[string]map[string][]string)
			var specificRoles []string
			for _, cr := range clientRoles {
				for _, client := range user.AuthorizedClients {
					if client.ID != cr.ClientID {
						continue
					}
					if _, ok := resourceAccess[client.ClientID]; !ok {
						resourceAccess[client.ClientID] = map[string][]string{"roles": {}}
					}
					resourceAccess[client.ClientID]["roles"] = append(resourceAccess[client.ClientID]["roles"], cr.Name)
					if client.ClientID == clientID {
						specificRoles = append(specificRoles, cr.Name)
					}
					break
				}
			}
			if len(resourceAccess) > 0 {
				claims["resource_access"] = resourceAccess
			}
			if len(specificRoles) > 0 {
				claims["client_roles"] = specificRoles
			}
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keys.KeyID()
	return token.SignedString(s.keys.PrivateKey())
}

func scopeContains(scope string, needle string) bool {
	for _, item := range strings.Fields(scope) {
		if item == needle {
			return true
		}
	}
	return false
}

func RandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
