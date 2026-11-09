import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Shell } from '../components/Shell'

export function LoginPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ email, password }),
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      setError(data.error ?? 'Ошибка входа')
      return
    }
    nav('/')
  }

  return (
    <Shell>
      <div style={{ maxWidth: 420, margin: '40px auto' }}>
        <h1>Вход</h1>
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
              data-testid="login-email"
            />
          </label>
          <label className="field">
            <span>Пароль</span>
            <input
              className="inp"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              data-testid="login-password"
            />
          </label>
          {error ? (
            <p className="hint bad" role="alert">
              {error}
            </p>
          ) : null}
          <button type="submit" className="btn pri" data-testid="login-submit">
            Войти
          </button>
        </form>
        <p className="note" style={{ marginTop: 16 }}>
          Нет аккаунта? <Link to="/register">Регистрация</Link>
        </p>
      </div>
    </Shell>
  )
}
