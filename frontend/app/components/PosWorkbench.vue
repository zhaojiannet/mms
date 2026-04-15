<template>
  <div>
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-6">
      <!-- 左：表单（占 3 列），各项之间用极浅分割线 -->
      <section class="lg:col-span-3 divide-y divide-stone-200/50 dark:divide-stone-800/40 [&>div]:py-5 [&>div:first-child]:pt-0 [&>div:last-child]:pb-0">

        <!-- 会员 -->
        <div>
          <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />会员</label>
          <div class="mt-1.5">

            <!-- 已选会员卡片：浅 primary 背景 + 头像 + 完整信息 -->
            <div v-if="member" class="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-primary-50/60 dark:bg-primary-950/30 ring-1 ring-primary-500/30">
              <div class="size-10 shrink-0 rounded-full bg-primary-500 text-white flex items-center justify-center text-base font-semibold">
                {{ avatarChar(member.name) }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-base font-semibold truncate">{{ member.name }}</span>
                  <UBadge v-if="member.status && member.status !== 'active'" :label="memberStatusLabel(member.status)" color="warning" variant="soft" size="xs" />
                  <span v-if="member.phone" class="text-sm text-stone-500 tabular-nums">{{ formatPhone(member.phone) }}</span>
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
                    <span class="tabular-nums font-medium text-error-600">¥{{ member.total_pending }}</span>
                  </span>
                </div>
              </div>
              <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-x" @click="clearMember" />
            </div>

            <!-- 散客 -->
            <div v-else-if="walkInName" class="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-stone-50 dark:bg-stone-800/60 ring-1 ring-stone-200 dark:ring-stone-700">
              <div class="size-10 shrink-0 rounded-full bg-stone-200 dark:bg-stone-700 text-stone-500 flex items-center justify-center">
                <UIcon name="i-lucide-user-round" class="size-5" />
              </div>
              <div class="flex-1 flex items-center gap-2">
                <span class="text-base font-semibold">{{ walkInName }}</span>
                <UBadge label="散客" color="neutral" variant="soft" size="xs" />
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
                @update:model-value="debouncedSearch"
                @keydown.enter="handleEnter"
              />
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
                    <div class="text-xs text-stone-400 tabular-nums">{{ formatPhone(m.phone) }}</div>
                  </div>
                  <div class="shrink-0 text-right">
                    <div v-if="parseFloat(m.total_balance ?? '0') > 0" class="text-sm font-semibold tabular-nums text-primary-600 dark:text-primary-400">
                      <span class="text-xs text-stone-400 mr-1.5 font-normal">余额</span>¥{{ m.total_balance }}
                    </div>
                    <div v-else class="text-xs text-stone-400">未办卡</div>
                    <div v-if="parseFloat(m.total_pending ?? '0') > 0" class="text-xs text-error-600 tabular-nums">
                      <span class="text-stone-400 font-normal mr-1.5">挂账</span>¥{{ m.total_pending }}
                    </div>
                  </div>
                </button>
              </div>
              <div
                v-else-if="memberSearch.trim().length > 0"
                class="absolute left-0 right-0 top-full mt-1 z-20 px-3 py-2.5 rounded-lg bg-white dark:bg-stone-900 ring-1 ring-stone-200 dark:ring-stone-700 shadow-lg text-sm text-stone-500"
              >
                无匹配，按 <kbd class="px-1.5 py-0.5 rounded bg-stone-100 dark:bg-stone-800 text-stone-600 dark:text-stone-300 text-xs">Enter</kbd> 以
                <span class="font-medium text-stone-700 dark:text-stone-200">「{{ memberSearch }}」</span> 散客开单
              </div>
            </div>
          </div>
        </div>

        <!-- 服务：label + 右侧紧凑搜索 + 8 个常用卡片 -->
        <div>
          <div class="flex items-center justify-between gap-3 mb-2">
            <div class="flex items-center gap-2">
              <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />服务</label>
              <span v-if="items.length > 0" class="text-xs text-stone-500">已选 {{ items.length }} 项</span>
            </div>
            <div class="relative">
              <UInput
                v-model="svcSearch"
                icon="i-lucide-search"
                placeholder="找其他项目"
                size="sm"
                class="w-36"
                :ui="{ base: 'rounded-full' }"
                @keydown.enter="addFirstMatch"
                @focus="onSvcFocus"
                @click="onSvcFocus"
                @blur="onSvcBlur"
              />
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
              <div class="text-sm font-medium truncate pr-6">{{ s.name }}</div>
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
              <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />服务员工</label>
              <span v-if="selectedStaff" class="text-xs text-stone-500">已选 {{ selectedStaffName }}</span>
            </div>
            <div v-if="staffList.length > 0" class="relative">
              <UInput
                v-model="staffSearch"
                icon="i-lucide-search"
                placeholder="找其他员工"
                size="sm"
                class="w-36"
                :ui="{ base: 'rounded-full' }"
                @focus="onStaffFocus"
                @click="onStaffFocus"
                @blur="onStaffBlur"
              />
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
              <div class="text-sm font-medium truncate pr-6">{{ s.name }}</div>
              <div class="text-xs text-stone-500 truncate">{{ s.position }}</div>
            </button>
          </div>
        </div>

        <!-- 支付方式（上下都不显示分割线，间距收紧） -->
        <div class="!border-y-0 !py-3">
          <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />支付方式</label>
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
        <div class="grid grid-cols-1 sm:[grid-template-columns:9fr_11fr] gap-4 !border-y-0 !pt-3">
          <div>
            <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />交易时间</label>
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
            <label class="inline-flex items-center gap-2 text-base font-medium text-stone-700 dark:text-stone-300"><span class="inline-block w-1 h-4 rounded-full bg-primary-500" />备注</label>
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
          <div class="px-4 py-3 min-h-[140px]">
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
                      class="text-xs text-stone-300 hover:text-error-500 mt-0.5 transition"
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
              <span class="tabular-nums text-base font-medium text-stone-700 dark:text-stone-300">¥{{ total.toFixed(2) }}</span>
            </div>
            <div v-if="discount > 0" class="flex justify-between items-baseline text-sm">
              <span class="text-stone-500">总优惠</span>
              <span class="tabular-nums text-base font-semibold text-error-600 dark:text-error-400">- ¥{{ discount.toFixed(2) }}</span>
            </div>
            <div v-else-if="discount < 0" class="flex justify-between items-baseline text-sm">
              <span class="text-stone-500">加价</span>
              <span class="tabular-nums text-base font-semibold text-warning-600">+ ¥{{ Math.abs(discount).toFixed(2) }}</span>
            </div>
            <!-- 实付金额：可点击改价；金额本身大字，旁边明显"改价"按钮提示可操作 -->
            <div class="flex items-baseline justify-between pt-2 mt-1 border-t border-stone-200/60 dark:border-stone-800">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-stone-700 dark:text-stone-300">实付金额</span>
                <UBadge v-if="useManualPrice" label="已改价" color="warning" variant="soft" size="xs" />
              </div>
              <div class="relative inline-flex group">
                <UPopover :ui="{ content: 'p-3 w-64' }">
                  <button
                    type="button"
                    class="inline-flex items-end gap-1.5 cursor-pointer"
                  >
                    <UIcon name="i-lucide-pencil-line" class="size-3.5 mb-1 text-stone-400 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition" />
                    <span class="text-3xl font-bold tabular-nums text-primary-600 dark:text-primary-400 leading-none group-hover:underline group-hover:decoration-dotted group-hover:underline-offset-4">¥{{ actualPaid.toFixed(2) }}</span>
                  </button>
                <template #content>
                  <div class="space-y-2">
                    <div class="text-xs text-stone-500">自定义实付金额</div>
                    <UInput v-model="manualPrice" type="number" step="0.01" :placeholder="`原 ¥${(total - (discount > 0 ? discount : 0)).toFixed(2)}`" size="sm" class="w-full" autofocus />
                    <UInput v-model="manualReason" placeholder="原因（必填）" size="sm" class="w-full" />
                    <div class="flex gap-1.5 pt-1">
                      <UButton size="xs" color="primary" :disabled="!manualPrice || !manualReason" @click="useManualPrice = true">应用</UButton>
                      <UButton v-if="useManualPrice" size="xs" variant="ghost" color="neutral" @click="useManualPrice = false; manualPrice = ''; manualReason = ''">恢复</UButton>
                    </div>
                  </div>
                </template>
              </UPopover>
                <span class="pointer-events-none absolute right-0 top-full mt-1 px-2.5 py-1 rounded-md bg-stone-900/90 dark:bg-stone-100/90 text-white dark:text-stone-900 text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity duration-150 shadow-md z-20">
                  {{ useManualPrice ? '已自定义，点击重新调整' : '点击自定义实付金额（折扣 / 抹零 / 加价）' }}
                </span>
              </div>
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
          <div v-else-if="isMemberCardPay && member && allocationPlan.length === 0 && items.length > 0" class="px-4 pb-3 text-xs text-warning-600">
            余额不足，请改支付方式或减少项目
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
      <table v-else class="w-full text-base">
        <thead class="bg-stone-50/60 dark:bg-stone-950/30 text-stone-500 text-xs">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium">姓名</th>
            <th class="text-left px-4 py-2.5 font-medium">会员卡</th>
            <th class="text-left px-4 py-2.5 font-medium">服务项目</th>
            <th class="text-center px-4 py-2.5 font-medium w-14">数量</th>
            <th class="text-right px-4 py-2.5 font-medium">金额</th>
            <th class="text-left px-4 py-2.5 font-medium">时间</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-100 dark:divide-stone-800">
          <tr
            v-for="t in visibleTx" :key="t.id"
            :class="['even:bg-stone-50/40 dark:even:bg-stone-800/20 hover:!bg-stone-100/60 dark:hover:!bg-stone-800/30 transition-colors', t.status === 'voided' ? 'opacity-50' : '']"
          >
            <td class="px-4 py-1.5">
              <span v-if="t.member_name" class="inline-flex items-center gap-1.5 text-stone-700 dark:text-stone-300">
                <UIcon name="i-lucide-user-round" class="size-4 text-primary-500" />
                {{ t.member_name }}
              </span>
              <span v-else-if="t.customer_name" class="text-stone-700 dark:text-stone-300">{{ t.customer_name }} <span class="text-xs text-stone-400 ml-1">散客</span></span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-1.5">
              <span v-if="t.card_type_name" class="inline-flex items-center gap-1.5 text-primary-600 dark:text-primary-400">
                <UIcon name="i-lucide-credit-card" class="size-4" />
                {{ t.card_type_name }}
              </span>
              <span v-else class="text-stone-400">—</span>
            </td>
            <td class="px-4 py-1.5 max-w-xs truncate text-stone-700 dark:text-stone-300">{{ t.summary || '—' }}</td>
            <td class="px-4 py-1.5 text-center tabular-nums text-stone-700 dark:text-stone-300">{{ t.item_qty || '—' }}</td>
            <td class="px-4 py-1.5 text-right align-middle">
              <!-- 实付金额：有卡余额变化时下加虚线，hover 展示快照 -->
              <div class="relative inline-block group">
                <div
                  class="flex items-baseline justify-end gap-1.5 leading-tight"
                  :class="t.card_snapshots && t.card_snapshots.length > 0 ? 'cursor-help border-b border-dashed border-stone-300 dark:border-stone-600 pb-0.5' : ''"
                >
                  <span v-if="parseFloat(t.discount_amount) > 0" class="text-sm tabular-nums text-stone-400 line-through">¥{{ t.total_amount }}</span>
                  <span class="text-lg font-semibold tabular-nums text-primary-600 dark:text-primary-400">¥{{ t.actual_paid_amount }}</span>
                </div>
                <div class="h-4 text-xs tabular-nums leading-none mt-0.5">
                  <span v-if="parseFloat(t.discount_amount) > 0" class="text-error-600">省 ¥{{ t.discount_amount }}</span>
                  <span v-else-if="t.status === 'voided'" class="text-error-600">已撤销</span>
                </div>

                <!-- 余额快照 tooltip -->
                <div
                  v-if="t.card_snapshots && t.card_snapshots.length > 0"
                  class="pointer-events-none absolute right-0 top-full mt-1 z-30 min-w-[16rem] px-3 py-2.5 rounded-md bg-stone-900/95 dark:bg-stone-100/95 text-white dark:text-stone-900 shadow-lg opacity-0 group-hover:opacity-100 transition-opacity duration-150"
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
            <td class="px-4 py-1.5 text-stone-500 text-sm tabular-nums">{{ formatHm(t.transaction_time) }}</td>
          </tr>
        </tbody>
      </table>

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
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const route = useRoute()

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
  card_snapshots: CardSnapshot[] | null
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
function kindLabel(k: string) { return ({ sale: '消费', recharge: '办卡', credit_settlement: '清账' } as Record<string,string>)[k] ?? k }
function kindColor(k: string): any { return ({ sale: 'primary', recharge: 'info', credit_settlement: 'neutral' } as Record<string,string>)[k] ?? 'neutral' }

async function fetchTodayTx() {
  todayTx.loading = true
  try {
    const todayStr = new Date().toISOString().slice(0, 10)
    const q = new URLSearchParams({
      start_date: `${todayStr}T00:00:00Z`,
      end_date:   `${todayStr}T23:59:59Z`,
      limit: '50',
      include_voided: '1',
    })
    const data = await api<{ items: TodayTx[]; total: number }>(`/api/transactions?${q}`)
    todayTx.items = data.items
    const valid = data.items.filter(t => t.status !== 'voided')
    todayTx.total = valid.length
    todayTx.voidedCount = data.items.length - valid.length
    todayTx.actualSum = valid
      .reduce((s, t) => s + parseFloat(t.actual_paid_amount), 0)
      .toFixed(2)
    todayTx.displayCount = PAGE_SIZE // 重新拉数据后回到首页
  } finally { todayTx.loading = false }
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

const canSubmit = computed(() => {
  if (items.value.length === 0 || !selectedPm.value || submitting.value) return false
  if (isMemberCardPay.value) {
    if (!member.value || allocationPlan.value.length === 0) return false
  }
  return true
})

// 实时时钟（每 5 秒刷新；分钟变化必能被显示捕捉到）
const _now = ref(new Date())
let _nowTimer: any
onMounted(() => { _nowTimer = setInterval(() => { _now.value = new Date() }, 5_000) })
onBeforeUnmount(() => clearInterval(_nowTimer))

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

// 会员搜索：debounce 150ms（POS 响应敏捷感）+ 取消旧请求避免乱序
let searchTimer: any
let searchAbort: AbortController | null = null
function debouncedSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    const q = memberSearch.value.trim()
    if (!q) { memberOptions.value = []; return }
    // 取消上一次未完成的请求
    if (searchAbort) searchAbort.abort()
    searchAbort = new AbortController()
    try {
      const d = await api<{ items: Member[] }>(`/api/members?search=${encodeURIComponent(q)}&limit=8`, { signal: searchAbort.signal })
      memberOptions.value = d.items || []
    } catch (e: any) {
      if (e?.name !== 'AbortError') memberOptions.value = []
    }
  }, 150)
}

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
    body.card_allocations = allocationPlan.value
  }

  submitting.value = true
  const settled = actualPaid.value
  try {
    await api('/api/transactions', { method: 'POST', body })
    toast.add({
      title: '已记账',
      description: `本单实收 ¥${settled.toFixed(2)}`,
      color: 'success',
      icon: 'i-lucide-check-circle',
    })
    items.value = []
    notes.value = ''
    useManualPrice.value = false
    manualPrice.value = ''
    manualReason.value = ''
    useCustomTime.value = false
    transactionTime.value = ''
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
let _txPollTimer: any
onMounted(() => {
  fetchBase()
  fetchTodayTx()
  _txPollTimer = setInterval(fetchTodayTx, 60_000)
})
onBeforeUnmount(() => {
  clearInterval(_txPollTimer)
  clearTimeout(searchTimer)
  searchAbort?.abort()
})
</script>
