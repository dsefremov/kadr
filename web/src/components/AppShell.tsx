import { Link, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { QuotaBar } from './QuotaBar'

type Quota = {
  limit: number
  used: number
  trash: number
  reserved: number
  status?: string
}

type Props = {
  children: ReactNode
  quota?: Quota | null
  onUpload?: () => void
}

export function AppShell({ children, quota, onUpload }: Props) {
  const loc = useLocation()
  const nav = (path: string) => (loc.pathname === path ? 'on' : '')
  return (
    <div className="hasnav" data-testid="kadr-app">
      <header className="top">
        <Link className="logo" to="/" aria-label="Кадр">
          <i aria-hidden="true" />
          Кадр
        </Link>
        <nav aria-label="Основное">
          <Link className={nav('/')} to="/">
            Моя лента
          </Link>
          <Link className={nav('/uploads')} to="/uploads" data-testid="nav-uploads">
            Загрузки
          </Link>
          <Link className={nav('/trash')} to="/trash" data-testid="open-trash">
            Корзина
          </Link>
          <Link className={nav('/account')} to="/account">
            Аккаунт
          </Link>
        </nav>
        <span className="sp" />
        <div className="top-actions hide-m">
          {quota ? (
            <QuotaBar
              used={quota.used}
              trash={quota.trash}
              reserved={quota.reserved}
              limit={quota.limit}
              status={quota.status}
              compact
            />
          ) : null}
          {onUpload ? (
            <button type="button" className="btn pri" data-testid="nav-upload" onClick={onUpload}>
              ＋ Загрузить
            </button>
          ) : null}
        </div>
      </header>
      <main className="wrap">{children}</main>
      <nav className="mob-nav" aria-label="Мобильная навигация">
        <Link className={nav('/')} to="/">
          <span>▦</span>
          Лента
        </Link>
        <Link className={nav('/trash')} to="/trash" data-testid="open-trash">
          <span>⌫</span>
          Корзина
        </Link>
        {onUpload ? (
          <button type="button" className="up" data-testid="mob-upload" onClick={onUpload}>
            <span>＋</span>
            Загрузить
          </button>
        ) : (
          <Link className={nav('/uploads')} to="/uploads">
            <span>＋</span>
            Загрузить
          </Link>
        )}
        <Link className={nav('/uploads')} to="/uploads">
          <span>◔</span>
          Загрузки
        </Link>
        <Link className={nav('/account')} to="/account">
          <span>☺</span>
          Аккаунт
        </Link>
      </nav>
    </div>
  )
}
