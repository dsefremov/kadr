import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Shell } from '../components/Shell'

export function RegisterPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const res = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      setError(data.error ?? 'Ошибка регистрации')
      return
    }
    nav('/email-sent', { state: { email } })
  }

  return (
    <Shell>
      <AuthPanel>
        <h1>Регистрация</h1>
        <p className="note">Откроем доступ после подтверждения почты.</p>
        <form onSubmit={onSubmit}>
          <label className="field">
            <span>Email</span>
            <input
              className="inp"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              data-testid="register-email"
            />
          </label>
          <label className="field">
            <span>Пароль (мин. 10 символов)</span>
            <input
              className="inp"
              type="password"
              autoComplete="new-password"
              required
              minLength={10}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              data-testid="register-password"
            />
          </label>
          {error ? (
            <p className="hint bad" role="alert">
              {error}
            </p>
          ) : null}
          <button type="submit" className="btn pri" data-testid="register-submit">
            Зарегистрироваться
          </button>
        </form>
      </AuthPanel>
    </Shell>
  )
}

function AuthPanel({ children }: { children: React.ReactNode }) {
  return <div style={{ maxWidth: 420, margin: '40px auto' }}>{children}</div>
}
