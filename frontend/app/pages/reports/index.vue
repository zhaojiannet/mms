<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
    <!-- Header：标题 + 日期 input（复用预约） + preset 分段按钮 -->
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">报表</h1>

      <div class="flex items-center gap-2 flex-wrap">
        <!-- 日期范围 input：复用预约 UInputDate + UCalendar -->
        <UPopover :ui="{ content: 'p-2 w-auto' }">
          <UInputDate
            :model-value="rangeCalValue"
            @update:model-value="onRangeUpdate"
            range
            size="sm"
            icon="i-lucide-calendar"
            class="h-8! py-0! text-sm!"
            :ui="{ leadingIcon: 'size-4 text-stone-400' }"
          />
          <template #content>
            <UCalendar :model-value="rangeCalValue" @update:model-value="onRangeUpdate" class="p-2" range />
          </template>
        </UPopover>

        <!-- Preset 快捷范围：独立按钮 + gap-1，与撤销记录页的分段控件统一 -->
        <div class="flex items-center gap-1">
          <UButton
            v-for="p in presets" :key="p.key"
            size="xs"
            :variant="preset === p.key ? 'soft' : 'ghost'"
            :color="preset === p.key ? 'primary' : 'neutral'"
            class="active:scale-95 transition-transform"
            @click="applyPreset(p)"
          >{{ p.label }}</UButton>
        </div>
      </div>
    </header>

    <!-- Tabs（下划线风格，与 /settings 统一） -->
    <nav class="flex items-center gap-1 flex-wrap border-b border-stone-200/60 dark:border-stone-800">
      <button
        v-for="t in tabs" :key="t.value"
        type="button"
        :class="[
          'inline-flex items-center gap-2 px-3 py-2.5 text-sm whitespace-nowrap border-b-2 -mb-px transition-colors duration-150 cursor-pointer focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:rounded',
          activeTab === t.value
            ? 'text-primary-700 dark:text-primary-300 border-primary-500 font-medium'
            : 'text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100 border-transparent',
        ]"
        @click="activeTab = t.value"
      >
        <UIcon :name="t.icon" class="size-4 shrink-0" />
        {{ t.label }}
      </button>
    </nav>

    <!-- ============ 流水（含概览 KPI） ============ -->
    <div v-show="activeTab === 'transactions'">
        <div class="space-y-5 mt-5">
          <!-- 概览 KPI：6 指标 -->
          <div class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
            <div
              v-for="k in overviewKpis"
              :key="k.label"
              class="group relative px-4 py-3.5 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 hover:ring-primary-200 dark:hover:ring-primary-900 transition-all"
            >
              <div class="flex items-center gap-1.5 text-xs text-stone-500">
                <UIcon :name="k.icon" class="size-3.5" :class="k.iconColor" />
                <span>{{ k.label }}</span>
              </div>
              <div v-if="overviewLoading" class="h-7 mt-1.5 w-20 rounded-md bg-stone-100 dark:bg-stone-800 animate-pulse" />
              <div v-else class="text-2xl font-semibold tabular-nums mt-1 tracking-tight">{{ k.value }}</div>
            </div>
          </div>

          <div
            v-if="biz && parseFloat(biz.credit_charged || '0') > 0"
            class="flex items-center gap-2 text-sm text-warning-700 dark:text-warning-400 bg-warning-50 dark:bg-warning-950/40 ring-1 ring-warning-200/60 dark:ring-warning-900/40 rounded-xl px-3.5 py-2"
          >
            <UIcon name="i-lucide-info" class="size-4 shrink-0" />
            <span>本期新增挂账 <strong class="tabular-nums">{{ fmtMoney(biz.credit_charged) }}</strong>（已计入营业总额）</span>
          </div>

          <!-- 流水 filter + 列表 -->
          <div class="flex items-center gap-2 flex-wrap pt-1">
            <UBadge
              v-for="f in txFilters"
              :key="f.key"
              :label="f.label"
              :color="txKind === f.key ? 'primary' : 'neutral'"
              :variant="txKind === f.key ? 'solid' : 'soft'"
              class="cursor-pointer active:scale-95 transition-transform"
              @click="txKind = f.key as any; fetchTx(true)"
            />
            <span class="ml-auto text-sm text-stone-500 tabular-nums">共 {{ txTotal }} 笔</span>
          </div>

          <div
            v-if="txItems.length > 0"
            class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-x-auto"
          >
            <table class="w-full min-w-[880px] text-sm table-fixed">
              <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                <tr>
                  <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">姓名</th>
                  <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-40">会员卡</th>
                  <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-20">类型</th>
                  <th class="text-left px-4 py-2.5 font-medium">服务项目</th>
                  <th class="text-center px-4 py-2.5 font-medium whitespace-nowrap w-14">数量</th>
                  <th class="text-right px-4 py-2.5 font-medium w-40">金额</th>
                  <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-44">时间</th>
                  <th v-if="canVoid" class="text-center px-4 py-2.5 font-medium whitespace-nowrap w-28">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="t in txItems" :key="t.id"
                  :class="['border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors', t.status === 'voided' ? 'opacity-60' : '']"
                >
                  <!-- 姓名 -->
                  <td class="px-4 py-2 whitespace-nowrap truncate">
                    <span v-if="t.member_name" class="inline-flex items-center gap-1.5 text-stone-900 dark:text-stone-100 font-medium">
                      <UIcon name="i-lucide-user-round" class="size-4 text-primary-500 shrink-0" />
                      <span class="truncate">{{ t.member_name }}</span>
                    </span>
                    <span v-else-if="t.customer_name" class="text-stone-900 dark:text-stone-100 font-medium">
                      {{ t.customer_name }}
                      <span class="text-xs text-stone-400 font-normal ml-1">散客</span>
                    </span>
                    <span v-else class="text-stone-400">—</span>
                  </td>
                  <!-- 会员卡 -->
                  <td class="px-4 py-2 whitespace-nowrap truncate">
                    <span v-if="t.card_type_name" class="inline-flex items-center gap-1.5 text-primary-600 dark:text-primary-400">
                      <UIcon name="i-lucide-credit-card" class="size-4 shrink-0" />
                      <span class="truncate">{{ t.card_type_name }}</span>
                    </span>
                    <span v-else class="text-stone-400">—</span>
                  </td>
                  <!-- 类型 -->
                  <td class="px-4 py-2 whitespace-nowrap">
                    <UBadge :label="kindLabel(t.kind)" :color="kindColor(t.kind)" variant="soft" size="md" />
                  </td>
                  <!-- 服务项目（允许换行） -->
                  <td class="px-4 py-2 text-stone-600 dark:text-stone-400 break-words">{{ t.summary || '—' }}</td>
                  <!-- 数量 -->
                  <td class="px-4 py-2 text-center tabular-nums text-stone-600 dark:text-stone-400 whitespace-nowrap">{{ t.item_qty || '—' }}</td>
                  <!-- 金额（实付大号 + 应收划线 + 省/已撤销 + 余额快照 hover） -->
                  <td class="px-4 py-2 text-right align-middle">
                    <div class="relative inline-block group">
                      <div
                        class="flex items-baseline justify-end gap-1.5 leading-tight"
                        :class="t.card_snapshots && t.card_snapshots.length > 0 ? 'cursor-help border-b border-dashed border-stone-300 dark:border-stone-600 pb-0.5' : ''"
                      >
                        <span v-if="parseFloat(t.discount_amount) > 0" class="text-xs tabular-nums text-stone-400 line-through">¥{{ t.total_amount }}</span>
                        <span class="text-base font-semibold tabular-nums text-primary-600 dark:text-primary-400">¥{{ t.actual_paid_amount }}</span>
                      </div>
                      <div class="h-4 text-xs tabular-nums leading-none mt-0.5">
                        <span v-if="parseFloat(t.discount_amount) > 0" class="text-error-600">省 ¥{{ t.discount_amount }}</span>
                        <span v-else-if="t.status === 'voided'" class="text-error-600">已撤销</span>
                      </div>
                      <!-- 余额快照 tooltip -->
                      <div
                        v-if="t.card_snapshots && t.card_snapshots.length > 0"
                        class="pointer-events-none absolute right-0 top-full mt-1 z-30 min-w-64 px-3 py-2.5 rounded-md bg-stone-900/95 dark:bg-stone-100/95 text-white dark:text-stone-900 shadow-lg opacity-0 group-hover:opacity-100 transition-opacity duration-150"
                      >
                        <div class="text-xs text-stone-400 dark:text-stone-500 mb-1.5">余额快照</div>
                        <div v-for="s in t.card_snapshots" :key="s.card_id" class="flex items-baseline justify-between gap-3 py-0.5 text-sm">
                          <span class="truncate text-left">
                            {{ s.card_type_name }}
                            <span v-if="s.change_type === 'void_restore'" class="text-xs text-warning-500 ml-1">还原</span>
                            <span v-else-if="s.change_type === 'issue'" class="text-xs text-primary-400 ml-1">办卡</span>
                          </span>
                          <span class="tabular-nums shrink-0">
                            <span class="text-stone-400 dark:text-stone-500">¥{{ s.balance_before }}</span>
                            <span class="text-stone-300 dark:text-stone-600 mx-1">→</span>
                            <span class="font-semibold">¥{{ s.balance_after }}</span>
                          </span>
                        </div>
                      </div>
                    </div>
                  </td>
                  <!-- 时间 -->
                  <td class="px-4 py-2 text-stone-500 text-sm tabular-nums whitespace-nowrap">{{ formatTime(t.transaction_time) }}</td>
                  <!-- 操作（仅当开启交易撤销时显示） -->
                  <td v-if="canVoid" class="px-4 py-2 text-center whitespace-nowrap">
                    <UButton
                      v-if="t.status !== 'voided'"
                      size="xs" variant="soft" color="error"
                      icon="i-lucide-rotate-ccw"
                      class="active:scale-95 transition-transform"
                      @click="openVoid(t)"
                    >撤销</UButton>
                  </td>
                </tr>
              </tbody>
            </table>
            <div class="px-4 py-2.5 border-t border-stone-200/60 dark:border-stone-800 bg-stone-50/30 dark:bg-stone-950/20 text-center">
              <button
                v-if="txItems.length < txTotal"
                type="button"
                class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md text-xs text-stone-600 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition cursor-pointer disabled:opacity-50"
                :disabled="txLoading"
                @click="fetchTx()"
              >
                <UIcon :name="txLoading ? 'i-lucide-loader-2' : 'i-lucide-chevron-down'" :class="['size-3.5', txLoading && 'animate-spin']" />
                加载更多（剩余 {{ txTotal - txItems.length }} 条）
              </button>
              <span v-else class="text-xs text-stone-400">已显示全部 {{ txTotal }} 条</span>
            </div>
          </div>
          <USkeleton v-if="txLoading && txItems.length === 0" class="h-64 rounded-2xl" />
          <EmptyState v-else-if="!txLoading && txItems.length === 0" icon="i-lucide-list" text="当前条件下无交易" />
        </div>
    </div>

    <!-- ============ 排行 ============ -->
    <div v-show="activeTab === 'ranking'">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mt-4">
          <!-- 项目排行 -->
          <section class="space-y-2">
            <div class="flex items-center gap-2 px-1">
              <UIcon name="i-lucide-package" class="size-4 text-stone-400" />
              <h3 class="text-sm font-medium text-stone-700 dark:text-stone-300">项目排行</h3>
            </div>
            <USkeleton v-if="svcLoading && svcRanking.length === 0" class="h-64 rounded-2xl" />
            <EmptyState v-else-if="svcRanking.length === 0" icon="i-lucide-package" text="无项目销售数据" />
            <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
              <table class="w-full text-sm">
                <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                  <tr>
                    <th class="text-left px-4 py-3 font-medium w-12">#</th>
                    <th class="text-left px-4 py-3 font-medium">项目</th>
                    <th class="text-right px-4 py-3 font-medium w-20">数量</th>
                    <th class="text-right px-4 py-3 font-medium w-28">金额</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(s, i) in svcRanking" :key="s.service_id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                    <td class="px-4 py-3 tabular-nums text-stone-500">{{ i + 1 }}</td>
                    <td class="px-4 py-3 truncate max-w-44">{{ s.service_name }}</td>
                    <td class="px-4 py-3 text-right tabular-nums text-stone-500">{{ s.total_count }}</td>
                    <td class="px-4 py-3 text-right tabular-nums font-medium">¥{{ s.total_sales }}</td>
                  </tr>
                </tbody>
              </table>
              <div v-if="svcHasMore" class="px-4 py-2.5 border-t border-stone-200/60 dark:border-stone-800 bg-stone-50/30 dark:bg-stone-950/20 text-center">
                <UButton size="xs" variant="ghost" color="neutral" :loading="svcLoading" @click="fetchSvc()">加载更多</UButton>
              </div>
            </div>
          </section>

          <!-- 会员排行 -->
          <section class="space-y-2">
            <div class="flex items-center gap-2 px-1">
              <UIcon name="i-lucide-users" class="size-4 text-stone-400" />
              <h3 class="text-sm font-medium text-stone-700 dark:text-stone-300">会员排行</h3>
            </div>
            <USkeleton v-if="memLoading && memberRanking.length === 0" class="h-64 rounded-2xl" />
            <EmptyState v-else-if="memberRanking.length === 0" icon="i-lucide-users" text="无会员消费数据" />
            <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
              <table class="w-full text-sm">
                <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                  <tr>
                    <th class="text-left px-4 py-3 font-medium w-12">#</th>
                    <th class="text-left px-4 py-3 font-medium">会员</th>
                    <th class="text-left px-4 py-3 font-medium">手机号</th>
                    <th class="text-right px-4 py-3 font-medium w-28">消费</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(m, i) in memberRanking" :key="m.id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                    <td class="px-4 py-3 tabular-nums text-stone-500">{{ i + 1 }}</td>
                    <td class="px-4 py-3 font-medium truncate max-w-32">{{ m.name }}</td>
                    <td class="px-4 py-3 text-stone-500 tabular-nums">{{ m.phone || '—' }}</td>
                    <td class="px-4 py-3 text-right tabular-nums font-medium">¥{{ m.total_consumption }}</td>
                  </tr>
                </tbody>
              </table>
              <div v-if="memHasMore" class="px-4 py-2.5 border-t border-stone-200/60 dark:border-stone-800 bg-stone-50/30 dark:bg-stone-950/20 text-center">
                <UButton size="xs" variant="ghost" color="neutral" :loading="memLoading" @click="fetchMember()">加载更多</UButton>
              </div>
            </div>
          </section>
        </div>
    </div>

    <!-- ============ 会员卡 ============ -->
    <div v-show="activeTab === 'cards'">
        <div class="space-y-4 mt-4">
          <USkeleton v-if="cardLoading" class="h-64 rounded-2xl" />
          <EmptyState v-else-if="cardSales.length === 0" icon="i-lucide-credit-card" text="区间内无会员卡销售" />
          <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
            <div class="px-5 py-3.5 border-b border-stone-200/60 dark:border-stone-800 flex items-baseline gap-3">
              <span class="text-xs text-stone-500">销售总额</span>
              <span class="text-2xl font-semibold tabular-nums tracking-tight">¥{{ cardTotal }}</span>
              <span class="ml-auto text-xs text-stone-400 tabular-nums">共 {{ cardSales.length }} 种</span>
            </div>
            <table class="w-full text-sm table-fixed">
              <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                <tr>
                  <th class="text-left px-4 py-3 font-medium whitespace-nowrap w-48">卡类型</th>
                  <th class="text-right px-4 py-3 font-medium whitespace-nowrap w-28">销售张数</th>
                  <th class="text-right px-4 py-3 font-medium whitespace-nowrap w-32">销售额</th>
                  <th class="text-left px-4 py-3 font-medium">占比</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in cardSales" :key="c.card_type_id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                  <td class="px-4 py-3 whitespace-nowrap truncate">{{ c.card_type_name }}</td>
                  <td class="px-4 py-3 text-right tabular-nums text-stone-500 whitespace-nowrap">{{ c.card_count }}</td>
                  <td class="px-4 py-3 text-right tabular-nums font-medium whitespace-nowrap">¥{{ c.total }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-3">
                      <div class="flex-1 h-1.5 rounded-full bg-stone-100 dark:bg-stone-800 overflow-hidden">
                        <div class="h-full bg-primary-500 rounded-full transition-all" :style="{ width: pctLabel(c.total, cardTotal) + '%' }" />
                      </div>
                      <span class="text-xs text-stone-500 tabular-nums w-12 text-right shrink-0">{{ pctLabel(c.total, cardTotal) }}%</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
    </div>

    <!-- ============ 支付 ============ -->
    <div v-show="activeTab === 'payment'">
        <div class="space-y-4 mt-4">
          <USkeleton v-if="payLoading" class="h-64 rounded-2xl" />
          <EmptyState v-else-if="paymentSum.length === 0" icon="i-lucide-wallet" text="区间内无支付记录" />
          <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
            <div class="px-5 py-3.5 border-b border-stone-200/60 dark:border-stone-800 flex items-baseline gap-3">
              <span class="text-xs text-stone-500">收款总额</span>
              <span class="text-2xl font-semibold tabular-nums tracking-tight">¥{{ payTotal }}</span>
              <span class="ml-auto text-xs text-stone-400 tabular-nums">共 {{ paymentSum.length }} 种</span>
            </div>
            <table class="w-full text-sm table-fixed">
              <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                <tr>
                  <th class="text-left px-4 py-3 font-medium whitespace-nowrap w-48">支付方式</th>
                  <th class="text-right px-4 py-3 font-medium whitespace-nowrap w-24">笔数</th>
                  <th class="text-right px-4 py-3 font-medium whitespace-nowrap w-32">金额</th>
                  <th class="text-left px-4 py-3 font-medium">占比</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in paymentSum" :key="p.payment_method_id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                  <td class="px-4 py-3 whitespace-nowrap truncate">{{ p.payment_method_name }}</td>
                  <td class="px-4 py-3 text-right tabular-nums text-stone-500 whitespace-nowrap">{{ p.tx_count }}</td>
                  <td class="px-4 py-3 text-right tabular-nums font-medium whitespace-nowrap">¥{{ p.total }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-3">
                      <div class="flex-1 h-1.5 rounded-full bg-stone-100 dark:bg-stone-800 overflow-hidden">
                        <div class="h-full bg-primary-500 rounded-full transition-all" :style="{ width: pctLabel(p.total, payTotal) + '%' }" />
                      </div>
                      <span class="text-xs text-stone-500 tabular-nums w-12 text-right shrink-0">{{ pctLabel(p.total, payTotal) }}%</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
    </div>

    <!-- ============ 挂账 ============ -->
    <div v-show="activeTab === 'pending'">
        <div class="space-y-4 mt-4">
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div
              v-for="k in pendingKpis"
              :key="k.label"
              class="p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800"
            >
              <div class="text-xs text-stone-500">{{ k.label }}</div>
              <div v-if="pendingLoading && !pendingStats" class="h-7 mt-1 w-20 rounded-md bg-stone-100 dark:bg-stone-800 animate-pulse" />
              <div v-else class="text-2xl font-semibold tabular-nums mt-1" :class="k.accent">{{ k.value }}</div>
            </div>
          </div>

          <USkeleton v-if="pendingLoading && !pendingStats" class="h-64 rounded-2xl" />
          <EmptyState v-else-if="pendingStats && pendingStats.data.length === 0" icon="i-lucide-clock" text="当前无挂账记录" />
          <div v-else-if="pendingStats" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                <tr>
                  <th class="text-left px-4 py-3 font-medium">会员</th>
                  <th class="text-left px-4 py-3 font-medium">手机号</th>
                  <th class="text-right px-4 py-3 font-medium w-20">记录数</th>
                  <th class="text-right px-4 py-3 font-medium w-32">挂账总额</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in pendingStats.data" :key="p.id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                  <td class="px-4 py-3 font-medium">{{ p.name }}</td>
                  <td class="px-4 py-3 text-stone-500 tabular-nums">{{ p.phone || '—' }}</td>
                  <td class="px-4 py-3 text-right tabular-nums text-stone-500">{{ p.record_count }}</td>
                  <td class="px-4 py-3 text-right tabular-nums font-medium text-warning-700">¥{{ p.total_pending }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
    </div>

    <!-- ============ 提醒 ============ -->
    <div v-show="activeTab === 'reminders'">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mt-4">
          <!-- 生日提醒 -->
          <section class="space-y-2">
            <div class="flex items-center gap-2 px-1">
              <UIcon name="i-lucide-cake" class="size-4 text-stone-400" />
              <h3 class="text-sm font-medium text-stone-700 dark:text-stone-300">生日提醒</h3>
              <span class="text-xs text-stone-400">未来 15 天</span>
            </div>
            <USkeleton v-if="birthdayLoading" class="h-40 rounded-2xl" />
            <EmptyState v-else-if="birthdays.length === 0" icon="i-lucide-cake" text="近期无生日会员" />
            <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
              <table class="w-full text-sm">
                <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                  <tr>
                    <th class="text-left px-4 py-3 font-medium">姓名</th>
                    <th class="text-left px-4 py-3 font-medium">手机号</th>
                    <th class="text-right px-4 py-3 font-medium w-28">生日</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="m in birthdays" :key="m.id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                    <td class="px-4 py-3 font-medium">{{ m.name }}</td>
                    <td class="px-4 py-3 text-stone-500 tabular-nums">{{ m.phone || '—' }}</td>
                    <td class="px-4 py-3 text-right tabular-nums text-stone-500 whitespace-nowrap">{{ formatDate(m.birthday) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <!-- 沉睡会员 -->
          <section class="space-y-2">
            <div class="flex items-center gap-2 px-1">
              <UIcon name="i-lucide-moon" class="size-4 text-stone-400" />
              <h3 class="text-sm font-medium text-stone-700 dark:text-stone-300">沉睡会员</h3>
              <span class="text-xs text-stone-400">90 天未消费</span>
            </div>
            <USkeleton v-if="sleepingLoading && sleeping.length === 0" class="h-40 rounded-2xl" />
            <EmptyState v-else-if="sleeping.length === 0" icon="i-lucide-moon" text="无沉睡会员" />
            <div v-else class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 overflow-hidden">
              <table class="w-full text-sm">
                <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
                  <tr>
                    <th class="text-left px-4 py-3 font-medium">姓名</th>
                    <th class="text-left px-4 py-3 font-medium">手机号</th>
                    <th class="text-right px-4 py-3 font-medium w-28">最后消费</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="m in sleeping" :key="m.id" class="border-t border-stone-100 dark:border-stone-800/60 even:bg-stone-50/40 dark:even:bg-stone-950/30 hover:bg-primary-50/30 dark:hover:bg-primary-950/10 transition-colors">
                    <td class="px-4 py-3 font-medium">{{ m.name }}</td>
                    <td class="px-4 py-3 text-stone-500 tabular-nums">{{ m.phone || '—' }}</td>
                    <td class="px-4 py-3 text-right tabular-nums text-stone-500 whitespace-nowrap">{{ m.last_visit ? formatDate(m.last_visit) : '从未' }}</td>
                  </tr>
                </tbody>
              </table>
              <div v-if="sleepingHasMore" class="px-4 py-2.5 border-t border-stone-200/60 dark:border-stone-800 bg-stone-50/30 dark:bg-stone-950/20 text-center">
                <UButton size="xs" variant="ghost" color="neutral" :loading="sleepingLoading" @click="fetchSleeping()">加载更多</UButton>
              </div>
            </div>
          </section>
        </div>
    </div>

    <!-- 撤销弹窗 -->
    <UModal v-model:open="voidOpen" title="撤销交易" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <div class="space-y-3">
          <p class="text-sm text-stone-600 dark:text-stone-400">将交易 <strong>{{ voiding?.id.slice(0, 8) }}</strong> 设为已撤销，关联的卡余额 / 挂账会自动恢复。</p>
          <UFormField label="撤销原因" required>
            <UTextarea v-model="voidReason" :rows="2" class="w-full" />
          </UFormField>
          <UAlert v-if="voidErr" :description="voidErr" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="voidOpen = false">取消</UButton>
          <UButton color="error" :loading="voidLoading" :disabled="!voidReason" @click="doVoid">确认撤销</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'at-least-admin' })

import { CalendarDate } from '@internationalized/date'

const api = useApi()
const route = useRoute()
const router = useRouter()
const { canVoid, ensureFetched: ensureVoidFetched } = useVoidEnabled()
onMounted(() => { ensureVoidFetched() })

// ===== 日期与 Preset =====
const today = new Date()
const startDate = ref(formatDateOnly(today))
const endDate = ref(formatDateOnly(today))
const preset = ref<string>('day')

// 列表默认/加载更多条数：参考 2025 桌面 SaaS 基准（15–30），取中位偏大 20
const PAGE_SIZE = 20

const presets = [
  { key: 'day',     label: '当日' },
  { key: 'week',    label: '当周' },
  { key: 'month',   label: '当月' },
  { key: 'quarter', label: '当季' },
  { key: 'year',    label: '当年' },
]
function firstDayOfMonth(d: Date) { return new Date(d.getFullYear(), d.getMonth(), 1) }
function formatDate(s: string) { return s ? s.slice(0, 10) : '' }
function fmtMoney(n: string | number | undefined | null) {
  if (n == null || n === '') return '¥0.00'
  const v = parseFloat(String(n))
  return '¥' + (Number.isFinite(v) ? v.toFixed(2) : '0.00')
}
function pctLabel(v: string | number, total: string | number) {
  const t = parseFloat(String(total))
  if (!t) return '0'
  return (parseFloat(String(v)) / t * 100).toFixed(1)
}

function calToStr(d: any): string {
  return `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`
}
function strToCal(s: string): CalendarDate | undefined {
  if (!s) return undefined
  const [y, m, d] = s.split('-').map(Number)
  if (!y || !m || !d) return undefined
  return new CalendarDate(y, m, d)
}
const rangeCalValue = computed(() => {
  const s = strToCal(startDate.value)
  const e = strToCal(endDate.value)
  return (s && e) ? { start: s, end: e } : undefined
})
function onRangeUpdate(v: any) {
  if (!v?.start || !v?.end) return
  startDate.value = calToStr(v.start)
  endDate.value = calToStr(v.end)
  preset.value = ''
  onQuery()
}

function applyPreset(p: typeof presets[number]) {
  preset.value = p.key
  const now = new Date()
  endDate.value = formatDateOnly(now)
  if (p.key === 'day') startDate.value = formatDateOnly(now)
  else if (p.key === 'week') {
    const s = new Date(now); s.setDate(s.getDate() - (s.getDay() === 0 ? 6 : s.getDay() - 1))
    startDate.value = formatDateOnly(s)
  }
  else if (p.key === 'month') startDate.value = formatDateOnly(firstDayOfMonth(now))
  else if (p.key === 'quarter') {
    const q = Math.floor(now.getMonth() / 3) * 3
    startDate.value = formatDateOnly(new Date(now.getFullYear(), q, 1))
  }
  else if (p.key === 'year') startDate.value = formatDateOnly(new Date(now.getFullYear(), 0, 1))
  onQuery()
}

// ===== Tabs =====
const tabs = [
  { label: '流水',   slot: 'transactions' as const, value: 'transactions', icon: 'i-lucide-list' },
  { label: '排行',   slot: 'ranking' as const,      value: 'ranking',      icon: 'i-lucide-trophy' },
  { label: '会员卡', slot: 'cards' as const,        value: 'cards',        icon: 'i-lucide-credit-card' },
  { label: '支付',   slot: 'payment' as const,      value: 'payment',      icon: 'i-lucide-wallet' },
  { label: '挂账',   slot: 'pending' as const,      value: 'pending',      icon: 'i-lucide-clock' },
  { label: '提醒',   slot: 'reminders' as const,    value: 'reminders',    icon: 'i-lucide-bell' },
]
const tabOrder = tabs.map(t => t.value)
const initialTab = typeof route.query.tab === 'string' && tabOrder.includes(route.query.tab)
  ? String(route.query.tab)
  : 'transactions'
const activeTab = ref<string>(initialTab)
watch(activeTab, (v) => {
  router.replace({ query: { ...route.query, tab: v } })
  ensureTabLoaded(v)
})

// ===== Overview =====
const biz = ref<any>(null)
const overviewLoading = ref(false)
// 图标配色：主 KPI（营业总额）primary，其余 stone；指标区分靠图标本身
const overviewKpis = computed(() => [
  { label: '营业总额', value: fmtMoney(biz.value?.total_revenue),        icon: 'i-lucide-coins',         iconColor: 'text-primary-500' },
  { label: '实收',     value: fmtMoney(biz.value?.sale_revenue),         icon: 'i-lucide-banknote',      iconColor: 'text-stone-400' },
  { label: '卡耗',     value: fmtMoney(biz.value?.card_consumption),     icon: 'i-lucide-credit-card',   iconColor: 'text-stone-400' },
  { label: '充值',     value: fmtMoney(biz.value?.recharge_amount),      icon: 'i-lucide-arrow-up-right',iconColor: 'text-stone-400' },
  { label: '客数',     value: biz.value ? String(biz.value.customer_count) : '—', icon: 'i-lucide-users',         iconColor: 'text-stone-400' },
  { label: '客单价',   value: fmtMoney(biz.value?.average_order_value),  icon: 'i-lucide-trending-up',   iconColor: 'text-stone-400' },
])
async function fetchBiz() {
  overviewLoading.value = true
  try {
    biz.value = await api(`/api/reports/business?${dateQS()}`)
  } finally { overviewLoading.value = false }
}

// ===== Transactions =====
interface CardSnapshot { card_id: string; card_type_name: string; balance_before: string; balance_after: string; delta: string; change_type: string }
interface Tx {
  id: string
  kind: 'sale' | 'recharge' | 'credit_settlement'
  status: 'completed' | 'voided'
  member_id: string | null; member_name: string | null
  customer_name: string | null
  card_id: string | null; card_type_name: string | null
  total_amount: string; actual_paid_amount: string; discount_amount: string
  transaction_time: string; summary: string | null
  item_qty: number
  card_snapshots: CardSnapshot[] | null
}
const txItems = ref<Tx[]>([])
const txTotal = ref(0)
const txLoading = ref(false)
const txPage = ref(1)
const txKind = ref<'' | 'sale' | 'recharge' | 'credit_settlement'>('')
const txFilters = [
  { key: '',                  label: '全部' },
  { key: 'sale',              label: '消费' },
  { key: 'recharge',          label: '办卡 / 充值' },
  { key: 'credit_settlement', label: '清账' },
]
function kindLabel(k: string) { return (TX_KIND_LABEL as Record<string, string>)[k] ?? k }
function kindColor(k: string) { return (TX_KIND_COLOR as Record<string, 'primary' | 'info' | 'neutral'>)[k] ?? 'neutral' }

async function fetchTx(reset = false) {
  if (reset) { txPage.value = 1; txItems.value = [] }
  txLoading.value = true
  try {
    // 日界按本地时区（与后端 Asia/Shanghai 业务日一致）；后端区间左闭右开，终点取次日本地 0 点
    const endNext = new Date(`${endDate.value}T00:00:00`)
    endNext.setDate(endNext.getDate() + 1)
    const q = new URLSearchParams({
      start_date: new Date(`${startDate.value}T00:00:00`).toISOString(),
      end_date: endNext.toISOString(),
      page: String(txPage.value),
      limit: String(PAGE_SIZE),
      include_voided: '1',
    })
    if (txKind.value) q.set('kind', txKind.value)
    const data = await api<{ items: Tx[]; total: number }>(`/api/transactions?${q}`)
    txItems.value = reset ? data.items : [...txItems.value, ...data.items]
    txTotal.value = data.total
    txPage.value++
  } finally { txLoading.value = false }
}

const voidOpen = ref(false)
const voiding = ref<Tx | null>(null)
const voidReason = ref('')
const voidLoading = ref(false)
const voidErr = ref('')
function openVoid(t: Tx) { voiding.value = t; voidReason.value = ''; voidErr.value = ''; voidOpen.value = true }
async function doVoid() {
  if (!voiding.value) return
  voidLoading.value = true; voidErr.value = ''
  try {
    await api(`/api/transactions/${voiding.value.id}/void`, { method: 'POST', body: { reason: voidReason.value } })
    voiding.value.status = 'voided'
    voidOpen.value = false
    fetchBiz()
  } catch (e: any) {
    voidErr.value = e?.data?.message || '撤销失败（请确认在"设置 → 交易撤销"里开启）'
  } finally { voidLoading.value = false }
}

// ===== Ranking =====
const svcRanking = ref<any[]>([])
const svcLoading = ref(false)
const svcPage = ref(1)
const svcHasMore = ref(false)
async function fetchSvc(reset = false) {
  if (reset) { svcPage.value = 1; svcRanking.value = [] }
  svcLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/service-ranking?${dateQS()}&limit=${PAGE_SIZE}&page=${svcPage.value}`)
    const items = data.data || []
    svcRanking.value = reset ? items : [...svcRanking.value, ...items]
    svcHasMore.value = items.length === PAGE_SIZE
    svcPage.value++
  } finally { svcLoading.value = false }
}

const memberRanking = ref<any[]>([])
const memLoading = ref(false)
const memPage = ref(1)
const memHasMore = ref(false)
async function fetchMember(reset = false) {
  if (reset) { memPage.value = 1; memberRanking.value = [] }
  memLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/member-ranking?${dateQS()}&limit=${PAGE_SIZE}&page=${memPage.value}`)
    const items = data.data || []
    memberRanking.value = reset ? items : [...memberRanking.value, ...items]
    memHasMore.value = items.length === PAGE_SIZE
    memPage.value++
  } finally { memLoading.value = false }
}

// ===== Cards =====
const cardSales = ref<any[]>([])
const cardLoading = ref(false)
const cardTotal = computed(() =>
  cardSales.value.reduce((s, c) => s + parseFloat(c.total || '0'), 0).toFixed(2),
)
async function fetchCardSales() {
  cardLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/card-sales-summary?${dateQS()}`)
    cardSales.value = data.data || []
  } finally { cardLoading.value = false }
}

// ===== Payment =====
const paymentSum = ref<any[]>([])
const payLoading = ref(false)
const payTotal = computed(() =>
  paymentSum.value.reduce((s, p) => s + parseFloat(p.total || '0'), 0).toFixed(2),
)
async function fetchPayment() {
  payLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/payment-summary?${dateQS()}`)
    paymentSum.value = data.data || []
  } finally { payLoading.value = false }
}

