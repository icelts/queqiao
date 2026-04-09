<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loadingInitial" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)]">
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ t('referral.shareTitle') }}
                  </h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('referral.shareDescription') }}
                  </p>
                </div>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="refreshing" @click="refreshAll">
                  {{ refreshing ? t('common.loading') : t('common.refresh') }}
                </button>
              </div>
            </div>

            <div class="space-y-5 p-6">
              <div class="rounded-2xl border border-primary-100 bg-primary-50/70 p-5 dark:border-primary-500/20 dark:bg-primary-500/10">
                <div class="text-xs font-semibold uppercase tracking-[0.24em] text-primary-600 dark:text-primary-300">
                  {{ t('referral.codeLabel') }}
                </div>
                <div class="mt-3 flex flex-wrap items-center gap-3">
                  <code class="rounded-xl bg-white px-4 py-2 font-mono text-xl font-semibold tracking-[0.2em] text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white">
                    {{ summary?.referral_code || '-' }}
                  </code>
                  <button type="button" class="btn btn-primary btn-sm" :disabled="!summary?.referral_code" @click="copyCode">
                    {{ t('referral.copyCode') }}
                  </button>
                </div>
              </div>

              <div>
                <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('referral.linkLabel') }}
                </div>
                <div class="flex flex-col gap-3 lg:flex-row">
                  <input :value="inviteLink" type="text" readonly class="input flex-1 font-mono text-sm" />
                  <button type="button" class="btn btn-secondary whitespace-nowrap" :disabled="!inviteLink" @click="copyLink">
                    {{ t('referral.copyLink') }}
                  </button>
                </div>
                <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('referral.linkHint') }}
                </p>
              </div>
            </div>
          </div>

          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('referral.withdraw.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('referral.withdraw.description') }}
              </p>
            </div>

            <div class="space-y-4 p-6">
              <div v-if="!summary?.withdraw_enabled" class="rounded-xl border border-dashed border-gray-200 px-4 py-5 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                {{ t('referral.withdraw.disabled') }}
              </div>

              <template v-else>
                <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <div class="rounded-xl border border-emerald-100 bg-emerald-50/70 p-4 dark:border-emerald-900/40 dark:bg-emerald-900/10">
                    <div class="text-xs text-emerald-700 dark:text-emerald-300">{{ t('referral.summary.availableCommission') }}</div>
                    <div class="mt-2 text-xl font-semibold text-emerald-700 dark:text-emerald-300">
                      {{ formatMoney(summary?.available_commission ?? 0) }}
                    </div>
                  </div>
                  <div class="rounded-xl border border-amber-100 bg-amber-50/70 p-4 dark:border-amber-900/40 dark:bg-amber-900/10">
                    <div class="text-xs text-amber-700 dark:text-amber-300">{{ t('referral.summary.pendingWithdrawal') }}</div>
                    <div class="mt-2 text-xl font-semibold text-amber-700 dark:text-amber-300">
                      {{ formatMoney(summary?.pending_withdrawal_amount ?? 0) }}
                    </div>
                  </div>
                  <div class="rounded-xl border border-blue-100 bg-blue-50/70 p-4 dark:border-blue-900/40 dark:bg-blue-900/10">
                    <div class="text-xs text-blue-700 dark:text-blue-300">{{ t('referral.summary.approvedWithdrawal') }}</div>
                    <div class="mt-2 text-xl font-semibold text-blue-700 dark:text-blue-300">
                      {{ formatMoney(summary?.approved_withdrawal_amount ?? 0) }}
                    </div>
                  </div>
                  <div class="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-900/20">
                    <div class="text-xs text-slate-700 dark:text-slate-300">{{ textOr('referral.summary.frozenCommission', '冻结中佣金') }}</div>
                    <div class="mt-2 text-xl font-semibold text-slate-700 dark:text-slate-300">
                      {{ formatMoney(summary?.frozen_commission ?? 0) }}
                    </div>
                  </div>
                </div>

                <div class="rounded-xl bg-gray-50 px-4 py-4 text-sm text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                  <div>{{ t('referral.withdraw.rules', { amount: formatMoney(summary?.withdraw_min_amount ?? 0), count: summary?.withdraw_min_invitees ?? 0 }) }}</div>
                  <div class="mt-1">
                    {{ t('referral.withdraw.effectiveInvitees', { count: summary?.effective_invitee_count ?? 0 }) }}
                  </div>
                  <div class="mt-1">
                    {{ textOr('referral.withdraw.frozenHint', '仍处于一个月冻结期的佣金：{amount}').replace('{amount}', formatMoney(summary?.frozen_commission ?? 0)) }}
                  </div>
                  <div v-if="summary?.next_unlock_at" class="mt-1">
                    {{ textOr('referral.withdraw.nextUnlockAt', '最近一笔解冻时间：{time}').replace('{time}', formatDateTime(summary.next_unlock_at)) }}
                  </div>
                  <div v-if="!summary?.can_withdraw" class="mt-2 text-rose-600 dark:text-rose-400">
                    {{ t('referral.withdraw.notEligible') }}
                  </div>
                </div>

                <form class="space-y-4" @submit.prevent="submitWithdrawal">
                  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('referral.withdraw.amount') }}
                      </label>
                      <input
                        v-model.number="withdrawalForm.amount"
                        type="number"
                        min="0"
                        step="0.01"
                        class="input"
                        :placeholder="String(summary?.withdraw_min_amount || 0)"
                      />
                    </div>

                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('referral.withdraw.method') }}
                      </label>
                      <select v-model="withdrawalForm.payment_method" class="input">
                        <option v-for="item in paymentMethodOptions" :key="item.value" :value="item.value">
                          {{ item.label }}
                        </option>
                      </select>
                    </div>

                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('referral.withdraw.accountName') }}
                      </label>
                      <input
                        v-model="withdrawalForm.account_name"
                        type="text"
                        class="input"
                        :placeholder="t('referral.withdraw.accountNamePlaceholder')"
                      />
                    </div>

                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('referral.withdraw.account') }}
                      </label>
                      <input
                        v-model="withdrawalForm.account_identifier"
                        type="text"
                        class="input"
                        :placeholder="t('referral.withdraw.accountPlaceholder')"
                      />
                    </div>
                  </div>

                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t('referral.withdraw.notes') }}
                    </label>
                    <textarea
                      v-model="withdrawalForm.notes"
                      rows="3"
                      class="input"
                      :placeholder="t('referral.withdraw.notesPlaceholder')"
                    />
                  </div>

                  <div class="flex justify-end">
                    <button type="submit" class="btn btn-primary" :disabled="submittingWithdrawal || !summary?.can_withdraw">
                      {{ submittingWithdrawal ? t('common.saving') : t('referral.withdraw.submit') }}
                    </button>
                  </div>
                </form>
              </template>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="card p-5">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('referral.summary.invitees') }}
            </div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary?.invitee_count ?? 0 }}
            </div>
          </div>

          <div class="card p-5">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('referral.summary.totalCommission') }}
            </div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatMoney(summary?.total_commission ?? 0) }}
            </div>
          </div>

          <div class="card p-5">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('referral.summary.firstCommissionCount') }}
            </div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ summary?.first_commission_count ?? 0 }}
            </div>
          </div>

          <div class="card p-5">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t('referral.summary.totalRecharge') }}
            </div>
            <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatMoney(summary?.total_recharge_amount ?? 0) }}
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 2xl:grid-cols-2">
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('referral.invitees.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('referral.invitees.description') }}
              </p>
            </div>

            <DataTable :columns="inviteeColumns" :data="invitees" :loading="loadingInvitees">
              <template #cell-user="{ row }">
                <div class="min-w-0">
                  <div class="font-medium text-gray-900 dark:text-white">
                    {{ row.username || t('common.notAvailable') }}
                  </div>
                  <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ row.email }}
                  </div>
                </div>
              </template>

              <template #cell-registered_at="{ value }">
                <span class="text-sm text-gray-600 dark:text-gray-400">
                  {{ formatDateTime(value) }}
                </span>
              </template>

              <template #cell-first_paid="{ row }">
                <div v-if="row.first_paid_at" class="text-sm text-gray-600 dark:text-gray-400">
                  <div>{{ formatMoney(row.first_paid_amount) }}</div>
                  <div class="text-xs">{{ formatDateTime(row.first_paid_at) }}</div>
                </div>
                <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
              </template>

              <template #cell-total_paid_amount="{ value }">
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ formatMoney(value) }}
                </span>
              </template>

              <template #cell-total_commission_amount="{ value }">
                <span class="font-medium text-emerald-600 dark:text-emerald-400">
                  {{ formatMoney(value) }}
                </span>
              </template>
            </DataTable>

            <div class="border-t border-gray-100 px-6 py-4 dark:border-dark-700">
              <Pagination
                v-if="inviteePagination.total > 0"
                :page="inviteePagination.page"
                :total="inviteePagination.total"
                :page-size="inviteePagination.page_size"
                @update:page="handleInviteePageChange"
                @update:pageSize="handleInviteePageSizeChange"
              />
            </div>
          </div>

          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('referral.commissions.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('referral.commissions.description') }}
              </p>
            </div>

            <DataTable :columns="commissionColumns" :data="commissions" :loading="loadingCommissions">
              <template #cell-created_at="{ value }">
                <span class="text-sm text-gray-600 dark:text-gray-400">
                  {{ formatDateTime(value) }}
                </span>
              </template>

              <template #cell-user="{ row }">
                <div class="min-w-0">
                  <div class="font-medium text-gray-900 dark:text-white">
                    {{ row.referred_username || t('common.notAvailable') }}
                  </div>
                  <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ row.referred_email || '-' }}
                  </div>
                </div>
              </template>

              <template #cell-commission_type="{ row }">
                <div class="space-y-1">
                  <span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="getCommissionTypeClass(row.commission_type)">
                    {{ getCommissionTypeLabel(row.commission_type) }}
                  </span>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ getCommissionStatusLabel(row.status) }}
                  </div>
                </div>
              </template>

              <template #cell-source_amount="{ row }">
                <div class="text-sm text-gray-700 dark:text-gray-300">
                  <div>{{ formatMoney(row.source_amount) }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ row.rate_snapshot.toFixed(2) }}%
                  </div>
                </div>
              </template>

              <template #cell-commission_amount="{ value }">
                <span class="font-medium text-emerald-600 dark:text-emerald-400">
                  {{ formatMoney(value) }}
                </span>
              </template>

              <template #cell-order="{ row }">
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  <div>{{ row.order_no || '-' }}</div>
                  <div v-if="row.paid_at" class="text-xs">{{ formatDateTime(row.paid_at) }}</div>
                </div>
              </template>
            </DataTable>

            <div class="border-t border-gray-100 px-6 py-4 dark:border-dark-700">
              <Pagination
                v-if="commissionPagination.total > 0"
                :page="commissionPagination.page"
                :total="commissionPagination.total"
                :page-size="commissionPagination.page_size"
                @update:page="handleCommissionPageChange"
                @update:pageSize="handleCommissionPageSizeChange"
              />
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.withdrawals.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('referral.withdrawals.description') }}
            </p>
          </div>

          <DataTable :columns="withdrawalColumns" :data="withdrawals" :loading="loadingWithdrawals">
            <template #cell-created_at="{ value }">
              <span class="text-sm text-gray-600 dark:text-gray-400">
                {{ formatDateTime(value) }}
              </span>
            </template>

            <template #cell-amount="{ row }">
              <div class="text-sm text-gray-700 dark:text-gray-300">
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ formatMoney(row.amount) }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.currency }}
                </div>
              </div>
            </template>

            <template #cell-method="{ row }">
              <div class="text-sm text-gray-700 dark:text-gray-300">
                <div>{{ paymentMethodLabel(row.payment_method) }}</div>
                <div v-if="row.account_name" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.account_name }}
                </div>
              </div>
            </template>

            <template #cell-account="{ row }">
              <span class="text-sm text-gray-600 dark:text-gray-400">
                {{ row.account_identifier || '-' }}
              </span>
            </template>

            <template #cell-status="{ row }">
              <div class="space-y-1">
                <span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="getWithdrawalStatusClass(row.status)">
                  {{ getWithdrawalStatusLabel(row.status) }}
                </span>
                <div v-if="row.reviewed_at" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatDateTime(row.reviewed_at) }}
                </div>
              </div>
            </template>

            <template #cell-notes="{ row }">
              <div class="max-w-[260px] text-sm text-gray-600 dark:text-gray-400">
                <div v-if="row.notes">{{ row.notes }}</div>
                <div v-if="row.review_notes" class="mt-1 text-xs text-gray-500 dark:text-gray-500">
                  {{ row.review_notes }}
                </div>
                <span v-if="!row.notes && !row.review_notes">-</span>
              </div>
            </template>

            <template #cell-actions="{ row }">
              <button
                v-if="row.status === 'pending'"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="cancelingWithdrawalId === row.id"
                @click="cancelWithdrawal(row.id)"
              >
                {{ cancelingWithdrawalId === row.id ? t('common.loading') : t('referral.withdrawals.cancel') }}
              </button>
              <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
            </template>
          </DataTable>

          <div class="border-t border-gray-100 px-6 py-4 dark:border-dark-700">
            <Pagination
              v-if="withdrawalPagination.total > 0"
              :page="withdrawalPagination.page"
              :total="withdrawalPagination.total"
              :page-size="withdrawalPagination.page_size"
              @update:page="handleWithdrawalPageChange"
              @update:pageSize="handleWithdrawalPageSizeChange"
            />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  referralAPI,
  type CreateReferralWithdrawalPayload,
  type ReferralCommission,
  type ReferralInvitee,
  type ReferralSummary,
  type ReferralWithdrawalRequest
} from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { Column } from '@/components/common/types'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const textOr = (key: string, fallback: string) => {
  const text = t(key)
  return text === key ? fallback : text
}

