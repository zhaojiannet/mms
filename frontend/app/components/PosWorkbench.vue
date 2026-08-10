<template>
  <div>
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-6">
      <!-- 左：表单（占 3 列），各项之间用极浅分割线 -->
      <section class="lg:col-span-3 divide-y divide-stone-200/50 dark:divide-stone-800/40 [&>div]:py-5 [&>div:first-child]:pt-0 [&>div:last-child]:pb-0">

        <!-- 会员 -->
        <div>
          <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />会员</label>
          <div class="mt-1.5">

            <!-- 已选会员卡片：浅 primary 背景 + 头像 + 完整信息 -->
            <div v-if="member" class="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-primary-50/60 dark:bg-primary-950/30 ring-1 ring-primary-500/30">
              <div class="size-10 shrink-0 rounded-full bg-primary-500 text-white flex items-center justify-center text-base font-semibold">
                {{ avatarChar(member.name) }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-base font-semibold truncate text-stone-900 dark:text-stone-100">{{ member.name }}</span>
                  <UBadge v-if="member.status && member.status !== 'active'" :label="memberStatusLabel(member.status)" color="warning" variant="soft" size="xs" />
                  <span v-if="member.phone" class="text-sm text-stone-600 dark:text-stone-400 tabular-nums">{{ formatPhone(member.phone) }}</span>
                </div>
                <div class="mt-0.5 flex items-center gap-3 text-xs">
                  <span class="inline-flex items-baseline gap-1.5">
                    <span class="text-stone-500">余额</span>
                    <span class="tabular-nums font-semibold text-primary-600 dark:text-primary-400">¥{{ member.total_balance ?? '0.00' }}</span>
                  </span>
                  <span class="inline-flex items-baseline gap-1 text-stone-500">
                    共 {{ memberCards.length }} 张卡
                  </span>
                  <span v-if="parseFloat(member.total_pending ?? '0') > 0" class="inline-flex items-baseline gap-1.5">
                    <span class="text-stone-500">未结挂账</span>
                    <span class="tabular-nums font-medium text-warning-600 dark:text-warning-400">¥{{ member.total_pending }}</span>
                  </span>
                </div>
                <div
                  v-if="parseFloat(member.total_pending ?? '0') > 0 && parseFloat(member.total_balance ?? '0') <= parseFloat(member.total_pending ?? '0')"
                  class="mt-1 text-xs text-error-600 dark:text-error-400 flex items-center gap-1"
                >
                  <UIcon name="i-lucide-alert-triangle" class="size-3.5 shrink-0" />
                  余额不足以覆盖挂账，请提醒充值
                </div>
              </div>
              <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-x" @click="clearMember" />
            </div>

            <!-- 非会员 -->
            <div v-else-if="walkInName" class="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-stone-50 dark:bg-stone-800/60 ring-1 ring-stone-200 dark:ring-stone-700">
              <div class="size-10 shrink-0 rounded-full bg-stone-200 dark:bg-stone-700 text-stone-500 flex items-center justify-center">
                <UIcon name="i-lucide-user-round" class="size-5" />
              </div>
              <div class="flex-1 flex items-center gap-2">
                <span class="text-base font-semibold">{{ walkInName }}</span>
                <UBadge label="非会员" color="neutral" variant="soft" size="xs" />
              </div>
              <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-x" @click="walkInName = ''" />
            </div>

            <!-- 搜索框 + dropdown -->
            <div v-else class="relative">
              <UInput
                v-model="memberSearch"
                icon="i-lucide-search"
                placeholder="搜索会员（姓名 / 手机号）"
                size="lg"
                class="w-full"
                :ui="{ trailing: 'pe-1' }"
                @update:model-value="debouncedSearch"
                @keydown.enter="handleEnter"
              >
                <template v-if="memberSearch?.length" #trailing>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="sm"
                    icon="i-lucide-circle-x"
                    aria-label="清空搜索"
                    @click="memberSearch = ''; debouncedSearch()"
                  />
                </template>
              </UInput>
              <div
                v-if="memberOptions.length > 0"
                class="absolute left-0 right-0 top-full mt-1 z-20 max-h-80 overflow-y-auto rounded-lg bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-lg"
              >
                <button
                  v-for="m in memberOptions" :key="m.id"
                  type="button"
                  class="w-full text-left px-3 py-2.5 hover:bg-primary-50/40 dark:hover:bg-primary-950/20 flex items-center gap-3 border-b last:border-0 border-stone-100 dark:border-stone-800 transition"
                  @click="pickMember(m)"
                >
                  <div class="size-9 shrink-0 rounded-full bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-300 flex items-center justify-center text-sm font-semibold">
                    {{ avatarChar(m.name) }}
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-1.5">
                      <span class="text-sm font-medium truncate">{{ m.name }}</span>
                      <UBadge v-if="m.status && m.status !== 'active'" :label="memberStatusLabel(m.status)" color="warning" variant="soft" size="xs" />
                    </div>
                    <div class="text-xs text-stone-500 dark:text-stone-400 tabular-nums">{{ formatPhone(m.phone) }}</div>
                  </div>
                  <div class="shrink-0 text-right">
                    <div v-if="parseFloat(m.total_balance ?? '0') > 0" class="text-sm font-semibold tabular-nums text-primary-600 dark:text-primary-400">
                      <span class="text-xs text-stone-400 mr-1.5 font-normal">余额</span>¥{{ m.total_balance }}
                    </div>
                    <div v-else class="text-xs text-stone-400">未办卡</div>
                    <div v-if="parseFloat(m.total_pending ?? '0') > 0" class="text-xs text-warning-600 dark:text-warning-400 tabular-nums">
                      <span class="text-stone-400 font-normal mr-1.5">挂账</span>¥{{ m.total_pending }}
                    </div>
                    <div
                      v-if="parseFloat(m.total_pending ?? '0') > 0 && parseFloat(m.total_balance ?? '0') <= parseFloat(m.total_pending ?? '0')"
                      class="text-xs text-error-500 mt-0.5"
                    >请提醒充值</div>
                  </div>
                </button>
              </div>
              <div
                v-else-if="memberSearch.trim().length > 0"
                class="absolute left-0 right-0 top-full mt-1 z-20 px-3 py-2.5 rounded-lg bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-lg text-sm text-stone-500"
              >
                无匹配，按 <kbd class="px-1.5 py-0.5 rounded bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-300 text-xs">Enter</kbd> 以
                <span class="font-medium text-stone-700 dark:text-stone-200">「{{ memberSearch }}」</span> 非会员消费
              </div>
            </div>
          </div>
        </div>

        <!-- 服务：label + 右侧紧凑搜索 + 8 个常用卡片 -->
        <div>
          <div class="flex items-center justify-between gap-3 mb-2">
            <div class="flex items-center gap-2">
              <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />服务</label>
              <span v-if="items.length > 0" class="text-xs text-stone-500">已选 {{ items.length }} 项</span>
            </div>
            <div class="relative">
              <UInput
                v-model="svcSearch"
                icon="i-lucide-search"
                placeholder="找其他项目"
                size="sm"
                class="w-36"
                :ui="{ base: 'rounded-full', trailing: 'pe-1' }"
                @keydown.enter="addFirstMatch"
                @focus="onSvcFocus"
                @click="onSvcFocus"
                @blur="onSvcBlur"
              >
                <template v-if="svcSearch?.length" #trailing>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="sm"
                    icon="i-lucide-circle-x"
                    aria-label="清空搜索"
                    @mousedown.prevent
                    @click="svcSearch = ''"
                  />
                </template>
              </UInput>
              <div
                v-if="svcDropdownOpen && filteredServices.length > 0"
                class="absolute right-0 top-full mt-1 w-72 z-20 max-h-72 overflow-y-auto rounded-md bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-lg"
              >
                <button
                  v-for="s in filteredServices" :key="s.id"
                  type="button"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-stone-50 dark:hover:bg-stone-800 flex items-center justify-between gap-3 border-b last:border-0 border-stone-100 dark:border-stone-800"
                  @mousedown.prevent="addItem(s)"
                >
                  <span class="truncate">
                    <span>{{ s.name }}</span>
                    <UBadge v-if="s.no_discount" label="不折" color="warning" variant="soft" size="xs" class="ml-1.5" />
                  </span>
                  <span class="tabular-nums shrink-0 text-sm">
                    <span class="text-stone-700 dark:text-stone-300 font-medium">¥{{ s.price }}</span>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <!-- 常用 8 个卡片 -->
          <div v-if="quickServices.length > 0" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
            <button
              v-for="s in quickServices" :key="s.id"
              type="button"
              class="group relative px-3 py-2.5 rounded-lg text-left bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-xs hover:shadow-sm hover:ring-2 hover:ring-primary-500/40 hover:bg-primary-50/30 dark:hover:bg-primary-950/20 active:scale-[0.97] transition"
              @click="addItem(s)"
            >
              <UBadge v-if="s.no_discount" label="不折" color="warning" variant="soft" size="xs" class="absolute top-1 right-1" />
              <div class="text-sm font-medium truncate pr-6 text-stone-900 dark:text-stone-100">{{ s.name }}</div>
              <div class="mt-1 flex items-baseline gap-1.5">
                <span class="text-base font-semibold tabular-nums text-stone-900 dark:text-stone-100">¥{{ s.price }}</span>
                <span v-if="previewRate < 1 && !s.no_discount" class="text-xs tabular-nums text-primary-600 dark:text-primary-400">→ ¥{{ (parseFloat(s.price) * previewRate).toFixed(2) }}</span>
              </div>
            </button>
          </div>
        </div>

        <!-- 员工：label + 右侧紧凑搜索 + 5 个常用卡片 -->
        <div>
          <div class="flex items-center justify-between gap-3 mb-2">
            <div class="flex items-center gap-2">
              <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />服务员工</label>
              <span v-if="selectedStaff" class="text-xs text-stone-500">已选 {{ selectedStaffName }}</span>
            </div>
            <div v-if="staffList.length > 0" class="relative">
              <UInput
                v-model="staffSearch"
                icon="i-lucide-search"
                placeholder="找其他员工"
                size="sm"
                class="w-36"
                :ui="{ base: 'rounded-full', trailing: 'pe-1' }"
                @focus="onStaffFocus"
                @click="onStaffFocus"
                @blur="onStaffBlur"
              >
                <template v-if="staffSearch?.length" #trailing>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="sm"
                    icon="i-lucide-circle-x"
                    aria-label="清空搜索"
                    @mousedown.prevent
                    @click="staffSearch = ''"
                  />
                </template>
              </UInput>
              <div
                v-if="staffDropdownOpen && filteredStaff.length > 0"
                class="absolute right-0 top-full mt-1 w-60 z-20 max-h-72 overflow-y-auto rounded-md bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-lg"
              >
                <button
                  v-for="s in filteredStaff" :key="s.id"
                  type="button"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-stone-50 dark:hover:bg-stone-800 flex items-center justify-between gap-3 border-b last:border-0 border-stone-100 dark:border-stone-800"
                  @mousedown.prevent="pickStaff(s.id)"
                >
                  <span>
                    <span class="font-medium">{{ s.name }}</span>
                    <span class="ml-1 text-stone-400">{{ s.position }}</span>
                  </span>
                  <UIcon v-if="selectedStaff === s.id" name="i-lucide-check" class="size-4 text-primary-500" />
                </button>
              </div>
            </div>
          </div>

          <!-- 常用 5 个员工卡片 -->
          <div v-if="quickStaff.length > 0" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2">
            <button
              v-for="s in quickStaff" :key="s.id"
              type="button"
              :class="[
                'group relative px-3 py-2.5 rounded-lg text-left transition active:scale-[0.97] shadow-xs hover:shadow-sm',
                selectedStaff === s.id
                  ? 'bg-primary-50 dark:bg-primary-950/40 ring-2 ring-primary-500/60'
                  : 'bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 hover:ring-2 hover:ring-primary-500/40 hover:bg-primary-50/30 dark:hover:bg-primary-950/20',
              ]"
              @click="toggleStaff(s.id)"
            >
              <span
                :class="[
                  'absolute top-1.5 right-1.5 size-4 rounded-full inline-flex items-center justify-center transition',
                  selectedStaff === s.id ? 'bg-primary-500 text-white' : 'bg-stone-100 dark:bg-stone-800 text-transparent',
                ]"
              >
                <UIcon name="i-lucide-check" class="size-3" />
              </span>
              <div class="text-sm font-medium truncate pr-6 text-stone-900 dark:text-stone-100">{{ s.name }}</div>
              <div class="text-xs text-stone-500 truncate">{{ s.position }}</div>
            </button>
          </div>
        </div>

        <!-- 支付方式（上下都不显示分割线，间距收紧） -->
        <div class="border-y-0! py-3!">
          <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />支付方式</label>
          <div class="mt-1.5 flex flex-wrap gap-1.5">
            <button
              v-for="pm in paymentMethods" :key="pm.id"
              type="button"
              :disabled="pm.name === '会员卡' && (!member || memberCards.length === 0)"
              :class="[
                'px-3 py-1.5 rounded-md text-sm transition',
                selectedPm === pm.id
                  ? 'bg-primary-50 dark:bg-primary-950/40 text-primary-700 dark:text-primary-300 ring-1 ring-primary-500/40 font-medium'
                  : 'bg-white dark:bg-stone-900 text-stone-700 dark:text-stone-300 ring-1 ring-stone-200 dark:ring-stone-700 hover:ring-primary-500/40 hover:bg-primary-50/30 dark:hover:bg-primary-950/20',
                pm.name === '会员卡' && (!member || memberCards.length === 0) ? 'opacity-40 cursor-not-allowed' : ''
              ]"
              @click="selectPm(pm.id)"
            >{{ pm.name }}</button>
          </div>

          <!-- 会员卡支付：默认自动按最优折扣分配；toggle 开启手动选卡 -->
          <div v-if="isMemberCardPay && member && memberCards.length > 1" class="mt-3 rounded-md bg-stone-50/60 dark:bg-stone-800/40 ring-1 ring-stone-200/60 dark:ring-stone-700 p-2.5">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <USwitch v-model="manualCardMode" size="xs" />
                <span class="text-sm text-stone-700 dark:text-stone-300">手动选卡</span>
              </div>
              <span class="text-xs text-stone-400">{{ manualCardMode ? '请选择下方卡片' : '已按最优折扣自动分配' }}</span>
            </div>
            <div v-if="manualCardMode" class="mt-2.5 grid grid-cols-2 sm:grid-cols-3 gap-1.5">
              <button
                v-for="c in memberCards" :key="c.id"
                type="button"
                :class="[
                  'group relative px-2.5 py-2 rounded-md text-left transition active:scale-[0.97]',
                  manualCardId === c.id
                    ? 'bg-primary-50 dark:bg-primary-950/40 ring-2 ring-primary-500/60'
                    : 'bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 hover:ring-primary-500/40',
                ]"
                @click="manualCardId = c.id"
              >
                <span
                  :class="[
                    'absolute top-1 right-1 size-3.5 rounded-full inline-flex items-center justify-center transition',
                    manualCardId === c.id ? 'bg-primary-500 text-white' : 'bg-stone-100 dark:bg-stone-800 text-transparent',
                  ]"
                >
                  <UIcon name="i-lucide-check" class="size-2.5" />
                </span>
                <div class="text-xs font-medium truncate pr-4">{{ c.card_type_name }}</div>
                <div class="mt-0.5 flex items-baseline gap-1">
                  <span class="text-sm font-semibold tabular-nums text-primary-600 dark:text-primary-400">¥{{ c.balance }}</span>
                  <span class="text-xs text-stone-400">{{ displayRate(c.final_discount_rate) }}</span>
                </div>
              </button>
            </div>
          </div>
        </div>

        <!-- 交易时间 + 备注：4.5:5.5 横向并列（与上方支付方式之间不显示分割线，间距收紧） -->
        <div class="grid grid-cols-1 sm:[grid-template-columns:9fr_11fr] gap-4 border-y-0! pt-3!">
          <div>
            <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />交易时间</label>
            <div class="mt-1.5">
              <UPopover v-model:open="timePopoverOpen" :ui="{ content: 'p-4 w-auto' }">
                <button
                  type="button"
                  class="w-full flex items-center justify-between gap-2 px-3 py-2 rounded-md text-base md:text-sm bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 hover:ring-primary-500/40 transition cursor-pointer"
                >
                  <span class="flex items-center gap-2">
                    <UIcon name="i-lucide-calendar-clock" class="size-4 text-stone-400" />
                    <span class="tabular-nums">{{ displayTimeLabel }}</span>
                    <UBadge v-if="useCustomTime" label="补录" color="warning" variant="soft" size="xs" />
                  </span>
                  <span class="text-xs text-stone-400">点击修改</span>
                </button>
                <template #content>
                  <div class="space-y-3">
                    <UCalendar v-model="customDate" />
                    <div class="flex items-center justify-center gap-2">
                      <USelect v-model="customHour" :items="hourOptions" size="sm" class="w-20" />
                      <span class="text-stone-400">:</span>
                      <USelect v-model="customMinute" :items="minuteOptions" size="sm" class="w-20" />
                    </div>
                    <div class="flex justify-between gap-2">
                      <UButton size="sm" variant="ghost" color="neutral" @click="resetTimeToNow">恢复为当前时间</UButton>
                      <UButton size="sm" color="primary" @click="applyCustomTime">应用</UButton>
                    </div>
                  </div>
                </template>
              </UPopover>
            </div>
          </div>

          <div>
            <label class="inline-flex items-center gap-2 text-base font-medium text-stone-900 dark:text-stone-100"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />备注</label>
            <UInput
              v-model="notes"
              placeholder="请输入备注信息（可选）"
              icon="i-lucide-pencil"
              size="lg"
              class="w-full mt-1.5"
              :ui="{
                base: 'bg-white dark:bg-stone-900 ring-stone-200 dark:ring-stone-700',
                leadingIcon: 'size-4 text-stone-400',
              }"
            />
          </div>
        </div>
      </section>

      <!-- 右：消费详情 + 结算（占 2 列） -->
      <aside class="lg:col-span-2">
        <div class="rounded-lg ring-1 ring-stone-200 dark:ring-stone-800 bg-white dark:bg-stone-900 sticky top-6">
          <div class="px-4 py-3 border-b border-stone-200 dark:border-stone-800 flex items-center justify-between">
            <h2 class="inline-flex items-center gap-2 text-base font-semibold">
              <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
              消费详情
            </h2>
            <button
              v-if="items.length > 0"
              type="button"
              class="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs text-stone-500 hover:text-error-600 hover:bg-error-50 dark:hover:bg-error-950/30 transition cursor-pointer"
              @click="clearItems"
            >
              <UIcon name="i-lucide-trash-2" class="size-3.5" />
              清空
            </button>
          </div>

          <!-- 项目表格 -->
          <div class="px-4 py-3 min-h-36">
            <div v-if="items.length === 0" class="text-center text-sm text-stone-400 py-8">
              请从左侧选择服务项目
            </div>
            <table v-else class="w-full text-sm">
              <thead>
                <tr class="text-xs text-stone-500 border-b border-stone-200/60 dark:border-stone-800">
                  <th class="text-left font-normal pb-2">项目</th>
                  <th class="text-center font-normal pb-2 w-24">数量</th>
                  <th class="text-right font-normal pb-2">小计</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(it, idx) in items" :key="idx" class="border-b border-stone-100 dark:border-stone-800/60 last:border-0">
                  <td class="py-2.5">
                    <div class="text-sm font-medium">{{ it.name }}</div>
                    <div class="text-xs text-stone-400 tabular-nums mt-0.5">
                      <span :class="lineDiscount(it) > 0 ? 'line-through' : ''">¥{{ it.price }}</span>
                      <span v-if="lineDiscount(it) > 0" class="ml-1 text-primary-600 dark:text-primary-400">折后 ¥{{ ((lineSubtotal(it) - lineDiscount(it)) / it.quantity).toFixed(2) }}</span>
                    </div>
                  </td>
                  <td class="py-2.5">
                    <div class="flex items-center justify-center gap-0.5">
                      <button
                        type="button"
                        class="size-6 inline-flex items-center justify-center rounded bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-stone-700 active:scale-95 transition"
                        @click="changeQty(idx, -1)"
                      ><UIcon name="i-lucide-minus" class="size-3.5" /></button>
                      <span class="tabular-nums w-6 text-center text-sm">{{ it.quantity }}</span>
                      <button
                        type="button"
                        class="size-6 inline-flex items-center justify-center rounded bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-stone-700 active:scale-95 transition"
                        @click="changeQty(idx, 1)"
                      ><UIcon name="i-lucide-plus" class="size-3.5" /></button>
                    </div>
                  </td>
                  <td class="py-2.5 text-right">
                    <div class="tabular-nums text-sm font-medium">¥{{ lineSubtotal(it).toFixed(2) }}</div>
                    <button
                      type="button"
                      class="text-xs text-stone-500 dark:text-stone-400 hover:text-error-600 dark:hover:text-error-400 mt-0.5 transition"
                      @click="items.splice(idx, 1)"
                    >移除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- 金额 -->
          <div class="px-4 py-3 border-t border-stone-200/60 dark:border-stone-800 space-y-2">
            <div class="flex justify-between items-baseline text-sm">
              <span class="text-stone-500">应付总额</span>
              <span class="tabular-nums text-base font-medium text-stone-900 dark:text-stone-100">¥{{ total.toFixed(2) }}</span>
            </div>
            <div v-if="discount > 0" class="flex justify-between items-baseline text-sm">
              <span class="text-stone-500">总优惠</span>
              <span class="tabular-nums text-base font-semibold text-error-600 dark:text-error-400">- ¥{{ discount.toFixed(2) }}</span>
            </div>
            <!-- 实付金额：可点击改价；金额本身大字，旁边明显"改价"按钮提示可操作 -->
            <div class="flex items-baseline justify-between pt-2 mt-1 border-t border-stone-200/60 dark:border-stone-800">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-stone-700 dark:text-stone-300">实付金额</span>
                <UBadge v-if="useManualPrice" label="已改价" color="warning" variant="soft" size="xs" />
              </div>
              <UPopover :ui="{ content: 'p-3 w-64' }">
                <UTooltip :text="useManualPrice ? '已自定义，点击重新调整' : '点击自定义实付金额（折扣 / 抹零）'" :delay-duration="200">
                  <button
                    type="button"
                    class="group inline-flex items-end gap-1.5 cursor-pointer"
                  >
                    <UIcon name="i-lucide-pencil-line" class="size-3.5 mb-1 text-stone-400 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition" />
                    <span class="text-3xl font-semibold tabular-nums text-primary-600 dark:text-primary-400 leading-none group-hover:underline group-hover:decoration-dotted group-hover:underline-offset-4">¥{{ actualPaid.toFixed(2) }}</span>
                  </button>
                </UTooltip>
                <template #content>
                  <div class="space-y-2">
                    <div class="text-xs text-stone-500">自定义实付金额</div>
                    <UInput v-model="manualPrice" type="number" step="0.01" min="0" :max="total.toFixed(2)" :placeholder="`原 ¥${(total - (discount > 0 ? discount : 0)).toFixed(2)}`" size="sm" class="w-full" autofocus />
                    <UInput v-model="manualReason" placeholder="原因（必填）" size="sm" class="w-full" />
                    <p v-if="manualPriceError" class="text-xs text-error-600 dark:text-error-400">{{ manualPriceError }}</p>
                    <div class="flex gap-1.5 pt-1">
                      <UButton size="xs" color="primary" :disabled="!manualPrice || !manualReason || !!manualPriceError" @click="useManualPrice = true">应用</UButton>
                      <UButton v-if="useManualPrice" size="xs" variant="ghost" color="neutral" @click="useManualPrice = false; manualPrice = ''; manualReason = ''">恢复</UButton>
                    </div>
                  </div>
                </template>
              </UPopover>
            </div>
          </div>

          <!-- 支付方案预览 -->
          <div v-if="isMemberCardPay && member && allocationPlan.length > 0" class="px-4 pb-3">
            <div class="rounded-md bg-stone-50 dark:bg-stone-800/40 px-3 py-2">
              <div class="text-xs text-stone-500 mb-1">支付方案</div>
              <div v-for="a in allocationPlan" :key="a.card_id" class="flex justify-between items-baseline text-sm">
                <span class="text-stone-700 dark:text-stone-300">
                  {{ cardName(a.card_id) }}
                  <span class="text-xs text-stone-400">{{ cardRate(a.card_id) }}</span>
                </span>
                <span class="tabular-nums font-medium">扣 ¥{{ a.deduct }}</span>
              </div>
            </div>
          </div>
          <div v-else-if="isMemberCardPay && member && memberCards.length === 0" class="px-4 pb-3 text-xs text-warning-600">
            该会员无可用卡，请改其他支付方式
          </div>
          <div v-else-if="isMemberCardPay && member && allocationPlan.length === 0 && items.length > 0" class="px-4 pb-3">
            <div class="p-3 rounded-lg bg-warning-50/60 dark:bg-warning-950/20 ring-1 ring-warning-200 dark:ring-warning-800 space-y-2.5">
              <div class="text-sm text-warning-700 dark:text-warning-300">
                会员卡余额不足（余额 ¥{{ memberTotalBalance }}，应付 ¥{{ discountedTotal.toFixed(2) }}）
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" color="neutral" @click="pendingMode = ''; selectOtherPm()">选择其他支付方式</UButton>
                <UButton size="xs" variant="soft" color="warning" @click="pendingMode = 'full'">挂账</UButton>
              </div>
              <div v-if="pendingMode" class="pt-2 border-t border-warning-200/60 dark:border-warning-800">
                <URadioGroup
                  v-model="pendingMode"
                  orientation="horizontal"
                  :items="[
                    { value: 'full', label: `全额挂账 ¥${discountedTotal.toFixed(2)}` },
                    { value: 'use_balance', label: `利用余额（扣卡 ¥${memberTotalBalance} + 挂账 ¥${(discountedTotal - parseFloat(memberTotalBalance)).toFixed(2)}）` },
                  ]"
                />
              </div>
            </div>
          </div>

          <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" class="mx-4 mb-3" />

          <!-- 结算按钮 -->
          <div class="px-4 pb-4">
            <UButton
              block
              size="xl"
              :disabled="!canSubmit"
              :loading="submitting"
              @click="submit"
            >结算 <span v-if="actualPaid > 0" class="ml-1 tabular-nums">¥{{ actualPaid.toFixed(2) }}</span></UButton>
          </div>
        </div>
      </aside>
    </div>

    <!-- 底部：今日消费记录 -->
    <section class="mt-8 rounded-lg ring-1 ring-stone-200 dark:ring-stone-800 bg-white dark:bg-stone-900">
      <div class="px-4 py-3 flex items-center justify-between border-b border-stone-200/60 dark:border-stone-800">
        <div class="flex items-center gap-2">
          <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
          <h2 class="text-base font-semibold">今日消费记录</h2>
          <span v-if="todayTx.total > 0" class="text-xs text-stone-500">
            共 {{ todayTx.total }} 笔 · 实收 ¥{{ todayTx.actualSum }}
          </span>
          <span v-if="todayTx.truncated" class="text-xs text-warning-600 dark:text-warning-400">合计仅含最近 200 笔</span>
          <button
            v-if="todayTx.voidedCount > 0"
            type="button"
            class="text-xs text-stone-400 hover:text-stone-600 dark:hover:text-stone-300 transition cursor-pointer"
            @click="todayTx.showVoided = !todayTx.showVoided"
          >
            {{ todayTx.showVoided ? '隐藏' : '显示' }}已撤销 {{ todayTx.voidedCount }} 笔
          </button>
        </div>
      </div>

      <div v-if="todayTx.loading && todayTx.items.length === 0" class="px-6 py-10 text-center text-sm text-stone-400">加载中…</div>
      <div v-else-if="todayTx.items.length === 0" class="px-6 py-10 text-center text-sm text-stone-400">今日暂无消费记录</div>
      <div v-else class="overflow-x-auto">
      <table class="w-full min-w-[720px] text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/30 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-32">姓名</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-40">会员卡</th>
            <th class="text-left px-4 py-2.5 font-medium">服务项目</th>
            <th class="text-center px-4 py-2.5 font-medium whitespace-nowrap w-20">数量</th>
            <th class="text-right px-4 py-2.5 font-medium w-40">金额</th>
            <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap w-20">时间</th>
            <th v-if="canVoid" class="text-center px-4 py-2.5 font-medium whitespace-nowrap w-28">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-100 dark:divide-stone-800">
          <tr
            v-for="t in visibleTx" :key="t.id"
            :class="['even:bg-stone-50/40 dark:even:bg-stone-800/20 hover:bg-stone-100/60! dark:hover:bg-stone-800/30! transition-colors', t.status === 'voided' ? 'opacity-60' : '']"
          >
            <td class="px-4 py-1.5 whitespace-nowrap truncate">
              <span v-if="t.member_name" class="inline-flex items-center gap-1.5 text-stone-900 dark:text-stone-100 font-medium">
                <UIcon name="i-lucide-user-round" class="size-4 text-primary-500 shrink-0" />
                <span class="truncate">{{ t.member_name }}</span>
              </span>
              <span v-else-if="t.customer_name" class="text-stone-500 dark:text-stone-400" :title="t.customer_name + '（非会员）'">{{ t.customer_name }}</span>
              <span v-else class="text-stone-500 dark:text-stone-400">非会员</span>
            </td>
            <td class="px-4 py-1.5 whitespace-nowrap truncate">
              <span v-if="t.card_type_name" class="inline-flex items-center gap-1.5 text-primary-600 dark:text-primary-400">
                <UIcon name="i-lucide-credit-card" class="size-4 shrink-0" />
                <span class="truncate">{{ t.card_type_name }}</span>
              </span>
              <!-- 多卡联合支付无单一 card_id，从余额流水判定，别让它显示成"—" -->
              <span v-else-if="isMultiCard(t)" class="inline-flex items-center gap-1.5 text-primary-600 dark:text-primary-400">
                <UIcon name="i-lucide-wallet-cards" class="size-4 shrink-0" />
                <span>多卡支付</span>
              </span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-1.5 text-stone-600 dark:text-stone-400 break-words">
              {{ t.summary || t.items_summary || '—' }}
              <div v-if="humanTxNotes(t.notes)" class="text-xs text-stone-400 dark:text-stone-500 mt-0.5">{{ humanTxNotes(t.notes) }}</div>
              <div v-if="multiCardText(t)" class="text-xs text-stone-400 dark:text-stone-500 mt-0.5 tabular-nums">{{ multiCardText(t) }}</div>
            </td>
            <td class="px-4 py-1.5 text-center tabular-nums text-stone-600 dark:text-stone-400 whitespace-nowrap">{{ t.item_qty || (isCreditTx(t) || t.kind === 'recharge' ? 1 : '—') }}</td>
            <td class="px-4 py-1.5 text-right align-middle">
              <!-- 实付金额：有卡余额变化时下加虚线，hover 弹出快照（UPopover hover 模式，富结构内容不适合 UTooltip） -->
              <UPopover
                v-if="t.card_snapshots && t.card_snapshots.length > 0"
                mode="hover"
                :open-delay="150"
                :ui="{ content: 'px-3 py-2.5 min-w-64' }"
              >
                <div class="inline-block cursor-help">
                  <div class="flex items-baseline justify-end gap-1.5 leading-tight border-b border-dashed border-stone-300 dark:border-stone-600 pb-0.5">
                    <span v-if="parseFloat(t.discount_amount) > 0 || surcharge(t) > 0" class="text-xs tabular-nums text-stone-400 line-through">¥{{ strikePrice(t) }}</span>
                    <span v-if="isCreditTx(t)" class="text-base font-semibold tabular-nums text-warning-600 dark:text-warning-400">¥{{ t.credit_amount }}</span>
                    <span v-else class="text-base font-semibold tabular-nums text-primary-600 dark:text-primary-400">¥{{ t.actual_paid_amount }}</span>
                  </div>
                  <div class="h-4 text-xs tabular-nums leading-none mt-0.5">
                    <span v-if="isCreditTx(t)" class="text-warning-600 dark:text-warning-400">挂账 · 暂未收款</span>
                    <span v-else-if="parseFloat(t.discount_amount) > 0" :class="discountLabel(t) === '减' ? 'text-stone-500 dark:text-stone-400' : 'text-error-600'">{{ isMultiCard(t) ? '多卡 · ' : '' }}{{ discountLabel(t) }} ¥{{ t.discount_amount }}</span>
                    <span v-else-if="surcharge(t) > 0" class="text-stone-500 dark:text-stone-400">加 ¥{{ surcharge(t).toFixed(2) }}</span>
                    <span v-else-if="t.status === 'voided'" class="text-error-600">已撤销</span>
                  </div>
                </div>
                <template #content>
                  <div>
                    <div class="text-xs text-stone-500 dark:text-stone-400 mb-1.5">余额快照</div>
                    <div v-for="s in t.card_snapshots" :key="s.card_id" class="flex items-baseline justify-between gap-3 py-0.5 text-sm">
                      <span class="truncate text-left text-stone-900 dark:text-stone-100">
                        {{ s.card_type_name }}
                        <span v-if="s.change_type === 'void_restore'" class="text-xs text-warning-600 dark:text-warning-400 ml-1">还原</span>
                        <span v-else-if="s.change_type === 'issue'" class="text-xs text-primary-600 dark:text-primary-400 ml-1">办卡</span>
                      </span>
                      <span class="tabular-nums shrink-0">
                        <span class="text-stone-500 dark:text-stone-400">¥{{ s.balance_before }}</span>
                        <span class="text-stone-400 dark:text-stone-500 mx-1">→</span>
                        <span class="font-semibold text-stone-900 dark:text-stone-100">¥{{ s.balance_after }}</span>
                      </span>
                    </div>
                  </div>
                </template>
              </UPopover>
              <div v-else class="inline-block">
                <div class="flex items-baseline justify-end gap-1.5 leading-tight">
                  <span v-if="parseFloat(t.discount_amount) > 0 || surcharge(t) > 0" class="text-xs tabular-nums text-stone-400 line-through">¥{{ strikePrice(t) }}</span>
                  <span v-if="isCreditTx(t)" class="text-base font-semibold tabular-nums text-warning-600 dark:text-warning-400">¥{{ t.credit_amount }}</span>
                  <span v-else class="text-base font-semibold tabular-nums text-primary-600 dark:text-primary-400">¥{{ t.actual_paid_amount }}</span>
                </div>
                <div class="h-4 text-xs tabular-nums leading-none mt-0.5">
                  <span v-if="isCreditTx(t)" class="text-warning-600 dark:text-warning-400">挂账 · 暂未收款</span>
                  <span v-else-if="parseFloat(t.discount_amount) > 0" :class="discountLabel(t) === '减' ? 'text-stone-500 dark:text-stone-400' : 'text-error-600'">{{ isMultiCard(t) ? '多卡 · ' : '' }}{{ discountLabel(t) }} ¥{{ t.discount_amount }}</span>
                  <span v-else-if="surcharge(t) > 0" class="text-stone-500 dark:text-stone-400">加 ¥{{ surcharge(t).toFixed(2) }}</span>
                  <span v-else-if="t.status === 'voided'" class="text-error-600">已撤销</span>
                </div>
              </div>
            </td>
            <td class="px-4 py-1.5 text-stone-500 text-sm tabular-nums whitespace-nowrap">{{ formatHm(t.transaction_time) }}</td>
            <td v-if="canVoid" class="px-4 py-1.5 text-center whitespace-nowrap">
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
      </div>

      <!-- 分页 footer：加载更多 / 已显示全部 -->
      <div v-if="todayTx.items.length > 0" class="px-4 py-3 border-t border-stone-200/60 dark:border-stone-800 flex items-center justify-center text-xs">
        <button
          v-if="hasMoreTx"
          type="button"
          class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md text-stone-600 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition cursor-pointer"
          @click="loadMoreTx"
        >
          <UIcon name="i-lucide-chevron-down" class="size-3.5" />
          加载更多（剩余 {{ filteredTx.length - todayTx.displayCount }} 条）
        </button>
        <span v-else class="text-stone-400">已显示全部 {{ filteredTx.length }} 条</span>
      </div>
    </section>

    <!-- 撤销弹窗 -->
    <UModal v-model:open="voidOpen" title="撤销交易" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <div class="space-y-3">
          <p class="text-sm text-stone-600">将交易 <strong>{{ voiding?.id.slice(0, 8) }}</strong> 设为已撤销，关联的卡余额 / 挂账会自动恢复。</p>
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
const api = useApi()
const route = useRoute()
const { canVoid, ensureFetched: ensureVoidFetched } = useVoidEnabled()

