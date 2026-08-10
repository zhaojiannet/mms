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
      // 默认同源相对路径（生产各商户域名下由反代按 /api 路径分流）；开发在 .env 指到 localhost:8081
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '',
      appDomain: process.env.NUXT_PUBLIC_APP_DOMAIN || 'vip.example.com',
      deploymentMode: process.env.NUXT_PUBLIC_DEPLOYMENT_MODE || 'self-hosted',
      tenantSlug: process.env.NUXT_PUBLIC_TENANT_SLUG || '',
    },
  },

  app: {
    head: {
      // 不设默认 title：app.vue 的 titleTemplate 统一兜底为「通用会员管理系统」，
      // 这里再设一个会被当作页面名拼进模板，产生「SaaS 版 · 通用会员管理系统」这种重复

      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },

  vite: {
    optimizeDeps: {
      // reka-ui 经 @nuxt/ui 运行时从 node_modules 内引入，开发态按源码直出、
      // 不参与预打包；应用侧 import 的 @internationalized/date 若被预打包，
      // 会产生第二份类实例，reka-ui 的 instanceof 判断失败，DateField 时分段丢失。
      exclude: ['@internationalized/date'],
    },
  },
})
