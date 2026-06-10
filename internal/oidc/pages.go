package oidc

import (
	"html"
)

// loginPage holds the data rendered into the hosted login screen.
type loginPage struct {
	Action      string
	ClientName  string
	ClientLogo  string
	ClientHome  string
	Email       string
	Notice      string // shown in a coloured banner when set
	NoticeError bool   // true => red error banner, false => blue info banner
	ShowForm    bool   // false hides the form (access fully blocked)
}

func renderLoginPage(p loginPage) string {
	clientBanner := ""
	if p.ClientName != "" {
		logo := ""
		if p.ClientLogo != "" {
			logo = `<img src="` + html.EscapeString(p.ClientLogo) + `" alt="" class="client-logo" />`
		}
		home := ""
		if p.ClientHome != "" {
			home = `<a class="client-home" href="` + html.EscapeString(p.ClientHome) + `">` + html.EscapeString(p.ClientHome) + `</a>`
		}
		clientBanner = `<div class="client-banner">` + logo +
			`<div><strong>` + html.EscapeString(p.ClientName) + `</strong> uygulamasına giriş yapıyorsunuz` + home + `</div></div>`
	}

	notice := ""
	if p.Notice != "" {
		cls := "notice notice-info"
		if p.NoticeError {
			cls = "notice notice-error"
		}
		notice = `<div class="` + cls + `">` + html.EscapeString(p.Notice) + `</div>`
	}

	form := ""
	if p.ShowForm {
		form = `<form method="post" action="` + html.EscapeString(p.Action) + `">
        <div class="form-group">
          <label>E-posta adresi</label>
          <div class="input-container">
            <input type="email" name="email" placeholder="örn. kullanici@gopenid.local" value="` + html.EscapeString(p.Email) + `" required autofocus />
          </div>
        </div>
        <div class="form-group">
          <label>Parola</label>
          <div class="input-container">
            <input type="password" name="password" placeholder="••••••••" required />
          </div>
        </div>
        <div class="form-group">
          <label>Doğrulama kodu</label>
          <div class="input-container">
            <input type="text" name="totp_code" inputmode="numeric" autocomplete="one-time-code" placeholder="000000" />
          </div>
        </div>
        <button type="submit" class="btn-primary">
          <span>Giriş yap</span>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter">
            <line x1="5" y1="12" x2="19" y2="12"></line>
            <polyline points="12 5 19 12 12 19"></polyline>
          </svg>
        </button>
      </form>`
	}

	return loginShell(clientBanner + notice + form)
}

// loginShell wraps the dynamic login panel content with the static chrome.
func loginShell(panelContent string) string {
	return `<!doctype html>
<html lang="tr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>gOpenID Giriş</title>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@300;400;600&family=IBM+Plex+Mono&display=swap" rel="stylesheet">
  <style>` + loginCSS + `</style>
</head>
<body>
  <div class="container">
    <div class="info-panel">
      <div class="brand-section">
        <div class="logo-tag">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="square" stroke-linejoin="miter">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
          </svg>
          gOpenID
        </div>
        <div class="brand-title">Kimlik Sunucusu<span>Güvenli giriş</span></div>
      </div>
      <div class="info-content">
        <h2>Tek oturum açma yetkilendirmesi</h2>
        <p>Güvenli bir uygulamaya erişiyorsunuz. Devam etmek için kurumsal dizin hesabınızla giriş yapın.</p>
        <ul class="features-list">
          <li class="feature-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>
            <div><strong>Güvenli kimlik doğrulama</strong><span>OAuth 2.0 üzerinde OpenID Connect 1.0 standartlarıyla korunur.</span></div>
          </li>
          <li class="feature-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><line x1="9" y1="3" x2="9" y2="21"></line></svg>
            <div><strong>Merkezi dizin kontrolü</strong><span>Profil bilgileriniz ve rolleriniz bu uygulama için dinamik olarak yetkilendirilir.</span></div>
          </li>
        </ul>
      </div>
      <div class="info-footer">&copy; 2026 gOpenID. Tüm hakları saklıdır. Kurumsal dizin konsolu.</div>
    </div>
    <div class="login-panel">
      <div class="login-header">
        <p>Kurumsal konsol</p>
        <h1>Giriş yap</h1>
      </div>
      ` + panelContent + `
      <div class="security-notice">
        <strong>Güvenlik notu:</strong>
        Bu kimlik doğrulama isteği uygulamayla kriptografik olarak ilişkilidir. Giriş yapmadan önce adres çubuğunda doğru issuer adresini kontrol edin.
      </div>
    </div>
  </div>
</body>
</html>`
}
