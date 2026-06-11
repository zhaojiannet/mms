<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">最近 {{ items.length }} 条操作记录</p>
      <UButton variant="soft" size="sm" icon="i-lucide-refresh-cw" class="active:scale-95 transition-transform" @click="fetchList">刷新</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-x-auto">
      <table class="w-full min-w-[760px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-44">时间</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">操作人</th>
            <th class="text-left px-4 py-2.5 font-medium">动作</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-40">对象</th>
            <th class="text-center px-4 py-2.5 font-medium whitespace-nowrap w-20">结果</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">IP</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="d in displayItems" :key="d.raw.id">
            <tr
              class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors cursor-pointer"
              @click="toggleExpand(d.raw.id)"
            >
              <td class="px-4 py-2 text-stone-500 tabular-nums whitespace-nowrap">{{ d.formattedTime }}</td>
              <td class="px-4 py-2">
                <div class="flex items-center gap-1.5">
                  <UIcon :name="d.actorIcon" class="size-3.5 text-stone-400 shrink-0" />
                  <span class="font-medium truncate">{{ d.actorDisplay }}</span>
                </div>
              </td>
              <td class="px-4 py-2">
                <span class="inline-flex items-center gap-1.5">
                  <UBadge :label="d.methodLabel" :color="d.methodColor" variant="soft" size="md" />
                  <span class="truncate">{{ d.actionLabel }}</span>
                </span>
              </td>
              <td class="px-4 py-2 text-stone-600 dark:text-stone-400">
                <span v-if="d.targetInfo" class="inline-flex items-center gap-1 text-xs">
                  <span class="text-stone-500">{{ d.targetInfo.typeLabel }}</span>
                  <span v-if="d.targetInfo.shortId" class="tabular-nums text-stone-400">{{ d.targetInfo.shortId }}</span>
                </span>
                <span v-else class="text-stone-400">—</span>
              </td>
              <td class="px-4 py-2 text-center">
                <UBadge
                  :label="d.outcome === 'ok' ? '成功' : '失败'"
                  :color="d.outcome === 'ok' ? 'success' : 'error'"
                  variant="soft" size="md"
                />
              </td>
              <td class="px-4 py-2 text-stone-500 tabular-nums text-xs">{{ d.raw.ip_address || '—' }}</td>
            </tr>
            <tr v-if="expanded.has(d.raw.id)" class="bg-stone-50/60 dark:bg-stone-950/30 border-t border-stone-100 dark:border-stone-800/60">
              <td colspan="6" class="px-4 py-3">
                <pre class="text-xs text-stone-600 dark:text-stone-300 whitespace-pre-wrap break-all leading-relaxed">{{ d.prettyPayload }}</pre>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
    <EmptyState v-else-if="!loading" icon="i-lucide-history" text="暂无操作日志" />
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-12 rounded-2xl" /></div>
  </div>
</template>

<script setup lang="ts">
interface Log {
  id: number
  actor_id: string | null
  actor_name: string | null
  actor_type: string
  action: string
  target_type: string | null
  target_id: string | null
  payload: any
  ip_address: string | null
  user_agent: string | null
  created_at: string
}

const api = useApi()
const items = ref<Log[]>([])
const loading = ref(true)
const expanded = reactive(new Set<number>())

// id→name 映射（用于把 target UUID 解析为可读名字）
// 仅对小量级资源做：users / staff / services / card-types / payment-methods
// members / transactions 数量大不拉（继续显示 UUID 短）
const nameMaps = reactive<Record<string, Record<string, string>>>({
  users: {},
  staff: {},
  services: {},
  'card-types': {},
  'payment-methods': {},
  // 以下 3 个量大，不全拉；按 audit 里出现的 UUID 懒加载
  members: {},
  transactions: {},
  appointments: {},
})

function decodePayload(r: Log): any {
  if (r.payload == null) return null
  if (typeof r.payload === 'string') {
    try {
      // atob 返回 latin-1 字符串，需要经 TextDecoder('utf-8') 还原中文
      const bin = atob(r.payload)
      const bytes = new Uint8Array(bin.length)
      for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
      return JSON.parse(new TextDecoder('utf-8').decode(bytes))
    } catch { return { raw: r.payload } }
  }
  return r.payload
}
function outcome(r: Log) { return decodePayload(r)?.outcome ?? 'unknown' }

