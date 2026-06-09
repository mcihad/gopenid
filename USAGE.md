# GOpenID Kullanım Kılavuzu

GOpenID, Go/Fiber tabanlı bir OpenID Connect authorization server ve kullanıcı yönetim panelidir. Kullanıcılar, departmanlar, roller, OIDC client kayıtları ve signing key kayıtları PostgreSQL üzerinde pgx ile tutulur; migration işlemleri `github.com/mcihad/pgxmigrate` ile çalışır. React arayüzü production build sırasında Go binary içine gömülür.

## Çalıştırma

Geliştirme için:

```bash
GOCACHE=/tmp/gocache go run ./cmd/server
```

Varsayılan adres:

```text
http://localhost:8080
```

Varsayılan seed admin kullanıcısı:

```text
Email: admin@gopenid.local
Password: admin12345
```

Önemli ortam değişkenleri:

```bash
GOPENID_ADDR=":8080"
GOPENID_ISSUER="http://localhost:8080"
GOPENID_KEY_ID="gopenid-rs256-1"
GOPENID_DEV_SEED="true"
GOPENID_ADMIN_EMAIL="admin@gopenid.local"
GOPENID_ADMIN_PASS="admin12345"

DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="postgres"
DB_SCHEMA="auth"
DB_SSLMODE="disable"
```

Uygulama açılışta `DB_SCHEMA` değerini oluşturur, PostgreSQL `search_path` değerini o şemaya ayarlar ve `migrations/` altındaki SQL dosyalarını pgxmigrate ile uygular. Varsayılan şema `auth`; `public` kullanmak için `DB_SCHEMA=public` verilebilir.

Frontend build çıktısını Go binary içine gömmek için:

```bash
npm run build
GOCACHE=/tmp/gocache go build ./cmd/server
```

## Yönetim Arayüzü

Tarayıcıdan şu adrese gidin:

```text
http://localhost:8080/
```

Admin panelinde şu kaynaklar yönetilir:

- Users
- Departments
- Roles

Login başarılı olduğunda React arayüzü JWT token’ı tarayıcı `localStorage` içinde saklar ve admin API isteklerinde `Authorization: Bearer <token>` olarak gönderir.

## Direct JWT API Kullanımı

Bu akış, kendi uygulamanızın kullanıcı adı/parola ile doğrudan RS256 imzalı JWT alması içindir.

Login isteği:

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@gopenid.local","password":"admin12345"}'
```

Örnek cevap:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 28800,
  "token_type": "Bearer"
}
```

Korumalı admin API kullanımı:

```bash
TOKEN="eyJhbGciOiJIUzI1NiIs..."

curl -s http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

## Admin API Endpointleri

Tüm `/api/admin/*` endpointleri Bearer token ister.

Departmanlar:

```text
GET    /api/admin/departments
POST   /api/admin/departments
GET    /api/admin/departments/:id
PUT    /api/admin/departments/:id
DELETE /api/admin/departments/:id
```

Departman oluşturma:

```bash
curl -s -X POST http://localhost:8080/api/admin/departments \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Engineering","description":"Product and platform teams"}'
```

Roller:

```text
GET    /api/admin/roles
POST   /api/admin/roles
GET    /api/admin/roles/:id
PUT    /api/admin/roles/:id
DELETE /api/admin/roles/:id
```

Rol oluşturma:

```bash
curl -s -X POST http://localhost:8080/api/admin/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"operator","description":"Operational access"}'
```

Kullanıcılar:

```text
GET    /api/admin/users
POST   /api/admin/users
GET    /api/admin/users/:id
PUT    /api/admin/users/:id
DELETE /api/admin/users/:id
```

Kullanıcı oluşturma:

```bash
curl -s -X POST http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "email":"user@example.test",
    "name":"Example User",
    "password":"strong-password",
    "active":true,
    "departmentId":1,
    "roleIds":[1]
  }'
```

Kullanıcı güncellemede `password` boş gönderilirse mevcut parola korunur.

## Normal Authorization Code Flow

Bu akış, bir web uygulamasının kullanıcıyı GOpenID login ekranına yönlendirmesi ve callback üzerinden `code` alması içindir.

Discovery endpoint:

```bash
curl -s http://localhost:8080/.well-known/openid-configuration
```

Public JWKS endpoint:

```bash
curl -s http://localhost:8080/.well-known/jwks.json
```

Kullanıcıyı authorize endpointine yönlendirin:

```text
http://localhost:8080/oauth/authorize?response_type=code&client_id=my-client&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fcallback&scope=openid%20profile%20email&state=random-state&nonce=random-nonce&code_challenge=<S256_CHALLENGE>&code_challenge_method=S256
```

Kullanıcı login olduktan sonra GOpenID şu adrese döner:

```text
http://localhost:3000/callback?code=<AUTH_CODE>&state=random-state
```

Code ile token alma:

```bash
curl -s -X POST http://localhost:8080/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code' \
  -d 'code=<AUTH_CODE>' \
  -d 'redirect_uri=http://localhost:3000/callback' \
  -d 'client_id=my-client' \
  -d 'client_secret=my-secret' \
  -d 'code_verifier=<PKCE_VERIFIER>'
```

Örnek cevap:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "id_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 28800,
  "token_type": "Bearer"
}
```

`access_token` ve `id_token` RS256 ile imzalanır. Token header içinde `kid` bulunur; consumer uygulamalar public key’i `jwks_uri` üzerinden alıp imzayı doğrulamalıdır.

## Password Grant

Backend-to-backend veya güvenilir legacy client senaryolarında `/oauth/token` password grant ile de kullanılabilir.

```bash
curl -s -X POST http://localhost:8080/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=password' \
  -d 'username=admin@gopenid.local' \
  -d 'password=admin12345' \
  -d 'scope=openid profile email roles client_roles'
```

## JWT İçeriği

Token içinde şu claim’ler bulunur:

```text
iss, sub, aud, iat, exp, email, name, roles, uid, department
```

Kullanıcının sistem rolleri her zaman `roles` claim’i içinde yer alır. Client rolleri yalnızca istenen scope içinde `client_roles` varsa aktif client için `client_roles` claim’i olarak eklenir.

Varsayılan issuer:

```text
http://localhost:8080
```

Admin API middleware’i `iss` değerini ve RS256 imzasını aktif public key ile doğrular.

## Üretim Notları

Canlı ortamdan önce en az şu değişiklikler yapılmalıdır:

- `GOPENID_DEV_SEED=false` kullanılmalı.
- Admin şifresi ortam değişkeninden verilmeli.
- Signing key rotasyonu planlanmalı.
- Authorization code flow için client kayıtları ve redirect URI allowlist kullanılmalı.
- HTTPS zorunlu hale getirilmeli.
