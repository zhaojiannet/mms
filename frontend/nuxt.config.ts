// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-04-01',

  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@vueuse/nuxt',
  ],

  css: ['~/assets/css/main.css'],

  devServer: {
    host: '0.0.0.0',
    port: 3000,
  },

  ssr: false,

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8081',
      appDomain: process.env.NUXT_PUBLIC_APP_DOMAIN || 'vip.example.com',
      deploymentMode: process.env.NUXT_PUBLIC_DEPLOYMENT_MODE || 'self-hosted',
      tenantSlug: process.env.NUXT_PUBLIC_TENANT_SLUG || 'demo',
    },
  },

  app: {
    head: {
      title: '通用会员管理系统 · SaaS 版',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },
})