// r.action 存储为 "METHOD /api/xxx/:uuid/yyy" 格式；拆出方法和规范化路径后查映射表
function parseAction(r: Log): { method: string; path: string } {
  const s = r.action || ''
  const space = s.indexOf(' ')
  if (space < 0) return { method: '', path: s }
  const method = s.slice(0, space)
  let path = s.slice(space + 1)
  // 去掉 query string
  const q = path.indexOf('?')
  if (q >= 0) path = path.slice(0, q)
  // UUID → :id（兼容 v4/v7 各种形态）
  path = path.replace(/\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi, '/:id')
  // tenant-settings 的 key（非特殊动作路径）→ :key 以便查映射表
  // 排除 enable-void / disable-void / booking-code / super-password 这些单独的动作端点
  if ((method === 'PUT' || method === 'DELETE') &&
      /^\/api\/tenant-settings\/[^/]+$/.test(path) &&
      !/\/(enable-void|disable-void|booking-code|super-password)(\/|$)/.test(path)) {
    path = '/api/tenant-settings/:key'
  }
  return { method, path }
}
function methodLabel(r: Log) { return parseAction(r).method || 'N/A' }
type BadgeColor = 'primary' | 'warning' | 'error' | 'neutral'
function methodColor(r: Log): BadgeColor {
  const m = methodLabel(r)
  if (m === 'POST') return 'primary'
  if (m === 'PUT' || m === 'PATCH') return 'warning'
  if (m === 'DELETE') return 'error'
  return 'neutral'
}

const actionMap: Record<string, string> = {
  // 读审计（仅员工角色触发，用于回溯批量查询行为）
  'GET /api/members':                               '查看会员列表',
  'GET /api/members/:id':                           '查看会员详情',
  'GET /api/transactions':                          '查看交易列表',
  'GET /api/transactions/:id':                      '查看交易详情',
  // 会员
  'POST /api/members':                              '新建会员',
  'PUT /api/members/:id':                           '编辑会员',
  'DELETE /api/members/:id':                        '删除会员',
  // 会员卡（前 3 个端点已从后端移除，标签保留仅为回看历史日志）
  'POST /api/members/:id/cards':                    '办卡（已废弃端点）',
  'PUT /api/cards/:id':                             '编辑会员卡（已废弃端点）',
  'DELETE /api/cards/:id':                          '删除会员卡（已废弃端点）',
  'POST /api/members/:id/cards/with-transaction':   '办卡并收款',
  // 挂账
  'POST /api/members/:id/pending':                  '新增挂账',
  'DELETE /api/members/:id/pending/:id':            '撤消挂账',
  'POST /api/members/:id/pending/:id/settle':       '清账',
  'POST /api/members/:id/pending/settle-all':       '批量清账',
  // 交易
  'POST /api/transactions':                         '记一笔收款',
  'POST /api/transactions/:id/void':                '撤销交易',
  // 预约
  'POST /api/appointments':                         '新建预约',
  'PATCH /api/appointments/:id/status':             '修改预约状态',
  'DELETE /api/appointments/:id':                   '删除预约',
  'POST /api/booking':                              '顾客线上预约',
  // 基础数据
  'POST /api/services':                             '新建服务项目',
  'PUT /api/services/:id':                          '编辑服务项目',
  'DELETE /api/services/:id':                       '删除服务项目',
  'POST /api/card-types':                           '新建会员卡类型',
  'PUT /api/card-types/:id':                        '编辑会员卡类型',
  'DELETE /api/card-types/:id':                     '删除会员卡类型',
  'POST /api/staff':                                '新增员工',
  'PUT /api/staff/:id':                             '编辑员工',
  'DELETE /api/staff/:id':                          '删除员工',
  'POST /api/payment-methods':                      '新增支付方式',
  'PUT /api/payment-methods/:id':                   '编辑支付方式',
  'DELETE /api/payment-methods/:id':                '删除支付方式',
  // 账号与安全
  'POST /api/users':                                '新建账号',
  'PUT /api/users/:id':                             '编辑账号',
  'DELETE /api/users/:id':                          '删除账号',
  'POST /api/users/:id/reset-password':             '重置登录密码（他人）',
  'POST /api/auth/change-password':                 '修改登录密码（本人）',
  'POST /api/tenant-settings/super-password':       '设置超级密码',
  // 交易撤销 / 预约码 / 系统设置
  'POST /api/tenant-settings/enable-void':          '开启交易撤销',
  'POST /api/tenant-settings/disable-void':         '关闭交易撤销',
  'POST /api/tenant-settings/booking-code':         '重新生成预约码',
  'PUT /api/tenant-settings/:key':                  '修改系统设置',
  'DELETE /api/tenant-settings/:key':               '清除系统设置',
  // 店铺 Logo
  'POST /api/store/logo':                           '上传店铺 Logo',
  'DELETE /api/store/logo':                         '删除店铺 Logo',
  // 提成规则（已下线；历史日志仍可能出现此 action，保留映射仅为回看）
  'POST /api/commission-rules':                     '新建提成规则',
  'PUT /api/commission-rules/:id':                  '编辑提成规则',
  'DELETE /api/commission-rules/:id':               '删除提成规则',
}

