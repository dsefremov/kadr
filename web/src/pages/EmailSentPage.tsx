import { Link, useLocation } from 'react-router-dom'
import { Shell } from '../components/Shell'

export function EmailSentPage() {
  const loc = useLocation()
  const email = (loc.state as { email?: string } | null)?.email

  return (
    <Shell>
      <div style={{ maxWidth: 480, margin: '48px auto', textAlign: 'center' }}>
        <h1>Проверьте почту</h1>
        <p className="note" data-testid="email-sent-message">
          Мы отправили письмо{email ? ` на ${email}` : ''}. Перейдите по ссылке, чтобы подтвердить
          адрес.
        </p>
        <Link className="btn" to="/login">
          Ко входу
        </Link>
      </div>
    </Shell>
  )
}
