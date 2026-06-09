package pages

import "github.com/gofiber/fiber/v3"

func Mount(app *fiber.App) {
	app.Get("/server/admin", func(c fiber.Ctx) error {
		return c.Type("html").SendString(adminHTML)
	})
}

const adminHTML = `<!doctype html><html><head><meta charset="utf-8"><title>GOpenID Admin</title>
<style>body{font-family:system-ui;margin:32px;background:#f7f8fa;color:#222}.wrap{max-width:880px;margin:auto}section{background:#fff;border:1px solid #dde2e8;border-radius:8px;margin:16px 0;padding:18px}code{background:#eef1f5;padding:2px 6px;border-radius:4px}li{margin:8px 0}</style></head>
<body><main class="wrap"><h1>GOpenID Server Pages</h1><section><h2>Open endpoints</h2><ul><li><code>POST /api/auth/login</code> direct JWT login</li><li><code>GET /.well-known/openid-configuration</code> OIDC discovery</li><li><code>GET|POST /oauth/authorize</code> authorization code flow</li><li><code>POST /oauth/token</code> password and authorization_code grants</li></ul></section><section><h2>Protected admin APIs</h2><p>Use <code>Authorization: Bearer &lt;token&gt;</code>.</p><ul><li><code>/api/admin/departments</code></li><li><code>/api/admin/roles</code></li><li><code>/api/admin/users</code></li></ul></section></main></body></html>`
