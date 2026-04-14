export default defineAppConfig({
  ui: {
    colors: {
      // 默认 teal，运行时可通过 composables/useTheme.ts 切换
      primary: 'teal',
      // 暖米中性色：比 slate 冷蓝灰更温润高端，对齐 Notion / Linear 的 UI 色温
      neutral: 'stone',
    },
    button: {
      defaultVariants: {
        size: 'md',
      },
    },
  },
})