const summary = ref<ReferralSummary | null>(null)
const invitees = ref<ReferralInvitee[]>([])
const commissions = ref<ReferralCommission[]>([])
const withdrawals = ref<ReferralWithdrawalRequest[]>([])

const loadingInitial = ref(true)
const refreshing = ref(false)
const loadingInvitees = ref(false)
const loadingCommissions = ref(false)
const loadingWithdrawals = ref(false)
const submittingWithdrawal = ref(false)
const cancelingWithdrawalId = ref<number | null>(null)

const inviteePagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

const commissionPagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

const withdrawalPagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

const withdrawalForm = reactive<CreateReferralWithdrawalPayload>({
  amount: 0,
  currency: 'CNY',
  payment_method: 'alipay',
  account_name: '',
  account_identifier: '',
  notes: ''
})

const inviteLink = computed(() => {
  const code = summary.value?.referral_code?.trim()
  if (!code || typeof window === 'undefined') {
    return ''
  }
  const url = new URL('/register', window.location.origin)
  url.searchParams.set('invitation_code', code)
  return url.toString()
})

const paymentMethodOptions = computed(() => [
  { value: 'alipay', label: t('referral.paymentMethods.alipay') },
  { value: 'wechat', label: t('referral.paymentMethods.wechat') },
  { value: 'bank', label: t('referral.paymentMethods.bank') },
  { value: 'usdt', label: t('referral.paymentMethods.usdt') },
  { value: 'other', label: t('referral.paymentMethods.other') }
])