interface Service { id: string; name: string; price: string; no_discount: boolean; sort_order: number; status: string }
interface Member { id: string; name: string; phone: string | null; status?: string; total_balance?: string; total_pending?: string; card_count?: number }
interface Card { id: string; card_type_name: string; balance: string; final_discount_rate: string; status: string }
interface PaymentMethod { id: string; name: string }
interface Staff { id: string; name: string; position: string }

const services = ref<Service[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const staffList = ref<Staff[]>([])

const member = ref<Member | null>(null)
const walkInName = ref('')
const memberSearch = ref('')
const memberOptions = ref<Member[]>([])
const memberCards = ref<Card[]>([])

const svcSearch = ref('')
const svcDropdownOpen = ref(false)

const items = ref<{ id: string; name: string; price: string; quantity: number; no_discount: boolean }[]>([])
const selectedPm = ref<string>('')
const selectedStaff = ref<string>('')
const manualCardMode = ref(false)
const manualCardId = ref<string>('')
const useManualPrice = ref(false)
const manualPrice = ref('')
const manualReason = ref('')
const useCustomTime = ref(false)
const transactionTime = ref('')
const notes = ref('')
const submitting = ref(false)
const err = ref('')
const toast = useToast()

const activeServices = computed(() => services.value.filter(s => s.status === 'active'))

// 常用服务：按 sort_order 取前 8 个
const quickServices = computed(() =>
  [...activeServices.value].sort((a, b) => a.sort_order - b.sort_order).slice(0, 8),
)

// 常用员工：取前 5 人
const quickStaff = computed(() => staffList.value.slice(0, 5))
const selectedStaffName = computed(() => {
  const s = staffList.value.find(x => x.id === selectedStaff.value)
  return s ? s.name : ''
})
function toggleStaff(id: string) {
  selectedStaff.value = selectedStaff.value === id ? '' : id
}

// 员工搜索（label 右侧紧凑搜索框）
const staffSearch = ref('')
const staffDropdownOpen = ref(false)
function onStaffFocus() { staffDropdownOpen.value = true }
function onStaffBlur() { setTimeout(() => { staffDropdownOpen.value = false }, 150) }
const filteredStaff = computed(() => {
  const q = staffSearch.value.trim().toLowerCase()
  if (!q) return staffList.value
  return staffList.value.filter(s => s.name.toLowerCase().includes(q) || (s.position || '').toLowerCase().includes(q))
})
function pickStaff(id: string) {
  selectedStaff.value = id
  staffSearch.value = ''
  staffDropdownOpen.value = false
}

// 服务下拉开关方法（替代 inline setTimeout，避免事件解析问题）
function onSvcFocus() { svcDropdownOpen.value = true }
function onSvcBlur() { setTimeout(() => { svcDropdownOpen.value = false }, 150) }

// 排序结果缓存：避免每次 filter 都重新 sort，且不能 in-place sort（会触发 reactive 死循环）
const sortedServices = computed(() =>
  [...activeServices.value].sort((a, b) => a.sort_order - b.sort_order),
)

const filteredServices = computed(() => {
  const q = svcSearch.value.trim().toLowerCase()
  if (!q) return sortedServices.value
  return sortedServices.value.filter(s =>
    s.name.toLowerCase().includes(q) || s.price.includes(q)
  )
})

const staffOptions = computed(() =>
  staffList.value.map(s => ({ label: `${s.name}（${s.position}）`, value: s.id })),
)

const isMemberCardPay = computed(() => {
  const pm = paymentMethods.value.find(p => p.id === selectedPm.value)
  return pm?.name === '会员卡'
})

const total = computed(() => items.value.reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0))

