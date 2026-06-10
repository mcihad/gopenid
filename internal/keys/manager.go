package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"sync"
	"time"

	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/store"
)

type Manager struct {
	db      *store.Store
	cfg     config.Config
	key     *rsa.PrivateKey
	keyByID map[string]*rsa.PrivateKey
	mu      sync.RWMutex
}

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func New(ctx context.Context, db *store.Store, cfg config.Config) (*Manager, error) {
	m := &Manager{db: db, cfg: cfg}
	key, err := m.loadOrCreate(ctx)
	if err != nil {
		return nil, err
	}
	m.key = key
	if err := m.loadActiveKeys(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) PrivateKey() *rsa.PrivateKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.key
}

func (m *Manager) PublicKey() *rsa.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &m.key.PublicKey
}

func (m *Manager) PublicKeyFor(kid string) *rsa.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if key, ok := m.keyByID[kid]; ok {
		return &key.PublicKey
	}
	return &m.key.PublicKey
}

func (m *Manager) KeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.KeyID
}

func (m *Manager) JWKS() JWKSet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]JWK, 0, len(m.keyByID))
	for kid, key := range m.keyByID {
		pub := &key.PublicKey
		keys = append(keys, JWK{
			Kty: "RSA",
			Use: "sig",
			Kid: kid,
			Alg: "RS256",
			N:   b64(pub.N.Bytes()),
			E:   b64(big.NewInt(int64(pub.E)).Bytes()),
		})
	}
	return JWKSet{Keys: keys}
}

func (m *Manager) Rotate(ctx context.Context) (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	pemText, err := encodePrivateKey(key)
	if err != nil {
		return "", err
	}
	kid := "gopenid-rs256-" + time.Now().UTC().Format("20060102150405")
	if err := m.db.CreateSigningKey(ctx, domain.SigningKey{KeyID: kid, PrivatePEM: pemText, Active: true}); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.KeyID = kid
	m.key = key
	m.keyByID[kid] = key
	return kid, nil
}

func (m *Manager) loadOrCreate(ctx context.Context) (*rsa.PrivateKey, error) {
	row, err := m.db.GetSigningKey(ctx, m.cfg.KeyID)
	if err == nil {
		return parsePrivateKey(row.PrivatePEM)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pemText, err := encodePrivateKey(key)
	if err != nil {
		return nil, err
	}
	row = domain.SigningKey{KeyID: m.cfg.KeyID, PrivatePEM: pemText, Active: true}
	return key, m.db.CreateSigningKey(ctx, row)
}

func (m *Manager) loadActiveKeys(ctx context.Context) error {
	rows, err := m.db.ListSigningKeys(ctx)
	if err != nil {
		return err
	}
	m.keyByID = make(map[string]*rsa.PrivateKey, len(rows)+1)
	for _, row := range rows {
		key, err := parsePrivateKey(row.PrivatePEM)
		if err != nil {
			return err
		}
		m.keyByID[row.KeyID] = key
	}
	m.keyByID[m.cfg.KeyID] = m.key
	return nil
}

func encodePrivateKey(key *rsa.PrivateKey) (string, error) {
	bytes := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: bytes})), nil
}

func parsePrivateKey(text string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return nil, errors.New("invalid signing key pem")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func b64(bytes []byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes)
}
