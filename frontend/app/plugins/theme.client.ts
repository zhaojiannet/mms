// 启动时从 localStorage 读取并应用用户上次选择的主题色
export default defineNuxtPlugin(() => {
  const { init } = useTheme()
  init()
})
