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

	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/store"
)

type Manager struct {
	db  *store.Store
	cfg config.Config
	key *rsa.PrivateKey
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

func New(db *store.Store, cfg config.Config) (*Manager, error) {
	m := &Manager{db: db, cfg: cfg}
	key, err := m.loadOrCreate()
	if err != nil {
		return nil, err
	}
	m.key = key
	return m, nil
}

func (m *Manager) PrivateKey() *rsa.PrivateKey {
	return m.key
}

func (m *Manager) PublicKey() *rsa.PublicKey {
	return &m.key.PublicKey
}

func (m *Manager) KeyID() string {
	return m.cfg.KeyID
}

func (m *Manager) JWKS() JWKSet {
	pub := m.PublicKey()
	return JWKSet{Keys: []JWK{{
		Kty: "RSA",
		Use: "sig",
		Kid: m.KeyID(),
		Alg: "RS256",
		N:   b64(pub.N.Bytes()),
		E:   b64(big.NewInt(int64(pub.E)).Bytes()),
	}}}
}

func (m *Manager) loadOrCreate() (*rsa.PrivateKey, error) {
	row, err := m.db.GetSigningKey(context.Background(), m.cfg.KeyID)
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
	return key, m.db.CreateSigningKey(context.Background(), row)
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
