import { defineStore } from 'pinia'

export type Role = 'super_admin' | 'admin' | 'staff'

export interface User {
  id: string
  email: string
  name: string
  role: Role
}

export const roleLabel = (r: Role | string): string =>
  ({ super_admin: '超级管理员', admin: '管理员', staff: '员工' } as Record<string, string>)[r] ?? r

export const roleHint = (r: Role | string): string =>
  ({
    super_admin: '全部权限，包括账号管理、交易撤销、超级密码等敏感操作',
    admin: '除账号管理、交易撤销外的全部业务操作，可看报表和操作日志',
    staff: '仅日常录入（开单、录入会员、预约），不可查看报表与统计',
  } as Record<string, string>)[r] ?? ''

export const ROLE_OPTIONS: Array<{ label: string; value: Role }> = [
  { label: '超级管理员', value: 'super_admin' },
  { label: '管理员',     value: 'admin' },
  { label: '员工',       value: 'staff' },
]

export const isAtLeastAdmin = (r?: Role | string): boolean =>
  r === 'super_admin' || r === 'admin'

export const isSuperAdmin = (r?: Role | string): boolean => r === 'super_admin'

interface LoginResponse {
  access_token: string
  expires_at: string
  user: User
}

const STORAGE_KEY = 'mms.auth'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: '' as string,
    user: null as User | null,
    expiresAt: '' as string,
  }),
  getters: {
    isAuthenticated: (s): boolean => {
      if (!s.token || !s.user) return false
      // 过期 token 视为未登录：防止"持过期 token 进页面白屏"的 UX bug
      if (s.expiresAt) {
        const exp = new Date(s.expiresAt).getTime()
        if (!isNaN(exp) && exp <= Date.now()) return false
      }
      return true
    },
  },
  actions: {
    hydrate() {
      if (typeof window === 'undefined') return
      const raw = window.localStorage.getItem(STORAGE_KEY)
      if (!raw) return
      try {
        const parsed = JSON.parse(raw)
        this.token = parsed.token ?? ''
        this.user = parsed.user ?? null
        this.expiresAt = parsed.expiresAt ?? ''
      } catch {
        // 解析失败则清空
        this.logout()
      }
    },
    async login(email: string, password: string, remember = false, captcha?: { id: string; answer: string }) {
      const cfg = useRuntimeConfig().public
      const body: Record<string, any> = { email, password, remember }
      if (captcha && captcha.id && captcha.answer) {
        body.captcha_id = captcha.id
        body.captcha_answer = captcha.answer
      }
      const data = await $fetch<LoginResponse>(`${cfg.apiBase}/api/login`, {
        method: 'POST',
        body,
        headers: { 'X-Tenant-Slug': cfg.tenantSlug },
      })
      this.token = data.access_token
      this.user = data.user
      this.expiresAt = data.expires_at
      this.persist()
    },
    logout() {
      this.token = ''
      this.user = null
      this.expiresAt = ''
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem(STORAGE_KEY)
      }
    },
    persist() {
      if (typeof window === 'undefined') return
      window.localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ token: this.token, user: this.user, expiresAt: this.expiresAt }),
      )
    },
  },
})
