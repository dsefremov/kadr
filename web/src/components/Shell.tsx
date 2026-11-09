import { Link } from 'react-router-dom'
import type { ReactNode } from 'react'

export function Shell({ children }: { children: ReactNode }) {
  return (
    <div data-testid="kadr-app">
      <header className="top">
        <Link className="logo" to="/" aria-label="Кадр">
          <i aria-hidden="true" />
          Кадр
        </Link>
        <nav aria-label="Основное">
          <Link to="/">Моя лента</Link>
          <Link to="/login">Вход</Link>
          <Link className="btn pri sm" to="/register">
            Регистрация
          </Link>
        </nav>
      </header>
      <main className="wrap">{children}</main>
    </div>
  )
}
