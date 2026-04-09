import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import AppSidebar from '../AppSidebar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({
    path: '/dashboard'
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    sidebarCollapsed: false,
    mobileOpen: false,
    publicSettingsLoaded: true,
    siteName: 'Sub2API',
    siteLogo: '',
    siteVersion: 'test',
    cachedPublicSettings: {
      purchase_subscription_enabled: true,
      xunhupay_enabled: true,
      affiliate_enabled: false,
      custom_menu_items: []
    },
    backendModeEnabled: false,
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn(),
    fetchPublicSettings: vi.fn()
  }),
  useAuthStore: () => ({
    isAdmin: false,
    isSimpleMode: false
  }),
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn()
  }),
  useAdminSettingsStore: () => ({
    opsMonitoringEnabled: false,
    customMenuItems: [],
    fetch: vi.fn()
  })
}))

vi.mock('@/utils/sanitize', () => ({
  sanitizeSvg: (value: string) => value
}))

describe('AppSidebar', () => {
  beforeEach(() => {
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn()
    })))
  })

  it('does not render the tutorial menu link', () => {
    const wrapper = mount(AppSidebar, {
      global: {
        stubs: {
          VersionBadge: true,
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>'
          }
        }
      }
    })

    expect(wrapper.html()).not.toContain('/tutorial')
    expect(wrapper.text()).not.toContain('nav.tutorialGuide')
  })
})
