package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/keys"
	"gopenid/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel authentication errors so callers can produce friendly messages.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUserBlocked        = errors.New("user is blocked")
	ErrUserLocked         = errors.New("user is temporarily locked")
	ErrMFACodeRequired    = errors.New("mfa code is required")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

type Service struct {
	db   *store.Store
	cfg  config.Config
	keys *keys.Manager
}

func New(db *store.Store, cfg config.Config, keyManager *keys.Manager) *Service {
	return &Service{db: db, cfg: cfg, keys: keyManager}
}

func (s *Service) Config() config.Config { return s.cfg }

// Authenticate verifies credentials and account state.
func (s *Service) Authenticate(ctx context.Context, email, password, totpCode string) (domain.User, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return user, ErrInvalidCredentials
	}
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return user, ErrUserLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		_ = s.db.RecordFailedLogin(ctx, user.ID, s.cfg.MaxLoginFailures, s.cfg.LoginLockout.String())
		return user, ErrInvalidCredentials
	}
	if !user.Active {
		return user, ErrUserInactive
	}
	if user.Blocked {
		return user, ErrUserBlocked
	}
	if user.MFAEnabled && !VerifyTOTP(user.TOTPSecret, totpCode, time.Now()) {
		return user, ErrMFACodeRequired
	}
	return user, nil
}

// Tokens bundles the credentials returned by a successful token exchange.
type Tokens struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int
	Scope        string
}

type TokenOptions struct {
	AuthTime *time.Time
	Code     string
}

// accessTTL returns the access-token lifetime for a client, falling back to the
// global default when the client does not override it.
func (s *Service) accessTTL(client domain.Client) time.Duration {
	if client.TokenTTLSeconds > 0 {
		return time.Duration(client.TokenTTLSeconds) * time.Second
	}
	return s.cfg.TokenTTL
}

func (s *Service) refreshTTL(client domain.Client) time.Duration {
	if client.RefreshTTLSeconds > 0 {
		return time.Duration(client.RefreshTTLSeconds) * time.Second
	}
	return s.cfg.RefreshTTL
}

// IssueTokens creates an access token (and optionally an id and refresh token)
// for a user/client pair. clientID is the public client identifier; client may
// be the zero value for first-party logins.
func (s *Service) IssueTokens(ctx context.Context, user domain.User, client domain.Client, clientID, scope, nonce string, withRefresh bool, options ...TokenOptions) (Tokens, error) {
	ttl := s.accessTTL(client)
	opts := TokenOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	access, _, err := s.signToken(user, clientID, scope, "", "access", ttl, nil)
	if err != nil {
		return Tokens{}, err
	}
	out := Tokens{AccessToken: access, ExpiresIn: int(ttl.Seconds()), Scope: strings.TrimSpace(scope)}
	if scopeContains(scope, "openid") {
		extra := jwt.MapClaims{"at_hash": oidcHash(access)}
		if opts.Code != "" {
			extra["c_hash"] = oidcHash(opts.Code)
		}
		authTime := opts.AuthTime
		if authTime == nil {
			authTime = user.LastLoginAt
		}
		if authTime == nil {
			now := time.Now()
			authTime = &now
		}
		extra["auth_time"] = authTime.Unix()
		idToken, _, err := s.signToken(user, clientID, scope, nonce, "id", ttl, extra)
		if err != nil {
			return Tokens{}, err
		}
		out.IDToken = idToken
	}
	if withRefresh {
		refresh, err := s.issueRefreshToken(ctx, user.ID, clientID, scope, s.refreshTTL(client))
		if err != nil {
			return Tokens{}, err
		}
		out.RefreshToken = refresh
	}
	return out, nil
}

// AccessToken issues a standalone access token (first-party login).
func (s *Service) AccessToken(user domain.User, clientID, scope string) (string, error) {
	token, _, err := s.signToken(user, clientID, scope, "", "access", s.cfg.TokenTTL, nil)
	return token, err
}

