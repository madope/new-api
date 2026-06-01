(function () {
  const translations = {
    zh: {
      title: 'ISURELINK - 智擎 ——企业级智能化业务平台',
      heroTitlePrefix: '智擎 ——',
      heroTitleAccent: '企业级智能化业务平台',
      featureTwo: '稳定高效',
      featureThree: '安全可控',
      loginCta: '登录',
    },
    en: {
      title: 'ISURELINK - Zhiqing Enterprise Intelligent Business Platform',
      heroTitlePrefix: 'Zhiqing - ',
      heroTitleAccent: 'Enterprise Intelligent Business Platform',
      featureTwo: 'Stable and efficient',
      featureThree: 'Secure and controllable',
      loginCta: 'Sign In',
    },
    fr: {
      title: "ISURELINK - Plateforme metier intelligente d'entreprise Zhiqing",
      heroTitlePrefix: 'Zhiqing - ',
      heroTitleAccent: "Plateforme metier intelligente d'entreprise",
      featureTwo: 'Stable et efficace',
      featureThree: 'Securise et maitrisable',
      loginCta: 'Connexion',
    },
    ja: {
      title: 'ISURELINK - 智擎 エンタープライズ向けインテリジェント業務プラットフォーム',
      heroTitlePrefix: '智擎 - ',
      heroTitleAccent: 'エンタープライズ向けインテリジェント業務プラットフォーム',
      featureTwo: '安定して高効率',
      featureThree: '安全で制御しやすい',
      loginCta: 'ログイン',
    },
    ru: {
      title: 'ISURELINK - Zhiqing korporativnaya intellektualnaya biznes-platforma',
      heroTitlePrefix: 'Zhiqing - ',
      heroTitleAccent: 'korporativnaya intellektualnaya biznes-platforma',
      featureTwo: 'Stabilno i effektivno',
      featureThree: 'Bezopasno i pod kontrolem',
      loginCta: 'Vhod',
    },
    vi: {
      title: 'ISURELINK - Zhiqing nen tang van hanh thong minh cap doanh nghiep',
      heroTitlePrefix: 'Zhiqing - ',
      heroTitleAccent: 'nen tang van hanh thong minh cap doanh nghiep',
      featureTwo: 'On dinh va hieu qua',
      featureThree: 'An toan va de kiem soat',
      loginCta: 'Dang nhap',
    },
  }

  const root = document.documentElement
  const searchParams = new URLSearchParams(window.location.search)
  const nav = document.querySelector('[data-purpose="navigation"]')
  const loginButton = document.getElementById('cta-login-button')
  const heroTitle = document.querySelector('[data-i18n="heroTitle"]')
  const exchangeChips = Array.from(document.querySelectorAll('.exchange-chip'))
  const flashTimeouts = []

  function normalizeLang(lang) {
    if (!lang) return 'zh'
    const value = String(lang).toLowerCase()
    if (value.startsWith('en')) return 'en'
    if (value.startsWith('fr')) return 'fr'
    if (value.startsWith('ja')) return 'ja'
    if (value.startsWith('ru')) return 'ru'
    if (value.startsWith('vi')) return 'vi'
    return 'zh'
  }

  function applyLanguage(lang) {
    const localeKey = normalizeLang(lang)
    const locale = translations[localeKey]
    root.lang = localeKey === 'zh' ? 'zh-CN' : localeKey
    document.title = locale.title

    document.querySelectorAll('[data-i18n]').forEach((element) => {
      const key = element.getAttribute('data-i18n')
      if (!key || !locale[key]) return

      if (key === 'heroTitle' && heroTitle) {
        heroTitle.innerHTML =
          locale.heroTitlePrefix +
          '<span class="brand-inline">' +
          locale.heroTitleAccent +
          '</span>'
        return
      }

      element.textContent = locale[key]
    })
  }

  function applyTheme(themeMode) {
    const theme = themeMode === 'dark' ? 'dark' : 'light'
    root.setAttribute('data-theme', theme)
  }

  function normalizeBase(input) {
    if (!input) return ''
    try {
      return new URL(input).origin.replace(/\/+$/, '')
    } catch (error) {
      return ''
    }
  }

  function resolveBase() {
    const explicitBase = searchParams.get('base') || searchParams.get('site') || ''
    const normalized = normalizeBase(explicitBase)
    if (normalized) return normalized

    if (
      window.location.protocol === 'http:' ||
      window.location.protocol === 'https:'
    ) {
      return window.location.origin.replace(/\/+$/, '')
    }

    return ''
  }

  function resolveLoginHref() {
    const base = resolveBase()
    const loginPath = searchParams.get('loginPath') || '/login'
    const normalizedPath = loginPath.startsWith('/') ? loginPath : '/' + loginPath
    const target = base || window.location.origin
    return new URL(normalizedPath, target).toString()
  }

  function applyLoginHref() {
    if (!loginButton) return
    loginButton.href = resolveLoginHref()
  }

  function notifyParentReady() {
    try {
      if (window.parent && window.parent !== window) {
        window.parent.postMessage({ type: 'isurelinkindex-ready' }, '*')
      }
    } catch (error) {
      // Ignore parent messaging failures.
    }
  }

  function flashRandomExchangeChip() {
    if (!exchangeChips.length) return
    const chip =
      exchangeChips[Math.floor(Math.random() * exchangeChips.length)]
    chip.classList.remove('is-flashing')
    void chip.offsetWidth
    chip.classList.add('is-flashing')
    window.setTimeout(function () {
      chip.classList.remove('is-flashing')
    }, 620)
  }

  function scheduleExchangeFlashLoop(beginMs, intervalMs) {
    function queueNext(delayMs) {
      const timeoutId = window.setTimeout(function () {
        flashRandomExchangeChip()
        queueNext(intervalMs)
      }, delayMs)
      flashTimeouts.push(timeoutId)
    }

    queueNext(beginMs)
  }

  function setupExchangeFlashSync() {
    if (!exchangeChips.length) return

    ;[
      { begin: 5280, interval: 11000 },
      { begin: 8040, interval: 13400 },
      { begin: 6660, interval: 12200 },
      { begin: 9500, interval: 14800 },
      { begin: 9644, interval: 12800 },
      { begin: 12884, interval: 15400 },
    ].forEach(function (item) {
      scheduleExchangeFlashLoop(item.begin, item.interval)
    })
  }

  window.addEventListener('message', function (event) {
    const data = event.data || {}
    if (typeof data.themeMode === 'string') {
      applyTheme(data.themeMode)
    }
    if (typeof data.lang === 'string') {
      applyLanguage(data.lang)
    }
  })

  window.addEventListener('scroll', function () {
    if (!nav) return
    if (window.scrollY > 20) {
      nav.classList.add('is-scrolled')
    } else {
      nav.classList.remove('is-scrolled')
    }
  })

  const requestedTheme = searchParams.get('theme')
  if (requestedTheme === 'dark' || requestedTheme === 'light') {
    applyTheme(requestedTheme)
  } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    applyTheme('dark')
  } else {
    applyTheme('light')
  }

  applyLanguage(searchParams.get('lang') || root.lang)
  applyLoginHref()
  setupExchangeFlashSync()
  notifyParentReady()
  window.addEventListener('load', notifyParentReady)
})()
