import { useEffect, useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { InvalidLink } from '../components/InvalidLink'
import { Shell } from '../components/Shell'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const [password, setPassword] = useState('')
  const [status, setStatus] = useState<'form' | 'done' | 'invalid'>('form')

  const token = params.get('token')

  useEffect(() => {
    if (!token) setStatus('invalid')
  }, [token])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token) return
    const res = await fetch('/api/auth/password/reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token, password }),
    })
    const data = await res.json().catch(() => ({}))
    if (data.status === 'reset') {
      setStatus('done')
    } else {
      setStatus('invalid')
    }
  }

  return (
    <Shell>
      <div style={{ maxWidth: 420, margin: '48px auto', textAlign: 'center' }} data-testid="reset-page">
        {status === 'invalid' && <InvalidLink />}
        {status === 'done' && (
          <>
            <h1>Пароль обновлён</h1>
            <Link className="btn pri" to="/login">
              Войти
            </Link>
          </>
        )}
        {status === 'form' && (
          <>
            <h1>Новый пароль</h1>
            <form onSubmit={onSubmit} style={{ textAlign: 'left' }}>
              <label className="field">
                <span>Пароль (мин. 10 символов)</span>
                <input
                  className="inp"
                  type="password"
                  required
                  minLength={10}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  data-testid="reset-password"
                />
              </label>
              <button type="submit" className="btn pri" data-testid="reset-submit">
                Сохранить
              </button>
            </form>
          </>
        )}
      </div>
    </Shell>
  )
}