function actionLabel(r: Log): string {
  const { method, path } = parseAction(r)
  const key = `${method} ${path}`
  const label = actionMap[key]
  if (label) return label
  // fallback: 展示路径尾段
  return path.split('/').filter(Boolean).slice(-2).join('/') || r.action || '—'
}
const resourceMap: Record<string, string> = {
  members: '会员',
  transactions: '交易',
  appointments: '预约',
  booking: '预约',
  services: '服务项目',
  'card-types': '会员卡类型',
  cards: '会员卡',
  staff: '员工',
  users: '账号',
  'payment-methods': '支付方式',
  'commission-rules': '提成规则',
  pending: '挂账',
  audit: '日志',
  store: '店铺',
}

// 复合路径精确匹配（typeKey/subKey[/subSubKey]）
const compoundTargetMap: Record<string, string> = {
  'tenant-settings/enable-void':           '交易撤销',
  'tenant-settings/disable-void':          '交易撤销',
  'tenant-settings/super-password':        '超级密码',
  'tenant-settings/super-password/status': '超级密码',
  'tenant-settings/booking-code':          '预约码',
  'auth/change-password':                  '登录密码',
  'store/logo':                            'Logo',
}

// tenant-settings 的常见 key 中文化（用于对象列显示）
const settingKeyLabel: Record<string, string> = {
  store_name:              '店铺名称',
  store_logo_url:          '店铺 Logo',
  login_bg_theme:          '登录背景',
  enable_transaction_void: '撤单开关',
  void_enabled_at:         '撤单开启时刻',
  booking_code:            '预约码',
  booking_code_updated_at: '预约码更新时刻',
}

function targetInfo(r: Log): { typeLabel: string; shortId: string } | null {
  const rawPath = (r.action || '').split(' ')[1]?.split('?')[0] ?? ''
  if (!rawPath) return null
  const rawSegs = rawPath.split('/').filter(Boolean)
  const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

  // api 之后的片段
  const idx = rawSegs.indexOf('api')
  const after = idx >= 0 ? rawSegs.slice(idx + 1) : rawSegs
  if (after.length === 0) return null

  const typeKey = after[0] ?? ''
  const sub = after[1]
  const sub2 = after[2]

  // 通用复合匹配（含 tenant-settings / auth / store 等）
  if (sub) {
    const k3 = sub2 ? `${typeKey}/${sub}/${sub2}` : ''
    const k2 = `${typeKey}/${sub}`
    const label = (k3 && compoundTargetMap[k3]) || compoundTargetMap[k2]
    if (label) return { typeLabel: label, shortId: '' }
  }

  // tenant-settings / auth 特殊 fallback
  if (typeKey === 'tenant-settings') {
    if (sub && !uuidRe.test(sub)) {
      return { typeLabel: settingKeyLabel[sub] ?? `配置·${sub}`, shortId: '' }
    }
    return { typeLabel: '系统设置', shortId: '' }
  }
  if (typeKey === 'auth') {
    return { typeLabel: '认证', shortId: '' }
  }

  // 嵌套资源：/members/:id/pending[/:pid[/settle]] / /members/:id/cards[/with-transaction]
  if (typeKey === 'members' && after.length >= 3) {
    const nested = after[2]
    const memberUuid = after[1] && uuidRe.test(after[1]) ? after[1] : ''
    const memberName = memberUuid ? nameMaps.members?.[memberUuid] : ''
    if (nested === 'pending') {
      // 优先显示会员名字（"挂账·张三"），fallback 显示 pending_id 短
      const pid = after[3] && uuidRe.test(after[3]) ? after[3] : ''
      const label = memberName || (pid ? pid.slice(0, 8) : (memberUuid ? memberUuid.slice(0, 8) : ''))
      return { typeLabel: '挂账', shortId: label }
    }
    if (nested === 'cards') {
      return { typeLabel: '会员卡', shortId: memberName || '' }
    }
  }

  // 默认：取最后一个 UUID 作为 shortId
  let lastUuid = ''
  for (const s of after) if (uuidRe.test(s)) lastUuid = s
  // 优先用名字映射（已删除账号/员工会 fallback 到 UUID 短片段）
  const resolved = lastUuid && nameMaps[typeKey]?.[lastUuid]
  const shortId = resolved || (lastUuid ? lastUuid.slice(0, 8) : '')
  const typeLabel = resourceMap[typeKey] ?? typeKey
  return { typeLabel, shortId }
}
function actorIcon(t: string) {
  return t === 'user' ? 'i-lucide-user' : 'i-lucide-user-round'
}
function actorDisplay(r: Log): string {
  if (r.actor_name) return r.actor_name
  if (r.actor_type === 'user' && r.actor_id) return r.actor_id.slice(0, 8)
  return '顾客'
}
function prettyPayload(r: Log) {
  const p = decodePayload(r) || {}
  return JSON.stringify(p, null, 2)
}

