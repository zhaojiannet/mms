// 收银工作台祝福语：跨组件共享 + 三种触发刷新（进页面 / 结账成功 / 30 分钟自动）
const greetingPool = [
  '今日开张大吉 ✨',
  '财源广进，客似云来 💎',
  '一店十福，生意兴隆 🍀',
  '笑迎八方客，喜纳四海财 🎊',
  '客如流水，财如江海 🌊',
  '顾客盈门，生意如春 🌸',
  '一手好手艺，一天好生意 ✂️',
  '钱袋鼓鼓，笑口常开 💰',
  '一日满堂彩，生意旺翻天 🎯',
  '红红火火，恭喜发财 🧧',
  '天天开门红，月月业绩高 📈',
  '今日辛苦，明日富裕 ☀️',
]

function pickGreeting(prev?: string) {
  const h = new Date().getHours()
  if (h < 5) return '夜深了，记得早点收工 🌙'
  if (h < 8) return '早安老板，今日开门红 🌅'
  if (h >= 22) return '辛苦一天了，好好休息 🌃'
  // 其他时段从 pool 随机；尽量不重复上一句
  let next: string
  let tries = 0
  do {
    next = greetingPool[Math.floor(Math.random() * greetingPool.length)]!
    tries++
  } while (next === prev && tries < 5)
  return next
}

export function useGreeting() {
  const greeting = useState<string>('mms:greeting', () => pickGreeting())
  function refresh() {
    greeting.value = pickGreeting(greeting.value)
  }
  return { greeting, refresh }
}
