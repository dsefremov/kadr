import { Link } from 'react-router-dom'

export function InvalidLink() {
  return (
    <div data-testid="invalid-link">
      <h1>Ссылка недействительна или устарела</h1>
      <p className="note">Запросите новое письмо при входе или регистрации.</p>
      <Link className="btn" to="/login">
        Войти
      </Link>
    </div>
  )
}
