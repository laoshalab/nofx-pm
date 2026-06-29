import { ConfirmDialogProvider } from './components/common/ConfirmDialog'
import { ThemedToaster } from './components/common/ThemedToaster'
import { AuthProvider } from './contexts/AuthContext'
import { LanguageProvider } from './contexts/LanguageContext'
import { ThemeProvider } from './contexts/ThemeContext'
import { AppRoutes } from './router/AppRoutes'

export default function App() {
  return (
    <ThemeProvider>
      <LanguageProvider>
        <AuthProvider>
          <ConfirmDialogProvider>
            <ThemedToaster />
            <AppRoutes />
          </ConfirmDialogProvider>
        </AuthProvider>
      </LanguageProvider>
    </ThemeProvider>
  )
}