const inviteeColumns = computed<Column[]>(() => [
  { key: 'user', label: t('referral.invitees.user') },
  { key: 'registered_at', label: t('referral.invitees.registeredAt') },
  { key: 'first_paid', label: t('referral.invitees.firstPaid') },
  { key: 'total_paid_amount', label: t('referral.invitees.totalPaid') },
  { key: 'total_commission_amount', label: t('referral.invitees.totalCommission') }
])

const commissionColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('referral.commissions.createdAt') },
  { key: 'user', label: t('referral.commissions.user') },
  { key: 'commission_type', label: t('referral.commissions.type') },
  { key: 'source_amount', label: t('referral.commissions.sourceAmount') },
  { key: 'commission_amount', label: t('referral.commissions.commissionAmount') },
  { key: 'order', label: t('referral.commissions.order') }
])

const withdrawalColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('referral.withdrawals.createdAt') },
  { key: 'amount', label: t('referral.withdrawals.amount') },
  { key: 'method', label: t('referral.withdrawals.method') },
  { key: 'account', label: t('referral.withdrawals.account') },
  { key: 'status', label: t('referral.withdrawals.status') },
  { key: 'notes', label: t('referral.withdrawals.notes') },
  { key: 'actions', label: t('referral.withdrawals.actions') }
])