// 今日消费记录
interface CardSnapshot {
  card_id: string
  card_type_name: string
  balance_before: string
  balance_after: string
  delta: string
  change_type: string
}
interface TodayTx {
  id: string; kind: 'sale' | 'recharge' | 'credit_settlement'
  status: 'completed' | 'voided'
  member_id: string | null; member_name: string | null
  customer_name: string | null
  card_id: string | null; card_type_name: string | null
  total_amount: string; actual_paid_amount: string; discount_amount: string
  transaction_time: string; summary: string | null
  item_qty: number
  items_summary: string | null
  credit_amount: string; items_total: string
  notes: string | null
  card_snapshots: CardSnapshot[] | null
}

// 挂账登记交易：0 实收 + 关联着一笔挂账
function isCreditTx(t: TodayTx) { return parseFloat(t.credit_amount || '0') > 0 }

// 折数：实付/应收 × 10，整数不带小数（8折），否则一位小数（8.5折）
// 多卡支付：本笔扣了 ≥2 张卡（按余额流水的 consume 行判断）
// 加价额：消费单实付高于应付的部分（现场加项目没改单，老系统迁入数据常见）。
// 挂账实付本来就是 0，充值的差价语义是卖卡折扣，都不算加价
function surcharge(t: TodayTx): number {
  if (t.kind !== 'sale' || isCreditTx(t)) return 0
  // 原应收优先取明细标价合计：迁移把加价单的 total 抬平到实收，原标价只在明细里
  const base = parseFloat(t.items_total) > 0 ? parseFloat(t.items_total) : parseFloat(t.total_amount)
  const d = parseFloat(t.actual_paid_amount) - base
  return d > 0.001 ? d : 0
}

