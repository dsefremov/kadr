import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { InvalidLink } from '../components/InvalidLink'
import { Shell } from '../components/Shell'

export function VerifyPage() {
  const [params] = useSearchParams()
  const [status, setStatus] = useState<'loading' | 'confirmed' | 'invalid'>('loading')

  useEffect(() => {
    const token = params.get('token')
    if (!token) {
      setStatus('invalid')
      return
    }
    fetch('/api/auth/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    })
      .then((r) => r.json())
      .then((d) => setStatus(d.status === 'confirmed' ? 'confirmed' : 'invalid'))
      .catch(() => setStatus('invalid'))
  }, [params])

  return (
    <Shell>
      <div style={{ maxWidth: 480, margin: '48px auto', textAlign: 'center' }} data-testid="verify-page">
        {status === 'loading' && <p>Подтверждаем почту…</p>}
        {status === 'confirmed' && (
          <>
            <h1>Почта подтверждена</h1>
            <Link className="btn pri" to="/login">
              Войти
            </Link>
          </>
        )}
        {status === 'invalid' && <InvalidLink />}
      </div>
    </Shell>
  )
}