function formatMoney(value: number): string {
  return `$${(value || 0).toFixed(2)}`
}

function paymentMethodLabel(method: string): string {
  switch (method) {
    case 'alipay':
    case 'wechat':
    case 'bank':
    case 'usdt':
    case 'other':
      return t(`referral.paymentMethods.${method}`)
    default:
      return method || '-'
  }
}

function getCommissionTypeLabel(type: string): string {
  if (type === 'first') return t('referral.commissionTypes.first')
  if (type === 'recurring') return t('referral.commissionTypes.recurring')
  return type || '-'
}

function getCommissionStatusLabel(status: string): string {
  if (['recorded', 'reversed', 'pending', 'approved', 'rejected', 'canceled'].includes(status)) {
    return t(`referral.statuses.${status}`)
  }
  return status || '-'
}

function getWithdrawalStatusLabel(status: string): string {
  if (['recorded', 'reversed', 'pending', 'approved', 'rejected', 'canceled'].includes(status)) {
    return t(`referral.statuses.${status}`)
  }
  return status || '-'
}

function getCommissionTypeClass(type: string): string {
  if (type === 'first') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  if (type === 'recurring') {
    return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  }
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

function getWithdrawalStatusClass(status: string): string {
  switch (status) {
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'approved':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'rejected':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
    case 'canceled':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

async function loadSummary(): Promise<void> {
  summary.value = await referralAPI.getSummary()
}

async function loadInvitees(): Promise<void> {
  loadingInvitees.value = true
  try {
    const response = await referralAPI.getInvitees(inviteePagination.page, inviteePagination.page_size)
    invitees.value = response.items
    inviteePagination.total = response.total
  } finally {
    loadingInvitees.value = false
  }
}

async function loadCommissions(): Promise<void> {
  loadingCommissions.value = true
  try {
    const response = await referralAPI.getCommissions(
      commissionPagination.page,
      commissionPagination.page_size
    )
    commissions.value = response.items
    commissionPagination.total = response.total
  } finally {
    loadingCommissions.value = false
  }
}

async function loadWithdrawals(): Promise<void> {
  loadingWithdrawals.value = true
  try {
    const response = await referralAPI.listWithdrawalRequests(
      withdrawalPagination.page,
      withdrawalPagination.page_size
    )
    withdrawals.value = response.items
    withdrawalPagination.total = response.total
  } finally {
    loadingWithdrawals.value = false
  }
}

async function refreshAll(): Promise<void> {
  refreshing.value = true
  try {
    await Promise.all([loadSummary(), loadInvitees(), loadCommissions(), loadWithdrawals()])
  } catch (error: any) {
    appStore.showError(error.message || t('referral.loadFailed'))
  } finally {
    refreshing.value = false
    loadingInitial.value = false
  }
}

async function copyCode(): Promise<void> {
  const code = summary.value?.referral_code
  if (!code) return
  await copyToClipboard(code, t('referral.codeCopied'))
}

async function copyLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('referral.linkCopied'))
}

async function submitWithdrawal(): Promise<void> {
  if (!summary.value?.can_withdraw) {
    appStore.showError(t('referral.withdraw.notEligible'))
    return
  }
  if (!withdrawalForm.amount || withdrawalForm.amount <= 0) {
    appStore.showError(t('referral.withdraw.amountRequired'))
    return
  }
  if (!withdrawalForm.account_identifier.trim()) {
    appStore.showError(t('referral.withdraw.accountRequired'))
    return
  }

  submittingWithdrawal.value = true
  try {
    await referralAPI.createWithdrawalRequest({
      ...withdrawalForm,
      account_name: withdrawalForm.account_name?.trim() || undefined,
      account_identifier: withdrawalForm.account_identifier.trim(),
      notes: withdrawalForm.notes?.trim() || undefined
    })
    appStore.showSuccess(t('referral.withdraw.submitSuccess'))
    withdrawalForm.amount = 0
    withdrawalForm.account_name = ''
    withdrawalForm.account_identifier = ''
    withdrawalForm.notes = ''
    withdrawalPagination.page = 1
    await refreshAll()
  } catch (error: any) {
    appStore.showError(error.message || t('referral.withdraw.submitFailed'))
  } finally {
    submittingWithdrawal.value = false
  }
}

async function cancelWithdrawal(id: number): Promise<void> {
  if (!window.confirm(t('referral.withdrawals.cancelConfirm'))) {
    return
  }
  cancelingWithdrawalId.value = id
  try {
    await referralAPI.cancelWithdrawalRequest(id)
    appStore.showSuccess(t('referral.withdrawals.cancelSuccess'))
    await refreshAll()
  } catch (error: any) {
    appStore.showError(error.message || t('referral.withdrawals.cancelFailed'))
  } finally {
    cancelingWithdrawalId.value = null
  }
}

function handleInviteePageChange(page: number): void {
  inviteePagination.page = page
  void loadInvitees()
}

function handleInviteePageSizeChange(pageSize: number): void {
  inviteePagination.page_size = pageSize
  inviteePagination.page = 1
  void loadInvitees()
}

function handleCommissionPageChange(page: number): void {
  commissionPagination.page = page
  void loadCommissions()
}

function handleCommissionPageSizeChange(pageSize: number): void {
  commissionPagination.page_size = pageSize
  commissionPagination.page = 1
  void loadCommissions()
}

function handleWithdrawalPageChange(page: number): void {
  withdrawalPagination.page = page
  void loadWithdrawals()
}

function handleWithdrawalPageSizeChange(pageSize: number): void {
  withdrawalPagination.page_size = pageSize
  withdrawalPagination.page = 1
  void loadWithdrawals()
}

onMounted(() => {
  void refreshAll()
})
</script>