// 划线展示的原价：加价单显示明细标价合计（total 已被抬平，划 total 等于没划）
function strikePrice(t: TodayTx): string {
  return surcharge(t) > 0 && parseFloat(t.items_total) > 0 ? t.items_total : t.total_amount
}

function isMultiCard(t: TodayTx) {
  return (t.card_snapshots || []).filter(s => s.change_type === 'consume').length >= 2
}

// 多卡分卡明细：直接从结构化余额流水组装（老系统靠 notes 文本，这里比它可靠）
function multiCardText(t: TodayTx): string {
  const parts = (t.card_snapshots || []).filter(s => s.change_type === 'consume')
  if (parts.length < 2) return ''
  return '多卡：' + parts
    .map(s => `${s.card_type_name}¥${(parseFloat(s.balance_before) - parseFloat(s.balance_after)).toFixed(2)}`)
    .join(' + ')
}

// 标称折扣率：从卡显示名（"500元储值卡 7折"）提取；多卡取第一张扣款卡
function nominalRate(t: TodayTx): number | null {
  const name = t.card_type_name
    || (t.card_snapshots || []).find(s => s.change_type === 'consume')?.card_type_name
  const m = (name || '').match(/([0-9]+(?:\.[0-9])?)\s*折/)
  return m ? parseFloat(m[1]!) / 10 : null
}

