import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { KeyRound } from 'lucide-react'
import { login } from '../lib/api'
import { Alert, Field } from '../components/ui'

export function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('admin@gopenid.local')
  const [password, setPassword] = useState('admin12345')
  const mutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: () => navigate({ to: '/' }),
  })

  return (
    <main className="login">
      <form onSubmit={(event) => { event.preventDefault(); mutation.mutate() }}>
        <div className="login-top">
          <div className="logo-tag"><KeyRound size={16} />gOpenID</div>
          <div>
            <h1>Yönetim konsoluna giriş</h1>
            <p>Kullanıcı, grup, rol, client, politika ve OIDC yetkilendirme yönetimi.</p>
          </div>
        </div>

        {mutation.error && <Alert type="error" message={mutation.error.message} />}

        <Field label="E-posta">
          <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="admin@gopenid.local" required />
        </Field>
        <Field label="Parola">
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="••••••••" required />
        </Field>

        <button className="btn-primary" disabled={mutation.isPending}>
          {mutation.isPending ? 'Giriş yapılıyor...' : 'Giriş yap'}
        </button>
      </form>
    </main>
  )
}
