import { defineStore } from 'pinia'

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'manager' | 'staff'
}

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
    isAuthenticated: (s): boolean => !!s.token && !!s.user,
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
    async login(email: string, password: string) {
      const cfg = useRuntimeConfig().public
      const data = await $fetch<LoginResponse>(`${cfg.apiBase}/api/login`, {
        method: 'POST',
        body: { email, password },
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