// 折扣描述：反推率与卡的标称折扣吻合才写「N折 省」；对不上（手动调价）
// 只写「减」——拿实付/应付反推出的"6.9折"是没人定过的伪折扣，不展示
function discountLabel(t: TodayTx): string {
  const total = parseFloat(t.total_amount)
  const paid = parseFloat(t.actual_paid_amount)
  if (!(total > 0) || paid >= total) return ''
  const rate = paid / total
  const nominal = nominalRate(t)
  if (nominal != null && Math.abs(rate - nominal) < 0.005) {
    const zhe = Math.round(rate * 100) / 10
    return `${Number.isInteger(zhe) ? zhe : zhe.toFixed(1)}折 省`
  }
  return '减'
}
const PAGE_SIZE = 10 // 默认显示 / 每次加载条数
const todayTx = reactive({
  items: [] as TodayTx[],
  total: 0,
  actualSum: '0.00',
  voidedCount: 0,
  loading: false,
  showVoided: false,
  displayCount: PAGE_SIZE,
  truncated: false, // 今日交易数超过单次拉取上限（200），合计只覆盖最近 200 笔
})
const filteredTx = computed(() =>
  todayTx.showVoided ? todayTx.items : todayTx.items.filter(t => t.status !== 'voided'),
)
const visibleTx = computed(() => filteredTx.value.slice(0, todayTx.displayCount))
const hasMoreTx = computed(() => filteredTx.value.length > todayTx.displayCount)
function loadMoreTx() { todayTx.displayCount += PAGE_SIZE }
function formatHm(s: string) {
  const d = new Date(s)
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}
function kindLabel(k: string) { return (TX_KIND_LABEL as Record<string, string>)[k] ?? k }
function kindColor(k: string) { return (TX_KIND_COLOR as Record<string, 'primary' | 'info' | 'neutral'>)[k] ?? 'neutral' }

