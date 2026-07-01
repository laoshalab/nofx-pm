export type Page =
  | 'agent'
  | 'competition'
  | 'traders'
  | 'trader'
  | 'strategy'
  | 'prediction'
  | 'data'
  | 'faq'
  | 'login'
  | 'register'

export const ROUTES = {
  home: '/',
  agent: '/agent',
  login: '/login',
  register: '/register',
  setup: '/setup',
  welcome: '/welcome',
  faq: '/faq',
  resetPassword: '/reset-password',
  settings: '/settings',
  data: '/data',
  competition: '/competition',
  traders: '/traders',
  dashboard: '/dashboard',
  strategy: '/strategy',
  prediction: '/prediction',
} as const

export const PAGE_PATHS: Record<Page, string> = {
  agent: ROUTES.agent,
  competition: ROUTES.competition,
  traders: ROUTES.traders,
  trader: ROUTES.dashboard,
  strategy: ROUTES.strategy,
  prediction: ROUTES.prediction,
  data: ROUTES.data,
  faq: ROUTES.faq,
  login: ROUTES.login,
  register: ROUTES.register,
}

export const LEGACY_HASH_ROUTES: Record<string, string> = {
  agent: ROUTES.agent,
  competition: ROUTES.competition,
  traders: ROUTES.traders,
  trader: ROUTES.dashboard,
  details: ROUTES.dashboard,
  strategy: ROUTES.strategy,
  'strategy-market': ROUTES.agent,
  prediction: ROUTES.prediction,
  data: ROUTES.data,
}

export function getCurrentPageForPath(pathname: string): Page | undefined {
  switch (pathname) {
    case ROUTES.agent:
      return 'agent'
    case ROUTES.welcome:
    case ROUTES.traders:
      return 'traders'
    case ROUTES.dashboard:
      return 'trader'
    case ROUTES.strategy:
      return 'strategy'
    case ROUTES.prediction:
      return 'prediction'
    case ROUTES.data:
      return 'data'
    case ROUTES.faq:
      return 'faq'
    case ROUTES.login:
      return 'login'
    case ROUTES.register:
      return 'register'
    case ROUTES.competition:
      return 'competition'
    default:
      return undefined
  }
}

export function buildDashboardPath(traderSlug?: string): string {
  if (!traderSlug) {
    return ROUTES.dashboard
  }

  return `${ROUTES.dashboard}?trader=${encodeURIComponent(traderSlug)}`
}

export function buildPredictionPath(traderId?: string): string {
  if (!traderId) {
    return ROUTES.prediction
  }

  return `${ROUTES.prediction}?trader=${encodeURIComponent(traderId)}`
}