func (s *Service) signToken(user domain.User, clientID, scope, nonce, tokenUse string, ttl time.Duration, extra jwt.MapClaims) (string, string, error) {
	now := time.Now()
	jti := uuid.NewString()
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
		"exp":       now.Add(ttl).Unix(),
		"jti":       jti,
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
		applyClientRoleClaims(claims, user, clientID)
	}
	for key, value := range extra {
		claims[key] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keys.KeyID()
	signed, err := token.SignedString(s.keys.PrivateKey())
	return signed, jti, err
}

func applyClientRoleClaims(claims jwt.MapClaims, user domain.User, clientID string) {
	if len(user.ClientRoles) == 0 {
		return
	}
	resourceAccess := make(map[string]map[string][]string)
	var specificRoles []string
	for _, cr := range user.ClientRoles {
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

// Verify parses and validates a bearer token, rejecting revoked tokens.
func (s *Service) Verify(ctx context.Context, tokenText string) (jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenText, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		kid, _ := token.Header["kid"].(string)
		return s.keys.PublicKeyFor(kid), nil
	}, jwt.WithIssuer(s.cfg.Issuer))
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if jti, _ := claims["jti"].(string); jti != "" {
		revoked, err := s.db.IsAccessTokenRevoked(ctx, jti)
		if err != nil {
			return nil, err
		}
		if revoked {
			return nil, ErrTokenRevoked
		}
	}
	return claims, nil
}

// RevokeAccessClaims blacklists an access token until its natural expiry and
// revokes the associated refresh tokens for the user.
func (s *Service) RevokeAccessClaims(ctx context.Context, claims jwt.MapClaims, reason string) error {
	jti, _ := claims["jti"].(string)
	uid := int64(0)
	if f, ok := claims["uid"].(float64); ok {
		uid = int64(f)
	}
	exp := time.Now().Add(s.cfg.TokenTTL)
	if e, ok := claims["exp"].(float64); ok {
		exp = time.Unix(int64(e), 0)
	}
	if err := s.db.RevokeAccessToken(ctx, jti, uid, reason, exp); err != nil {
		return err
	}
	if uid > 0 {
		return s.db.RevokeAllUserRefreshTokens(ctx, uid)
	}
	return nil
}

// issueRefreshToken generates an opaque refresh token and stores its hash.
func (s *Service) issueRefreshToken(ctx context.Context, userID int64, clientID, scope string, ttl time.Duration) (string, error) {
	raw, err := RandomToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.db.CreateRefreshToken(ctx, domain.RefreshToken{
		TokenHash: HashToken(raw),
		UserID:    userID,
		ClientID:  clientID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(ttl),
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

// ConsumeRefreshToken validates a refresh token and returns the owning user.
// The presented token is rotated: the old one is revoked.
func (s *Service) ConsumeRefreshToken(ctx context.Context, raw string) (domain.User, domain.RefreshToken, error) {
	stored, err := s.db.GetRefreshTokenByHash(ctx, HashToken(raw))
	if err != nil {
		return domain.User{}, stored, errors.New("invalid refresh token")
	}
	if stored.Revoked || time.Now().After(stored.ExpiresAt) {
		return domain.User{}, stored, errors.New("refresh token expired or revoked")
	}
	user, err := s.db.GetUser(ctx, stored.UserID)
	if err != nil {
		return domain.User{}, stored, errors.New("invalid refresh token")
	}
	if !user.Active || user.Blocked {
		return domain.User{}, stored, errors.New("user not allowed")
	}
	if err := s.db.RevokeRefreshTokenByHash(ctx, stored.TokenHash); err != nil {
		return domain.User{}, stored, err
	}
	return user, stored, nil
}

func scopeContains(scope, needle string) bool {
	for _, item := range strings.Fields(scope) {
		if item == needle {
			return true
		}
	}
	return false
}

func ScopeContains(scope, needle string) bool {
	return scopeContains(scope, needle)
}

func oidcHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
}

// RandomToken returns a URL-safe random string of the given byte size.
func RandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken returns a hex SHA-256 digest used to store opaque tokens at rest.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