// fetchTodayTx 使用 ETag 缓存
//   - 后端 List handler 在 filter 范围内 max(updated_at) 没变时返回 304
//   - 命中 → 跳过 JSON 解析 + 列表更新；只更新 loading 标记
//   - 多店员多 tab 60s 轮询场景下大幅省网络/CPU
let todayTxEtag = ''
async function fetchTodayTx() {
  todayTx.loading = true
  try {
    // 本地日界：formatDateOnly 取本地 YYYY-MM-DD（无偏移字符串按本地时区解析），
    // 再 toISOString 转成对应 UTC 时刻；后端区间是 transaction_time < end_date 左闭右开，
    // end 必须传次日本地 0 点而非 23:59:59（否则漏最后一秒）
    const todayStr = formatDateOnly(new Date())
    const dayStart = new Date(`${todayStr}T00:00:00`)
    const dayEnd = new Date(dayStart)
    dayEnd.setDate(dayEnd.getDate() + 1)
    const q = new URLSearchParams({
      start_date: dayStart.toISOString(),
      end_date: dayEnd.toISOString(),
      limit: '200', // 后端单次上限 200，超出由 truncated 提示兜底
      include_voided: '1',
    })
    const headers: Record<string, string> = {}
    if (todayTxEtag) headers['If-None-Match'] = todayTxEtag
    const res = await api.raw<{ items: TodayTx[]; total: number }>(`/api/transactions?${q}`, {
      headers,
      // 让 304 也走 success 通道而不是抛错
      ignoreResponseError: true,
    })
    if (res.status === 304) {
      // 数据未变，保持当前 items 不变
      return
    }
    if (res.status !== 200 || !res._data) return
    const newEtag = res.headers.get('ETag') || ''
    if (newEtag) todayTxEtag = newEtag
    todayTx.items = res._data.items
    const valid = res._data.items.filter(t => t.status !== 'voided')
    todayTx.total = valid.length
    todayTx.voidedCount = res._data.items.length - valid.length
    todayTx.truncated = (res._data.total ?? 0) > res._data.items.length
    todayTx.actualSum = valid
      .reduce((s, t) => s + parseFloat(t.actual_paid_amount), 0)
      .toFixed(2)
    todayTx.displayCount = PAGE_SIZE // 重新拉数据后回到首页
  } finally { todayTx.loading = false }
}

// 撤销交易
const voidOpen = ref(false)
const voiding = ref<TodayTx | null>(null)
const voidReason = ref('')
const voidLoading = ref(false)
const voidErr = ref('')
function openVoid(t: TodayTx) {
  voiding.value = t
  voidReason.value = ''
  voidErr.value = ''
  voidOpen.value = true
}
async function doVoid() {
  if (!voiding.value) return
  voidLoading.value = true
  voidErr.value = ''
  try {
    await api(`/api/transactions/${voiding.value.id}/void`, { method: 'POST', body: { reason: voidReason.value } })
    voiding.value.status = 'voided'
    todayTx.voidedCount = todayTx.items.filter(t => t.status === 'voided').length
    todayTx.total = todayTx.items.length - todayTx.voidedCount
    voidOpen.value = false
    // 撤销改变实收合计与卡余额：整体重拉今日记录（撤单后 updated_at 变化，ETag 必失效拿到新数据）
    fetchTodayTx()
    if (member.value) {
      const cards = await api<{ items: Card[] }>(`/api/members/${member.value.id}/cards`)
      memberCards.value = cards.items.filter(c => c.status === 'active' && parseFloat(c.balance) > 0)
      const sum = memberCards.value.reduce((s, c) => s + parseFloat(c.balance), 0)
      member.value = { ...member.value, total_balance: sum.toFixed(2) }
    }
  } catch (e: any) {
    voidErr.value = e?.data?.message || '撤销失败（请确认在"设置 → 交易撤销"里开启）'
  } finally { voidLoading.value = false }
}

