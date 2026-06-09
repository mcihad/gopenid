# gOpenID — Yol Haritası ve Sistem Değerlendirmesi

> Son güncelleme: 2026-06-09

Bu belge, gOpenID merkezi kimlik ve yetkilendirme sunucusunun mevcut durumunu
değerlendirir ve önceliklendirilmiş iyileştirme önerilerini listeler.

## Genel değerlendirme

Sağlam bir temel mevcut: katmanlı mimari (domain / store / service / handler),
RS256 JWT + JWKS, OIDC akışları (authorization_code + PKCE, password,
refresh_token), politika motoru, audit log, rate limiting, modüler React +
URL tabanlı routing. Çalışan ve test edilen bir merkezi kimlik sistemi. Ancak
"üretim hazır" demeden önce kapatılması gereken güvenlik ve olgunluk boşlukları
var.

## Kritik güvenlik açıkları (P0)

1. **Client secret'lar düz metin saklanıyor.** `internal/store/clients.go`
   secret'ı plaintext tutuyor ve API'den geri döndürüyor. Bunlar bcrypt/argon2
   ile hash'lenmeli; secret yalnızca oluşturma anında bir kez gösterilmeli.

2. **Admin API'de yetki (authorization) yok.** `internal/app/app.go`
   `/api/admin`'i sadece `RequireBearer` ile koruyor — geçerli token'ı olan
   herhangi bir kullanıcı admin işlemleri yapabilir. `admin` rolü kontrolü
   (RequireRole middleware) şart.

3. **Trusted proxy doğrulaması yok.** `ProxyHeader: X-Forwarded-For` ayarlı ama
   `EnableTrustedProxyCheck` yok. Saldırgan `X-Forwarded-For` header'ı uydurarak
   IP politikalarını ve rate limit'i atlatabilir.

4. **Password grant açık.** OAuth2 password grant deprecated ve phishing'e açık.
   En azından client bazlı bir flag ile kapatılabilir olmalı.

5. **Brute-force koruması zayıf.** Genel rate limit var ama hesap bazlı kilitleme
   (X başarısız denemeden sonra geçici blok) yok. `blocked` alanı manuel;
   otomatik kilitleme eklenmeli.

## Eksik önemli özellikler

- **id_token'da `at_hash`, `auth_time`, `c_hash`** yok — tam OIDC uyumu için.
- **`prompt`, `max_age`, `login_hint`** parametreleri ve cookie tabanlı gerçek
  SSO oturumu yok — her authorize'da yeniden login isteniyor.
- **RP-initiated logout** (`end_session_endpoint`) yok.
- **Scope bazlı claim filtreleme** zayıf — `profile`/`email` scope'ları
  userinfo'da gerçek bir kısıtlamaya yol açmıyor.
- **Anahtar rotasyonu** yok — tek imzalama anahtarı, süresiz. JWKS birden fazla
  anahtar destekleyip rotasyon yapmalı.
- **E-posta doğrulama / şifre sıfırlama** akışı yok.
- **2FA/MFA** (TOTP) yok — kurumsal IAM için neredeyse zorunlu.
- **Webhook/event yayını** yok — "kullanıcı bloklandı" gibi olayları dış
  sistemlere bildirme.

## Kod kalitesi ve operasyon

- **`context.Background()` her yerde** — handler'larda `c.Context()`
  kullanılmalı (iptal, timeout, trace propagation).
- **Yapısal loglama yok** — `log.Printf` yerine `slog` + request ID.
- **Metrics/health check yok** — `/healthz`, `/readyz`, Prometheus metrikleri.
- **Graceful shutdown yok** — SIGTERM'de bağlantıları düzgün kapatma yok.
- **CI/CD ve Dockerfile yok** — Makefile'a `docker`, GitHub Actions workflow.
- **DB testleri PostgreSQL'e bağımlı** — `testcontainers-go` ile izolasyon.
- **OpenAPI/Swagger yok** — admin API için şema üretimi.

## Frontend

- **Token sessiz yenileme yok** — access token süresi dolunca query'ler patlıyor;
  `refresh_token` ile otomatik yenileme interceptor'ı eklenmeli.
- **Hata sınırı (ErrorBoundary)** ve toast bildirim sistemi yok.
- **Rol bazlı UI gizleme** yok — admin olmayan kullanıcı tüm menüleri görüyor.
- **Bundle ~374 KB** — lazy route ile kod bölme.

## Önerilen öncelik sırası

| Öncelik | İş                                                                        |
| ------- | ------------------------------------------------------------------------- |
| P0      | Admin rol kontrolü middleware'i + client secret hash'leme + trusted proxy |
| P0      | Hesap bazlı brute-force kilitleme                                         |
| P1      | Anahtar rotasyonu, graceful shutdown, slog, healthz                       |
| P1      | Frontend sessiz token yenileme + rol bazlı menü                           |
| P2      | TOTP 2FA, şifre sıfırlama (e-posta), SSO cookie oturumu                   |
| P2      | Dockerfile + CI + OpenAPI                                                 |

En yüksek değer/risk oranı P0 güvenlik kalemlerinde: şu an geçerli token'ı olan
herkesin admin olabilmesi ve IP politikalarının header sahteciliğiyle
atlatılabilmesi, sistemin temel güvenlik vaadini bozuyor.
