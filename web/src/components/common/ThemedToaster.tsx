import { Toaster } from 'sonner'
import { useTheme } from '../../contexts/ThemeContext'

export function ThemedToaster() {
  const { theme } = useTheme()

  return (
    <Toaster
      theme={theme}
      richColors
      closeButton
      position="top-center"
      duration={2200}
      toastOptions={{
        className: 'nofx-toast',
        style: {
          background: 'var(--panel-bg)',
          border: '1px solid var(--panel-border)',
          color: 'var(--text-primary)',
        },
      }}
    />
  )
}