function autoAllocate(): { card_id: string; deduct: string }[] {
  if (!isMemberCardPay.value || !member.value) return []
  const discountable = items.value.filter(i => !i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
  const noDiscount   = items.value.filter(i =>  i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
  const cards = [...memberCards.value]
    .filter(c => parseFloat(c.balance) > 0)
    .sort((a, b) => parseFloat(a.final_discount_rate) - parseFloat(b.final_discount_rate) || parseFloat(b.balance) - parseFloat(a.balance))
  const result: { card_id: string; deduct: string }[] = []
  let remainNoDiscount = noDiscount
  let remainDiscountable = discountable
  for (const c of cards) {
    if (remainNoDiscount <= 0 && remainDiscountable <= 0) break
    let bal = parseFloat(c.balance)
    const rate = parseFloat(c.final_discount_rate)
    let deduct = 0
    if (remainNoDiscount > 0) {
      const take = Math.min(remainNoDiscount, bal)
      deduct += take; bal -= take; remainNoDiscount -= take
    }
    if (remainDiscountable > 0 && bal > 0) {
      const maxCoverable = bal / rate
      const take = Math.min(remainDiscountable, maxCoverable)
      deduct += take * rate; remainDiscountable -= take
    }
    if (deduct > 0) result.push({ card_id: c.id, deduct: deduct.toFixed(2) })
  }
  if (remainNoDiscount > 0 || remainDiscountable > 0) return []
  return result
}

const allocationPlan = computed<{ card_id: string; deduct: string }[]>(() => {
  if (!isMemberCardPay.value || !member.value || items.value.length === 0) return []
  if (manualCardMode.value && manualCardId.value) {
    const c = memberCards.value.find(x => x.id === manualCardId.value)
    if (!c) return []
    const rate = parseFloat(c.final_discount_rate)
    const discountable = items.value.filter(i => !i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
    const noDiscount   = items.value.filter(i =>  i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
    const need = noDiscount + discountable * rate
    if (parseFloat(c.balance) < need) return []
    return [{ card_id: c.id, deduct: need.toFixed(2) }]
  }
  return autoAllocate()
})

const previewRate = computed(() => {
  if (!isMemberCardPay.value || !member.value || memberCards.value.length === 0) return 1
  if (manualCardMode.value && manualCardId.value) {
    const c = memberCards.value.find(x => x.id === manualCardId.value)
    return c ? parseFloat(c.final_discount_rate) : 1
  }
  const best = [...memberCards.value]
    .filter(c => parseFloat(c.balance) > 0)
    .sort((a, b) => parseFloat(a.final_discount_rate) - parseFloat(b.final_discount_rate))[0]
  return best ? parseFloat(best.final_discount_rate) : 1
})

function lineSubtotal(it: { price: string; quantity: number }) {
  return parseFloat(it.price) * it.quantity
}
function lineDiscount(it: { price: string; quantity: number; no_discount: boolean }) {
  if (it.no_discount || previewRate.value >= 1) return 0
  return lineSubtotal(it) * (1 - previewRate.value)
}

const discount = computed(() => {
  if (useManualPrice.value) {
    const mp = parseFloat(manualPrice.value)
    return isNaN(mp) ? 0 : total.value - mp
  }
  if (!isMemberCardPay.value) return 0
  const totalDeduct = allocationPlan.value.reduce((s, a) => s + parseFloat(a.deduct), 0)
  if (totalDeduct === 0) return 0
  return total.value - totalDeduct
})

const actualPaid = computed(() => {
  if (useManualPrice.value) {
    const mp = parseFloat(manualPrice.value)
    return isNaN(mp) ? 0 : mp
  }
  return total.value - discount.value
})

// 改价校验：后端拒绝 manual_price 超过应收金额（不支持加价），且库内金额是 NUMERIC(10,2)，
// 前端先拦 0 ≤ 值 ≤ 应收、最多 2 位小数
const manualPriceError = computed(() => {
  if (!manualPrice.value) return ''
  if (!/^\d+(\.\d{1,2})?$/.test(manualPrice.value.trim())) return '金额需为不小于 0 的数字，最多 2 位小数'
  if (parseFloat(manualPrice.value) > total.value) return `不能超过应收 ¥${total.value.toFixed(2)}`
  return ''
})

const pendingMode = ref('')

// 挂账决定只对「当时的购物车 / 会员 / 卡分配」有效；任一变化后余额可能重新够扣，
// 残留的挂账标记会把可正常扣卡的单提交成 0 实收全额挂账，故一律重置
watch(
  [items, member, memberCards, selectedPm, manualCardMode, manualCardId],
  () => { pendingMode.value = '' },
  { deep: true },
)

const discountedTotal = computed(() => {
  if (!isMemberCardPay.value || previewRate.value >= 1) return total.value
  const discountable = items.value.filter(i => !i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
  const noDiscount = items.value.filter(i => i.no_discount).reduce((s, it) => s + parseFloat(it.price) * it.quantity, 0)
  return noDiscount + discountable * previewRate.value
})

const memberTotalBalance = computed(() => {
  return memberCards.value
    .filter(c => parseFloat(c.balance) > 0)
    .reduce((s, c) => s + parseFloat(c.balance), 0)
    .toFixed(2)
})

const useBalanceAllocations = computed<{ card_id: string; deduct: string }[]>(() => {
  if (!member.value || memberCards.value.length === 0) return []
  const result: { card_id: string; deduct: string }[] = []
  for (const c of memberCards.value) {
    const bal = parseFloat(c.balance)
    if (bal > 0) result.push({ card_id: c.id, deduct: bal.toFixed(2) })
  }
  return result
})

function selectOtherPm() {
  const cash = paymentMethods.value.find(p => p.name === '现金')
  if (cash) selectedPm.value = cash.id
}

const canSubmit = computed(() => {
  if (items.value.length === 0 || !selectedPm.value || submitting.value) return false
  if (isMemberCardPay.value) {
    if (!member.value) return false
    if (allocationPlan.value.length === 0 && !pendingMode.value) return false
  }
  return true
})

// 实时时钟（每 30 秒刷新；分钟显示精度足够，避免每 5s 触发响应式）
const _now = ref(new Date())
useIntervalFn(() => { _now.value = new Date() }, 30_000)

function pad(n: number) { return String(n).padStart(2, '0') }
const nowLabel = computed(() => {
  const d = _now.value
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
})
function formatLocalTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 时间 picker：UCalendar 的 modelValue 是 @internationalized/date CalendarDate，
// UCalendar 自身能 emit 该类型，我们读 .year/.month/.day 即可，不需直接 import 类型
const customDate = ref<any>(null)
const customHour = ref<string>(pad(_now.value.getHours()))
const customMinute = ref<string>(pad(_now.value.getMinutes()))
const hourOptions = Array.from({ length: 24 }, (_, i) => ({ label: pad(i), value: pad(i) }))
const minuteOptions = Array.from({ length: 60 }, (_, i) => ({ label: pad(i), value: pad(i) }))

const displayTimeLabel = computed(() => {
  if (useCustomTime.value && transactionTime.value) return formatLocalTime(transactionTime.value)
  return nowLabel.value
})

// popover 开关：监听打开时把时分 reset 到当前（避免显示陈旧时间）
const timePopoverOpen = ref(false)
watch(timePopoverOpen, (open) => {
  if (open && !useCustomTime.value) {
    const n = new Date()
    customHour.value = pad(n.getHours())
    customMinute.value = pad(n.getMinutes())
  }
})

function applyCustomTime() {
  const d = customDate.value
  let year: number, month: number, day: number
  // 用户没在 calendar 里点选 → fallback 为今天
  if (d && typeof d.year === 'number') {
    year = d.year; month = d.month; day = d.day
  } else {
    const n = new Date()
    year = n.getFullYear(); month = n.getMonth() + 1; day = n.getDate()
  }
  const iso = `${year}-${pad(month)}-${pad(day)}T${customHour.value}:${customMinute.value}:00`
  transactionTime.value = iso
  useCustomTime.value = true
  timePopoverOpen.value = false
}
function resetTimeToNow() {
  useCustomTime.value = false
  transactionTime.value = ''
  customDate.value = null
  const n = new Date()
  customHour.value = pad(n.getHours())
  customMinute.value = pad(n.getMinutes())
  timePopoverOpen.value = false
}

function memberStatusLabel(s: string) {
  return ({ inactive: '停用', frozen: '冻结', deleted: '已删' } as Record<string, string>)[s] || s
}
// 头像字符：取姓名最后一个字（中文姓名末字 = 名字，更易区分同姓客户）
function avatarChar(name: string) {
  if (!name) return '?'
  return [...name.trim()].slice(-1)[0] || '?'
}
// 手机号脱敏：158****1234（POS 屏隐私保护）
function formatPhone(p: string | null) {
  if (!p) return '—'
  if (p.length === 11) return `${p.slice(0, 3)}****${p.slice(7)}`
  return p
}
function cardName(id: string) { return memberCards.value.find(x => x.id === id)?.card_type_name || '卡' }
function cardRate(id: string) {
  const c = memberCards.value.find(x => x.id === id)
  return c ? displayRate(c.final_discount_rate) : ''
}
function displayRate(r: string) { const n = parseFloat(r); return (Math.round(n * 100) / 10).toFixed(1) + '折' }

function clearMember() {
  member.value = null
  memberCards.value = []
  manualCardId.value = ''
  manualCardMode.value = false
  if (isMemberCardPay.value) {
    const cash = paymentMethods.value.find(p => p.name === '现金')
    if (cash) selectedPm.value = cash.id
  }
}

function selectPm(id: string) {
  const pm = paymentMethods.value.find(p => p.id === id)
  if (!pm) return
  if (pm.name === '会员卡' && (!member.value || memberCards.value.length === 0)) return
  selectedPm.value = id
}

async function fetchBase() {
  const [svc, pm, sf] = await Promise.all([
    api<{ items: Service[] }>('/api/services?status=active'),
    api<{ items: PaymentMethod[] }>('/api/payment-methods?active=1'),
    api<{ items: Staff[] }>('/api/staff?status=active'),
  ])
  services.value = svc.items
  paymentMethods.value = pm.items
  staffList.value = sf.items
  const cash = pm.items.find(p => p.name === '现金')
  if (cash) selectedPm.value = cash.id
  if (route.query.member) {
    const m = await api<Member>(`/api/members/${route.query.member}`)
    pickMember(m)
  }
}

// 会员搜索：useDebounceFn 150ms（POS 响应敏捷感）。
// 序号 + 当前关键词双守卫：乱序返回 / 清空后才返回的旧响应一律丢弃。
// 不用 AbortController：ofetch 中断时抛 FetchError（AbortError 只挂在 cause 上），靠 catch 判错不可靠
let searchSeq = 0
const debouncedSearch = useDebounceFn(async () => {
  const seq = ++searchSeq
  const q = memberSearch.value.trim()
  if (!q) { memberOptions.value = []; return }
  try {
    const d = await api<{ items: Member[] }>(`/api/members?search=${encodeURIComponent(q)}&limit=8`)
    if (seq !== searchSeq || memberSearch.value.trim() !== q) return
    memberOptions.value = d.items || []
  } catch {
    if (seq === searchSeq) memberOptions.value = []
  }
}, 150)

function handleEnter() {
  if (memberOptions.value.length > 0) {
    pickMember(memberOptions.value[0]!)
    return
  }
  const v = memberSearch.value.trim()
  if (v) {
    walkInName.value = v
    memberSearch.value = ''
  }
}

async function pickMember(m: Member) {
  member.value = m
  walkInName.value = ''
  memberOptions.value = []
  memberSearch.value = ''
  const cards = await api<{ items: Card[] }>(`/api/members/${m.id}/cards`)
  memberCards.value = cards.items.filter(c => c.status === 'active' && parseFloat(c.balance) > 0)
  if (memberCards.value.length > 0) {
    const mc = paymentMethods.value.find(p => p.name === '会员卡')
    if (mc) selectedPm.value = mc.id
  }
}

function addItem(s: Service) {
  const exist = items.value.find(i => i.id === s.id)
  if (exist) exist.quantity++
  else items.value.push({ id: s.id, name: s.name, price: s.price, quantity: 1, no_discount: s.no_discount })
  svcSearch.value = ''
  svcDropdownOpen.value = false
}
function addFirstMatch() {
  if (filteredServices.value.length > 0) addItem(filteredServices.value[0]!)
}
function changeQty(idx: number, delta: number) {
  const it = items.value[idx]!
  const next = it.quantity + delta
  if (next <= 0) items.value.splice(idx, 1)
  else it.quantity = next
}
function clearItems() {
  items.value = []
  useManualPrice.value = false
  manualPrice.value = ''
  manualReason.value = ''
}

async function submit() {
  err.value = ''
  if (!canSubmit.value) return

  const body: any = {
    payment_method_id: selectedPm.value,
    items: items.value.map(i => ({ service_id: i.id, quantity: i.quantity })),
  }
  if (member.value) body.member_id = member.value.id
  else if (walkInName.value) body.customer_name = walkInName.value

  if (selectedStaff.value) body.staff_id = selectedStaff.value

  if (useManualPrice.value) {
    if (!manualReason.value) { err.value = '手动改价需要填原因'; return }
    // 应用改价后购物车可能又变了（应收变小），提交前按当前应收重新校验
    if (manualPriceError.value) { err.value = `改价无效：${manualPriceError.value}`; return }
    body.manual_price = manualPrice.value
  }

  const noteParts: string[] = []
  if (useManualPrice.value && manualReason.value) noteParts.push(`[价格调整] ${manualReason.value}`)
  if (notes.value.trim()) noteParts.push(notes.value.trim())
  if (noteParts.length) body.notes = noteParts.join(' · ')

  if (useCustomTime.value && transactionTime.value) {
    body.transaction_time = new Date(transactionTime.value).toISOString()
  }

  if (isMemberCardPay.value) {
    // 双保险：余额已够扣（allocationPlan 非空）时绝不挂账，兜住 watch 之外的任何状态残留
    if (pendingMode.value && allocationPlan.value.length > 0) pendingMode.value = ''
    if (pendingMode.value === 'full') {
      body.pending_mode = 'full'
      body.manual_price = '0'
    } else if (pendingMode.value === 'use_balance') {
      body.pending_mode = 'use_balance'
      body.card_allocations = useBalanceAllocations.value
    } else {
      body.card_allocations = allocationPlan.value
    }
  }

  submitting.value = true
  const settled = actualPaid.value
  const isPending = !!pendingMode.value
  const pendingAmt = isPending
    ? (pendingMode.value === 'full' ? discountedTotal.value : discountedTotal.value - parseFloat(memberTotalBalance.value))
    : 0
  try {
    await api('/api/transactions', { method: 'POST', body })
    if (isPending) {
      toast.add({
        title: '已挂账',
        description: `挂账 ¥${pendingAmt.toFixed(2)}${pendingMode.value === 'use_balance' ? `，扣卡 ¥${memberTotalBalance.value}` : ''}`,
        color: 'warning',
        icon: 'i-lucide-clock',
      })
    } else {
      toast.add({
        title: '已记账',
        description: `本单实收 ¥${settled.toFixed(2)}`,
        color: 'success',
        icon: 'i-lucide-check-circle',
      })
    }
    items.value = []
    notes.value = ''
    useManualPrice.value = false
    manualPrice.value = ''
    manualReason.value = ''
    useCustomTime.value = false
    transactionTime.value = ''
    pendingMode.value = ''
    // 结账成功 → 换一句祝福语（仪式感 + 正反馈）+ 刷新今日记录
    useGreeting().refresh()
    fetchTodayTx()
    if (member.value) {
      const cards = await api<{ items: Card[] }>(`/api/members/${member.value.id}/cards`)
      memberCards.value = cards.items.filter(c => c.status === 'active' && parseFloat(c.balance) > 0)
      const sum = memberCards.value.reduce((s, c) => s + parseFloat(c.balance), 0)
      member.value = { ...member.value, total_balance: sum.toFixed(2) }
    }
  } catch (e: any) {
    err.value = e?.data?.message || '结账失败'
  } finally { submitting.value = false }
}

// 静默轮询：每 60 秒拉一次今日记录，自动同步其他店员的开单（多店员场景）
onMounted(() => {
  fetchBase()
  fetchTodayTx()
  ensureVoidFetched()
})
useIntervalFn(fetchTodayTx, 60_000)
</script>
