import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import './styles/tokens.css'
import './styles/app.css'
import { EmailSentPage } from './pages/EmailSentPage'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { VerifyPage } from './pages/VerifyPage'
import { ForgotPasswordPage } from './pages/ForgotPasswordPage'
import { ResetPasswordPage } from './pages/ResetPasswordPage'
import { TrashPage } from './pages/TrashPage'
import { AccountPage } from './pages/AccountPage'
import { UploadsPage } from './pages/UploadsPage'
import { ViewerPage } from './pages/ViewerPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/email-sent" element={<EmailSentPage />} />
        <Route path="/verify" element={<VerifyPage />} />
        <Route path="/forgot" element={<ForgotPasswordPage />} />
        <Route path="/reset" element={<ResetPasswordPage />} />
        <Route path="/trash" element={<TrashPage />} />
        <Route path="/uploads" element={<UploadsPage />} />
        <Route path="/account" element={<AccountPage />} />
        <Route path="/view/:mediaId" element={<ViewerPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
