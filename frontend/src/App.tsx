import LoginPage from './page/login/LoginPage'
import { Navigate, Route, Routes } from 'react-router-dom'
import HomePage from './page/home/HomePage'
import AppLayout from './page/layout/AppLayout'
import TestcasePage from './page/testcase/TestcasePage'
import { TestcaseProvider } from './context/testcase-context'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token')
  if (!token) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <TestcaseProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={(
            <RequireAuth>
              <AppLayout />
            </RequireAuth>
          )}
        >
          <Route index element={<HomePage />} />
          <Route path="testcase" element={<TestcasePage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </TestcaseProvider>
  )
}
