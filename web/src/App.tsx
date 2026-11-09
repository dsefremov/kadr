import { useEffect, useState } from 'react'
import './styles/tokens.css'
import './styles/app.css'

type HealthPayload = {
  status: string
  build?: { version?: string; commit?: string }
}

export default function App() {
  const [apiVersion, setApiVersion] = useState<string>('…')

  useEffect(() => {
    fetch('/api/health')
      .then((r) => r.json())
      .then((data: HealthPayload) => {
        const v = data.build?.version ?? 'unknown'
        setApiVersion(v)
      })
      .catch(() => setApiVersion('offline'))
  }, [])

  return (
    <div data-testid="kadr-app">
      <header className="top">
        <a className="logo" href="/" aria-label="Кадр">
          <i aria-hidden="true" />
          Кадр
        </a>
        <nav aria-label="Основное">
          <a className="on" href="/">
            Моя лента
          </a>
          <a href="/">Загрузки</a>
          <a href="/">Корзина</a>
          <a href="/">Аккаунт</a>
        </nav>
        <span className="sp" />
        <span className="pill priv" title="Этап 0 — заглушка">
          Этап 0
        </span>
      </header>
      <main className="wrap">
        <div className="banner" role="status">
          <BrandMark />
          <div>
            <div className="t1">Инфраструктура готова</div>
            <p className="note" style={{ margin: 0 }}>
              API: {apiVersion}. Продуктовые экраны появятся на следующих этапах.
            </p>
          </div>
        </div>
        <div className="feedhead">
          <div>
            <h1>Кадр</h1>
            <p>Личная лента фото и видео — заглушка этапа 0</p>
          </div>
          <span className="sp" />
          <span className="pill priv">Личная лента (макет)</span>
        </div>
        <div className="drop">
          <b>Здесь будет ваша лента</b>
          <p className="note">
            Перетащите фото и видео или выберите файлы. На этапе 0 работает только оболочка и
            проверка инфраструктуры.
          </p>
          <button type="button" className="btn pri" disabled title="Этап 1">
            Выбрать файлы
          </button>
        </div>
        <p className="placeholder-badge">Stage 0 placeholder · memories.gallery style shell</p>
      </main>
    </div>
  )
}

function BrandMark() {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true">
      <rect x="4" y="8" width="14" height="14" fill="var(--accent)" rx="2" />
      <rect x="22" y="18" width="14" height="14" fill="var(--ink)" rx="2" />
    </svg>
  )
}