// ===== Pending =====
const pendingStats = ref<any>(null)
const pendingLoading = ref(false)
const pendingKpis = computed(() => {
  const s = pendingStats.value?.summary
  return [
    { label: '挂账总额',   value: s ? fmtMoney(s.total_amount)   : '—', accent: 'text-warning-700' },
    { label: '挂账会员数', value: s ? String(s.member_count)     : '—', accent: '' },
    { label: '记录数',     value: s ? String(s.record_count)     : '—', accent: '' },
    { label: '平均金额',   value: s ? fmtMoney(s.average_amount) : '—', accent: '' },
  ]
})
async function fetchPending() {
  pendingLoading.value = true
  try {
    pendingStats.value = await api(`/api/reports/pending-stats?limit=${PAGE_SIZE}`)
  } finally { pendingLoading.value = false }
}

// ===== Reminders =====
const birthdays = ref<any[]>([])
const birthdayLoading = ref(false)
async function fetchBirthday() {
  birthdayLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/birthday-reminders`)
    birthdays.value = data.data || []
  } finally { birthdayLoading.value = false }
}

const sleeping = ref<any[]>([])
const sleepingLoading = ref(false)
const sleepingPage = ref(1)
const sleepingHasMore = ref(false)
async function fetchSleeping(reset = false) {
  if (reset) { sleepingPage.value = 1; sleeping.value = [] }
  sleepingLoading.value = true
  try {
    const data = await api<{ data: any[] }>(`/api/reports/sleeping-members?limit=${PAGE_SIZE}&page=${sleepingPage.value}`)
    const items = data.data || []
    sleeping.value = reset ? items : [...sleeping.value, ...items]
    sleepingHasMore.value = items.length === PAGE_SIZE
    sleepingPage.value++
  } finally { sleepingLoading.value = false }
}

// ===== Tab 懒加载 =====
const loadedTabs = new Set<string>()
function ensureTabLoaded(tab: string) {
  if (loadedTabs.has(tab)) return
  loadedTabs.add(tab)
  if (tab === 'transactions') { fetchBiz(); fetchTx(true) }
  else if (tab === 'ranking') { fetchSvc(true); fetchMember(true) }
  else if (tab === 'cards') fetchCardSales()
  else if (tab === 'payment') fetchPayment()
  else if (tab === 'pending') fetchPending()
  else if (tab === 'reminders') { fetchBirthday(); fetchSleeping(true) }
}

function dateQS() { return `start_date=${startDate.value}&end_date=${endDate.value}` }

function onQuery() {
  loadedTabs.clear()
  biz.value = null
  txItems.value = []; txPage.value = 1; txTotal.value = 0
  svcRanking.value = []; svcPage.value = 1
  memberRanking.value = []; memPage.value = 1
  cardSales.value = []
  paymentSum.value = []
  pendingStats.value = null
  birthdays.value = []
  sleeping.value = []; sleepingPage.value = 1
  ensureTabLoaded(activeTab.value)
}

onMounted(() => { ensureTabLoaded(activeTab.value) })
</script>
