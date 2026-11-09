import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Shell } from '../components/Shell'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const res = await fetch('/api/auth/password/reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    })
    if (!res.ok) {
      setError('Не удалось отправить письмо')
      return
    }
    setSent(true)
  }

  return (
    <Shell>
      <div style={{ maxWidth: 420, margin: '40px auto' }}>
        <h1>Забыли пароль</h1>
        {sent ? (
          <p data-testid="forgot-sent">Если адрес зарегистрирован, мы отправили письмо со ссылкой.</p>
        ) : (
          <form onSubmit={onSubmit}>
            <label className="field">
              <span>Email</span>
              <input
                className="inp"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                data-testid="forgot-email"
              />
            </label>
            {error ? <p className="hint bad">{error}</p> : null}
            <button type="submit" className="btn pri" data-testid="forgot-submit">
              Отправить ссылку
            </button>
          </form>
        )}
        <p className="note" style={{ marginTop: 24 }}>
          <Link to="/login">Назад к входу</Link>
        </p>
      </div>
    </Shell>
  )
}
