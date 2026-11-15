import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'

// 只跑不依赖 Nuxt 运行时的纯函数测试，所以不经过 Nuxt 的构建；
// `~` 别名照 Nuxt 4 指向 app/，被测文件里的显式导入才能解析
export default defineConfig({
  resolve: {
    alias: { '~': fileURLToPath(new URL('./app', import.meta.url)) },
  },
  test: {
    include: ['test/unit/**/*.test.ts'],
    environment: 'node',
  },
})
