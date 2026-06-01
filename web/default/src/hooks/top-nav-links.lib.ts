type Translate = (key: string) => string

export type TopNavLink = {
  title: string
  href: string
  disabled?: boolean
  external?: boolean
}

type HeaderAccessModule = {
  enabled: boolean
  requireAuth: boolean
}

type HeaderNavModules = {
  home: boolean
  console: boolean
  pricing: HeaderAccessModule
  rankings: HeaderAccessModule
  docs: boolean
  about: boolean
}

type StatusLike = {
  HeaderNavModules?: unknown
  docs_link?: string | undefined
}

export const DEFAULT_HEADER_NAV_MODULES: HeaderNavModules = {
  home: true,
  console: true,
  pricing: { enabled: true, requireAuth: false },
  rankings: { enabled: true, requireAuth: false },
  docs: true,
  about: true,
}

function parseAccessModule(
  raw: unknown,
  fallback: HeaderAccessModule
): HeaderAccessModule {
  if (
    typeof raw === 'boolean' ||
    typeof raw === 'string' ||
    typeof raw === 'number'
  ) {
    return {
      enabled: raw === true || raw === 'true' || raw === '1' || raw === 1,
      requireAuth: fallback.requireAuth,
    }
  }

  if (raw && typeof raw === 'object') {
    const record = raw as Record<string, unknown>
    return {
      enabled:
        typeof record.enabled === 'boolean' ? record.enabled : fallback.enabled,
      requireAuth:
        typeof record.requireAuth === 'boolean'
          ? record.requireAuth
          : fallback.requireAuth,
    }
  }

  return { ...fallback }
}

export function parseHeaderNavModules(raw: unknown): HeaderNavModules {
  if (!raw || String(raw).trim() === '') {
    return DEFAULT_HEADER_NAV_MODULES
  }

  try {
    const parsed = JSON.parse(String(raw)) as Record<string, unknown>
    return {
      ...DEFAULT_HEADER_NAV_MODULES,
      ...parsed,
      pricing: parseAccessModule(
        parsed.pricing,
        DEFAULT_HEADER_NAV_MODULES.pricing
      ),
      rankings: parseAccessModule(
        parsed.rankings,
        DEFAULT_HEADER_NAV_MODULES.rankings
      ),
    }
  } catch {
    return DEFAULT_HEADER_NAV_MODULES
  }
}

export function buildTopNavLinks({
  t,
  status,
  isAuthed,
}: {
  t: Translate
  status?: StatusLike
  isAuthed: boolean
}): TopNavLink[] {
  const modules = parseHeaderNavModules(status?.HeaderNavModules)
  const docsLink = status?.docs_link
  const links: TopNavLink[] = []

  if (modules.home !== false) {
    links.push({ title: t('Home'), href: '/' })
  }

  if (modules.console !== false) {
    links.push({ title: t('Console'), href: '/dashboard' })
  }

  if (
    modules.pricing.enabled &&
    (!modules.pricing.requireAuth || isAuthed)
  ) {
    links.push({ title: t('Model Square'), href: '/pricing' })
  }

  if (modules.rankings.enabled) {
    const disabled = modules.rankings.requireAuth && !isAuthed
    links.push({ title: t('Rankings'), href: '/rankings', disabled })
  }

  if (modules.docs !== false) {
    if (docsLink) {
      links.push({ title: t('Docs'), href: docsLink, external: true })
    } else {
      links.push({ title: t('Docs'), href: '/docs' })
    }
  }

  if (modules.about !== false) {
    links.push({ title: t('About'), href: '/about' })
  }

  return links
}
