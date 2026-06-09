package oidc

const loginCSS = `
:root {
  --font-sans: 'IBM Plex Sans', -apple-system, sans-serif;
  --colors-primary: #0f62fe;
  --colors-primary-hover: #0353e9;
  --colors-ink: #161616;
  --colors-ink-muted: #525252;
  --colors-ink-subtle: #8c8c8c;
  --colors-canvas: #ffffff;
  --colors-surface-1: #f4f4f4;
  --colors-hairline: #e0e0e0;
  --colors-error: #da1e28;
}
* { box-sizing: border-box; margin: 0; padding: 0; border-radius: 0 !important; }
body { background-color: var(--colors-canvas); color: var(--colors-ink); font-family: var(--font-sans); min-height: 100vh; display: flex; }
.container { display: flex; width: 100%; min-height: 100vh; }
.info-panel { flex: 1; background-color: var(--colors-ink); color: #fff; padding: 64px; display: flex; flex-direction: column; justify-content: space-between; border-right: 1px solid var(--colors-hairline); }
.login-panel { width: 520px; background-color: var(--colors-canvas); padding: 64px; display: flex; flex-direction: column; justify-content: center; }
@media (max-width: 900px) { .container { flex-direction: column; } .info-panel { padding: 40px; min-height: 200px; } .login-panel { width: 100%; padding: 40px; } }
.brand-section { display: flex; align-items: center; gap: 12px; }
.logo-tag { font-size: 14px; font-weight: 700; color: var(--colors-ink); background-color: #fff; padding: 4px 8px; letter-spacing: 0.5px; display: inline-flex; align-items: center; gap: 6px; }
.brand-title { font-size: 14px; font-weight: 600; }
.brand-title span { color: var(--colors-primary); font-weight: 400; margin-left: 4px; }
.info-content { margin-top: 48px; }
.info-content h2 { font-size: 32px; font-weight: 300; line-height: 1.25; margin-bottom: 24px; }
.info-content p { font-size: 15px; color: var(--colors-ink-subtle); line-height: 1.6; margin-bottom: 32px; }
.features-list { list-style: none; display: flex; flex-direction: column; gap: 16px; }
.feature-item { display: flex; align-items: flex-start; gap: 12px; font-size: 14px; color: #fff; }
.feature-item svg { color: var(--colors-primary); flex-shrink: 0; margin-top: 2px; }
.feature-item div strong { display: block; margin-bottom: 2px; }
.feature-item div span { color: var(--colors-ink-subtle); }
.info-footer { font-size: 12px; color: var(--colors-ink-muted); margin-top: 48px; }
.login-header { margin-bottom: 24px; }
.login-header p { font-size: 14px; color: var(--colors-ink-muted); margin-bottom: 4px; }
.login-header h1 { font-size: 28px; font-weight: 300; color: var(--colors-ink); }
.client-banner { display: flex; align-items: center; gap: 12px; padding: 14px 16px; margin-bottom: 20px; background-color: var(--colors-surface-1); border-left: 3px solid var(--colors-primary); font-size: 14px; color: var(--colors-ink-muted); }
.client-banner strong { color: var(--colors-ink); }
.client-banner .client-logo { width: 36px; height: 36px; object-fit: contain; }
.client-banner .client-home { display: block; font-size: 12px; color: var(--colors-primary); text-decoration: none; margin-top: 2px; }
.notice { padding: 14px 16px; margin-bottom: 20px; font-size: 13px; line-height: 1.5; border-left: 3px solid var(--colors-primary); background-color: var(--colors-surface-1); color: var(--colors-ink-muted); }
.notice-error { border-left-color: var(--colors-error); color: var(--colors-error); background-color: #fff1f1; }
.form-group { margin-bottom: 24px; display: flex; flex-direction: column; }
.form-group label { font-size: 12px; color: var(--colors-ink-muted); margin-bottom: 8px; }
.input-container { position: relative; background-color: var(--colors-surface-1); }
.input-container input { width: 100%; min-height: 40px; background-color: transparent; border: 0; border-bottom: 1px solid var(--colors-ink-subtle); color: var(--colors-ink); font-family: var(--font-sans); font-size: 14px; padding: 10px 16px; outline: none; transition: all 0.15s ease; }
.input-container input:focus { border-bottom: 2px solid var(--colors-primary); }
.btn-primary { width: 100%; min-height: 48px; background-color: var(--colors-primary); color: #fff; border: 0; font-family: var(--font-sans); font-size: 14px; font-weight: 600; cursor: pointer; display: inline-flex; align-items: center; justify-content: space-between; padding: 0 16px; margin-top: 8px; transition: background-color 0.15s ease; }
.btn-primary:hover { background-color: var(--colors-primary-hover); }
.security-notice { margin-top: 32px; padding: 16px; background-color: var(--colors-surface-1); border-left: 3px solid var(--colors-primary); font-size: 12px; line-height: 1.5; color: var(--colors-ink-muted); }
`
