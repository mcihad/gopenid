package oidc

import "html"

func authForm(action string) string {
	return `<!doctype html>
<html lang="tr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>gOpenID Giriş</title>
  <link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@300;400;600&family=IBM+Plex+Mono&display=swap" rel="stylesheet">
  <style>
    :root {
      --font-sans: 'IBM Plex Sans', -apple-system, sans-serif;
      --font-mono: 'IBM Plex Mono', monospace;
      --colors-primary: #0f62fe;
      --colors-primary-hover: #0353e9;
      --colors-ink: #161616;
      --colors-ink-muted: #525252;
      --colors-ink-subtle: #8c8c8c;
      --colors-canvas: #ffffff;
      --colors-surface-1: #f4f4f4;
      --colors-surface-2: #e0e0e0;
      --colors-hairline: #e0e0e0;
    }

    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
      border-radius: 0px !important; /* Force IBM Carbon flat-square geometry */
    }

    body {
      background-color: var(--colors-canvas);
      color: var(--colors-ink);
      font-family: var(--font-sans);
      min-height: 100vh;
      display: flex;
    }

    /* Layout: Split-screen */
    .container {
      display: flex;
      width: 100%;
      min-height: 100vh;
    }

    /* Left panel: Info */
    .info-panel {
      flex: 1;
      background-color: var(--colors-ink);
      color: #ffffff;
      padding: 64px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      border-right: 1px solid var(--colors-hairline);
    }

    /* Right panel: Login box */
    .login-panel {
      width: 520px;
      background-color: var(--colors-canvas);
      padding: 64px;
      display: flex;
      flex-direction: column;
      justify-content: center;
    }

    @media (max-width: 900px) {
      .container {
        flex-direction: column;
      }
      .info-panel {
        padding: 40px;
        min-height: 250px;
      }
      .login-panel {
        width: 100%;
        padding: 40px;
      }
    }

    .brand-section {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .logo-tag {
      font-size: 14px;
      font-weight: 700;
      color: var(--colors-ink);
      background-color: #ffffff;
      padding: 4px 8px;
      letter-spacing: 0.5px;
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }

    .brand-title {
      font-size: 14px;
      font-weight: 600;
      letter-spacing: 0.16px;
    }

    .brand-title span {
      color: var(--colors-primary);
      font-weight: 400;
      margin-left: 4px;
    }

    .info-content {
      margin-top: 48px;
    }

    .info-content h2 {
      font-size: 32px;
      font-weight: 300;
      line-height: 1.25;
      margin-bottom: 24px;
    }

    .info-content p {
      font-size: 15px;
      color: var(--colors-ink-subtle);
      line-height: 1.6;
      margin-bottom: 32px;
    }

    .features-list {
      list-style: none;
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    .feature-item {
      display: flex;
      align-items: flex-start;
      gap: 12px;
      font-size: 14px;
      color: #ffffff;
    }

    .feature-item svg {
      color: var(--colors-primary);
      flex-shrink: 0;
      margin-top: 2px;
    }

    .feature-item div strong {
      display: block;
      margin-bottom: 2px;
    }

    .feature-item div span {
      color: var(--colors-ink-subtle);
    }

    .info-footer {
      font-size: 12px;
      color: var(--colors-ink-muted);
      margin-top: 48px;
    }

    /* Form Styles */
    .login-header {
      margin-bottom: 32px;
    }

    .login-header p {
      font-size: 14px;
      color: var(--colors-ink-muted);
      margin-bottom: 4px;
    }

    .login-header h1 {
      font-size: 28px;
      font-weight: 300;
      color: var(--colors-ink);
    }

    .form-group {
      margin-bottom: 24px;
      display: flex;
      flex-direction: column;
    }

    .form-group label {
      font-size: 12px;
      font-weight: 400;
      color: var(--colors-ink-muted);
      margin-bottom: 8px;
    }

    .input-container {
      position: relative;
      background-color: var(--colors-surface-1);
    }

    .input-container input {
      width: 100%;
      min-height: 40px;
      background-color: transparent;
      border: 0;
      border-bottom: 1px solid var(--colors-ink-subtle);
      color: var(--colors-ink);
      font-family: var(--font-sans);
      font-size: 14px;
      padding: 10px 16px;
      outline: none;
      transition: all 0.15s ease;
    }

    .input-container input:focus {
      border-bottom: 2px solid var(--colors-primary);
    }

    .btn-primary {
      width: 100%;
      min-height: 48px;
      background-color: var(--colors-primary);
      color: #ffffff;
      border: 0;
      font-family: var(--font-sans);
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 16px;
      margin-top: 8px;
      transition: background-color 0.15s ease;
    }

    .btn-primary:hover {
      background-color: var(--colors-primary-hover);
    }

    .btn-primary svg {
      margin-left: 8px;
    }

    .security-notice {
      margin-top: 32px;
      padding: 16px;
      background-color: var(--colors-surface-1);
      border-left: 3px solid var(--colors-primary);
      font-size: 12px;
      line-height: 1.5;
      color: var(--colors-ink-muted);
    }
  </style>
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
        <div class="brand-title">
          Kimlik Sunucusu<span>Güvenli giriş</span>
        </div>
      </div>

      <div class="info-content">
        <h2>Tek oturum açma yetkilendirmesi</h2>
        <p>Güvenli bir uygulamaya erişiyorsunuz. Devam etmek ve bu bağlantıyı yetkilendirmek için kurumsal dizin hesabınızla giriş yapın.</p>
        
        <ul class="features-list">
          <li class="feature-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
            </svg>
            <div>
              <strong>Güvenli kimlik doğrulama</strong>
              <span>OAuth 2.0 üzerinde OpenID Connect 1.0 standartlarıyla korunur.</span>
            </div>
          </li>
          <li class="feature-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="9" y1="3" x2="9" y2="21"></line>
            </svg>
            <div>
              <strong>Merkezi dizin kontrolü</strong>
              <span>Profil bilgileriniz ve rolleriniz bu client için dinamik olarak yetkilendirilir.</span>
            </div>
          </li>
        </ul>
      </div>

      <div class="info-footer">
        &copy; 2026 gOpenID. Tüm hakları saklıdır. Kurumsal dizin konsolu.
      </div>
    </div>

    <div class="login-panel">
      <div class="login-header">
        <p>Kurumsal konsol</p>
        <h1>Giriş yap</h1>
      </div>

      <form method="post" action="` + html.EscapeString(action) + `">
        <div class="form-group">
          <label>E-posta adresi</label>
          <div class="input-container">
            <input 
              type="email" 
              name="email" 
              placeholder="örn. kullanici@gopenid.local" 
              required 
              autofocus
            />
          </div>
        </div>

        <div class="form-group">
          <label>Parola</label>
          <div class="input-container">
            <input 
              type="password" 
              name="password" 
              placeholder="••••••••" 
              required
            />
          </div>
        </div>

        <button type="submit" class="btn-primary">
          <span>Giriş yap</span>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="miter">
            <line x1="5" y1="12" x2="19" y2="12"></line>
            <polyline points="12 5 19 12 12 19"></polyline>
          </svg>
        </button>
      </form>

      <div class="security-notice">
        <strong>Güvenlik notu:</strong>
        Bu kimlik doğrulama isteği client uygulamasıyla kriptografik olarak ilişkilidir. Giriş yapmadan önce adres çubuğunda doğru issuer adresinin göründüğünü kontrol edin.
      </div>
    </div>
  </div>
</body>
</html>`
}
