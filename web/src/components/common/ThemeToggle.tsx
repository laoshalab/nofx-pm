import { Moon, Sun } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'
import { useTheme } from '../../contexts/ThemeContext'
import { t } from '../../i18n/translations'

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  const { language } = useLanguage()
  const isDark = theme === 'dark'

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="p-2 rounded-lg transition-colors text-nofx-text-muted hover:text-nofx-gold hover:bg-white/5"
      title={isDark ? t('themeSwitchLight', language) : t('themeSwitchDark', language)}
      aria-label={isDark ? t('themeSwitchLight', language) : t('themeSwitchDark', language)}
    >
      {isDark ? <Sun className="w-[18px] h-[18px]" /> : <Moon className="w-[18px] h-[18px]" />}
    </button>
  )
}