function toggleExpand(id: number) {
  if (expanded.has(id)) expanded.delete(id)
  else expanded.add(id)
}

// 派生显示字段：派生量较重（payload base64 解码 / atob / JSON.parse / parseAction 多次）
// 整体 computed 缓存 → items 或 nameMaps 变更时整列重算一次，模板内只读字段
const displayItems = computed(() => items.value.map((r) => {
  const decoded = decodePayload(r)
  const parsed = parseAction(r)
  const method = parsed.method || 'N/A'
  return {
    raw: r,
    formattedTime: formatTime(r.created_at),
    actorIcon:     actorIcon(r.actor_type),
    actorDisplay:  actorDisplay(r),
    methodLabel:   method,
    methodColor:   methodColor(r),
    actionLabel:   actionLabel(r),
    targetInfo:    targetInfo(r),
    outcome:       decoded?.outcome ?? 'unknown',
    prettyPayload: JSON.stringify(decoded || {}, null, 2),
  }
}))

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: Log[] }>('/api/audit-logs?limit=100')
    items.value = data.items
  } finally { loading.value = false }
}

// 并行加载 id→name 字典，让对象列显示"账号·张三"而不是"账号·019d9f5b"
// 对 admin+ 这些接口都有权限；失败时静默 fallback 到 UUID 显示
async function loadNameMaps() {
  const endpoints: Array<{ key: string; url: string; field?: 'name' | 'email' }> = [
    { key: 'users',           url: '/api/users?limit=200' },
    { key: 'staff',           url: '/api/staff' },
    { key: 'services',        url: '/api/services' },
    { key: 'card-types',      url: '/api/card-types' },
    { key: 'payment-methods', url: '/api/payment-methods' },
  ]
  await Promise.all(endpoints.map(async (e) => {
    try {
      const data = await api<{ items: Array<{ id: string; name?: string; email?: string }> }>(e.url)
      for (const it of data.items || []) {
        nameMaps[e.key]![it.id] = it.name || it.email || it.id.slice(0, 8)
      }
    } catch {}
  }))
}

// 审计加载后，按需批量解析 members/transactions/appointments 的 UUID → 名字
async function resolveLazyNames() {
  const uuidRe = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi
  const memberIds = new Set<string>()
  const txIds = new Set<string>()
  const apptIds = new Set<string>()

  for (const r of items.value) {
    const path = (r.action || '').split(' ')[1]?.split('?')[0] ?? ''
    const uuids = path.match(uuidRe) || []
    const segs = path.split('/').filter(Boolean)
    const api = segs.indexOf('api')
    const after = api >= 0 ? segs.slice(api + 1) : segs

    if (after[0] === 'members' && uuids[0]) memberIds.add(uuids[0])
    if (after[0] === 'transactions' && uuids[0]) txIds.add(uuids[0])
    if (after[0] === 'appointments' && uuids[0]) apptIds.add(uuids[0])
  }

  const qs: string[] = []
  if (memberIds.size) qs.push('members=' + [...memberIds].join(','))
  if (txIds.size) qs.push('transactions=' + [...txIds].join(','))
  if (apptIds.size) qs.push('appointments=' + [...apptIds].join(','))
  if (qs.length === 0) return

  try {
    const data = await api<{
      members: Record<string, string>
      transactions: Record<string, string>
      appointments: Record<string, string>
    }>('/api/audit-logs/lookup?' + qs.join('&'))
    Object.assign(nameMaps.members!,      data.members ?? {})
    Object.assign(nameMaps.transactions!, data.transactions ?? {})
    Object.assign(nameMaps.appointments!, data.appointments ?? {})
  } catch {}
}

onMounted(async () => {
  loadNameMaps() // 并行拉小资源字典
  await fetchList()
  await resolveLazyNames() // audit 加载完后按需解析大资源
})
</script>
