<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="!hasAnyFeature">
        <div class="card flex items-center justify-center p-10 text-center">
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="creditCard" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('purchase.notEnabledTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('purchase.notEnabledDesc') }}
            </p>
          </div>
        </div>
      </template>

      <template v-else>
        <div v-if="loadingProducts || subscriptionProducts.length > 0" class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('purchase.subscription.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('purchase.subscription.description') }}
            </p>
          </div>

          <div class="p-6">
            <div v-if="loadingProducts" class="flex items-center justify-center py-6">
              <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
            </div>

            <div v-else class="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
              <article
                v-for="product in subscriptionProducts"
                :key="product.group_id"
                class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm transition hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                      {{ product.group_name }}
                    </h3>
                    <p class="mt-1 min-h-[2.5rem] text-sm text-gray-500 dark:text-gray-400">
                      {{ product.description || t('purchase.subscription.noDescription') }}
                    </p>
                  </div>
                  <span
                    class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
                    :class="
                      product.is_renewal
                        ? 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
                        : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                    "
                  >
                    {{
                      product.is_renewal
                        ? t('purchase.subscription.renew')
                        : t('purchase.subscription.buy')
                    }}
                  </span>
                </div>

                <div class="mt-5 flex items-end justify-between">
                  <div>
                    <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.subscription.price') }}
                    </div>
                    <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
                      {{ formatMoney(product.purchase_price) }}
                    </div>
                  </div>
                  <div class="text-right text-sm text-gray-500 dark:text-gray-400">
                    {{ t('purchase.subscription.validityDaysShort', { days: product.validity_days }) }}
                  </div>
                </div>

                <div class="mt-4 rounded-2xl bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:bg-dark-700/60 dark:text-gray-300">
                  {{ formatSubscriptionLimitText(product) }}
                </div>

                <div
                  v-if="product.current_subscription?.expires_at"
                  class="mt-3 text-sm text-gray-500 dark:text-gray-400"
                >
                  {{
                    t('purchase.subscription.currentExpires', {
                      date: formatDateTimeSafe(product.current_subscription.expires_at)
                    })
                  }}
                </div>

                <button
                  type="button"
                  class="btn btn-primary mt-5 w-full"
                  :disabled="creatingSubscriptionGroupId === product.group_id"
                  @click="createSubscriptionOrder(product)"
                >
                  <span v-if="creatingSubscriptionGroupId === product.group_id">
                    {{ t('purchase.subscription.creating') }}
                  </span>
                  <span v-else>{{ getSubscriptionActionLabel(product) }}</span>
                </button>
              </article>
            </div>
          </div>
        </div>

        <div
          v-if="xunhuEnabled"
          class="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]"
        >
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('purchase.recharge.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('purchase.recharge.description') }}
              </p>
            </div>

            <div class="space-y-6 p-6">
              <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div class="rounded-2xl bg-primary-50 p-4 dark:bg-primary-500/10">
                  <div class="text-xs font-medium uppercase tracking-wide text-primary-600 dark:text-primary-300">
                    {{ t('purchase.recharge.currentBalance') }}
                  </div>
                  <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                    {{ formatBalanceAmount(currentBalance) }}
                  </div>
                  <div class="mt-1 text-xs text-primary-700 dark:text-primary-200">
                    {{ t('purchase.recharge.balanceUnit') }}
                  </div>
                </div>

                <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/60">
                  <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                    {{ t('purchase.recharge.channel') }}
                  </div>
                  <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                    {{ t('purchase.recharge.channelXunhu') }}
                  </div>
                </div>

                <div class="rounded-2xl bg-emerald-50 p-4 dark:bg-emerald-500/10">
                  <div class="text-xs font-medium uppercase tracking-wide text-emerald-600 dark:text-emerald-300">
                    {{ t('purchase.recharge.commissionRule') }}
                  </div>
                  <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                    {{ t('purchase.recharge.commissionRuleHint') }}
                  </div>
                </div>
              </div>

              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('purchase.recharge.amount') }}
                </label>
                <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <button
                    v-for="preset in amountPresets"
                    :key="preset"
                    type="button"
                    class="rounded-xl border px-4 py-3 text-left transition"
                    :class="
                      rechargeAmount === preset
                        ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
                        : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200'
                    "
                    @click="rechargeAmount = preset"
                  >
                    <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('purchase.recharge.preset') }}</div>
                    <div class="mt-1 text-lg font-semibold">{{ formatMoney(preset) }}</div>
                  </button>
                </div>

                <div class="mt-4">
                  <input
                    v-model.number="rechargeAmount"
                    type="number"
                    min="0.01"
                    step="0.01"
                    class="input max-w-xs"
                    :placeholder="t('purchase.recharge.amountPlaceholder')"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('purchase.recharge.amountHint') }}
                  </p>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t('purchase.recharge.ratioHint', {
                        ratio: formatBalanceRatio(balanceRechargeRatio)
                      })
                    }}
                  </p>
                  <div class="mt-3 rounded-2xl bg-primary-50 px-4 py-3 text-sm font-medium text-primary-700 dark:bg-primary-500/10 dark:text-primary-200">
                    {{
                      t('purchase.recharge.creditedBalancePreview', {
                        amount: formatBalanceAmount(estimatedBalanceCredit)
                      })
                    }}
                  </div>
                </div>
              </div>

              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('purchase.recharge.notes') }}
                </label>
                <input
                  v-model="rechargeNotes"
                  type="text"
                  maxlength="120"
                  class="input"
                  :placeholder="t('purchase.recharge.notesPlaceholder')"
                />
              </div>

              <div class="flex flex-wrap items-center gap-3">
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="creatingOrder"
                  @click="createOrder"
                >
                  <span v-if="creatingOrder">{{ t('purchase.recharge.creating') }}</span>
                  <span v-else>{{ t('purchase.recharge.createOrder') }}</span>
                </button>
                <button
                  type="button"
                  class="btn btn-secondary"
                  :disabled="loadingOrders"
                  @click="loadOrders"
                >
                  <Icon name="refresh" size="sm" class="mr-1.5" />
                  {{ t('purchase.recharge.refreshOrders') }}
                </button>
              </div>
            </div>
          </div>

          <div class="space-y-6">
            <div class="card overflow-hidden">
              <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('purchase.recharge.currentOrder') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('purchase.recharge.currentOrderHint') }}
                </p>
              </div>

              <div v-if="activeOrder" class="space-y-5 p-6">
                <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div>
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.orders.orderNo') }}
                    </div>
                    <div class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white">
                      {{ activeOrder.order_no }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ getOrderSubtitle(activeOrder) }}
                    </div>
                    <div v-if="isPendingStatus(activeOrder.status)" class="mt-2">
                      <span
                        class="inline-flex rounded-full px-2.5 py-1 text-[11px] font-medium"
                        :class="getPendingStatusClass(activeOrder)"
                      >
                        {{ getPendingStatusLabel(activeOrder) }}
                      </span>
                    </div>
                    <div
                      v-if="canReopenPayment(activeOrder)"
                      class="mt-2 text-xs font-medium"
                      :class="getPaymentRemainingClass(activeOrder)"
                    >
                      {{ formatPaymentRemaining(activeOrder) }}
                    </div>
                  </div>
                  <div>
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.orders.status') }}
                    </div>
                    <span
                      class="mt-1 inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
                      :class="getStatusClass(activeOrder.status)"
                    >
                      {{ getStatusLabel(activeOrder.status) }}
                    </span>
                  </div>
                  <div>
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.orders.amount') }}
                    </div>
                    <div class="mt-1 text-base font-semibold text-gray-900 dark:text-white">
                      {{ formatMoney(activeOrder.amount) }}
                    </div>
                  </div>
                  <div v-if="!isSubscriptionOrder(activeOrder)">
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.recharge.creditedBalance') }}
                    </div>
                    <div class="mt-1 text-base font-semibold text-gray-900 dark:text-white">
                      {{ formatBalanceAmount(activeOrder.credited_amount) }}
                    </div>
                  </div>
                  <div>
                    <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      {{ t('purchase.orders.createdAt') }}
                    </div>
                    <div class="mt-1 text-sm text-gray-700 dark:text-gray-300">
                      {{ formatDateTimeSafe(activeOrder.created_at) }}
                    </div>
                  </div>
                </div>

                <div class="flex flex-wrap items-center gap-3">
                  <button
                    v-if="canReopenPayment(activeOrder)"
                    type="button"
                    class="btn btn-primary"
                    @click="openPaymentModal(activeOrder, activePayment)"
                  >
                    <Icon name="creditCard" size="sm" class="mr-1.5" />
                    {{ t('purchase.paymentModal.reopen') }}
                  </button>
                  <button
                    v-if="activePayment?.payment_url"
                    type="button"
                    class="btn btn-primary"
                    @click="openPayment(activePayment.payment_url)"
                  >
                    <Icon name="externalLink" size="sm" class="mr-1.5" />
                    {{ t('purchase.recharge.openPayment') }}
                  </button>
                  <button
                    v-if="activeOrder.status === 'pending'"
                    type="button"
                    class="btn btn-secondary"
                    @click="refreshPendingOrder(activeOrder.order_no, false)"
                  >
                    <Icon name="refresh" size="sm" class="mr-1.5" />
                    {{ t('purchase.recharge.checkStatus') }}
                  </button>
                </div>
              </div>

              <div v-else class="p-6 text-sm text-gray-500 dark:text-gray-400">
                {{ t('purchase.recharge.currentOrderEmpty') }}
              </div>
            </div>

            <div class="card overflow-hidden">
              <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('purchase.orders.title') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('purchase.orders.description') }}
                </p>
              </div>

              <div v-if="loadingOrders" class="flex items-center justify-center py-10">
                <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
              </div>

              <div v-else-if="orders.length === 0" class="p-6 text-sm text-gray-500 dark:text-gray-400">
                {{ t('purchase.orders.empty') }}
              </div>

              <div v-else class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
                  <thead class="bg-gray-50 dark:bg-dark-800/80">
                    <tr>
                      <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        {{ t('purchase.orders.orderNo') }}
                      </th>
                      <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        {{ t('purchase.orders.amount') }}
                      </th>
                      <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        {{ t('purchase.orders.status') }}
                      </th>
                      <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        {{ t('purchase.orders.createdAt') }}
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                    <tr v-for="order in orders" :key="order.order_no">
                      <td class="px-6 py-4">
                        <div class="font-mono text-sm text-gray-900 dark:text-white">
                          {{ order.order_no }}
                        </div>
                        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          {{ getOrderSubtitle(order) }}
                        </div>
                        <div v-if="isPendingStatus(order.status)" class="mt-2">
                          <span
                            class="inline-flex rounded-full px-2.5 py-1 text-[11px] font-medium"
                            :class="getPendingStatusClass(order)"
                          >
                            {{ getPendingStatusLabel(order) }}
                          </span>
                        </div>
                        <div
                          v-if="canReopenPayment(order)"
                          class="mt-2 text-xs font-medium"
                          :class="getPaymentRemainingClass(order)"
                        >
                          {{ formatPaymentRemaining(order) }}
                        </div>
                        <button
                          v-if="canReopenPayment(order)"
                          type="button"
                          class="mt-2 text-xs font-medium text-primary-600 hover:text-primary-500 dark:text-primary-300"
                          @click="openPaymentModal(order)"
                        >
                          {{ t('purchase.paymentModal.reopen') }}
                        </button>
                      </td>
                      <td class="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">
                        {{ formatMoney(order.amount) }}
                      </td>
                      <td class="px-6 py-4">
                        <span
                          class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
                          :class="getStatusClass(order.status)"
                        >
                          {{ getStatusLabel(order.status) }}
                        </span>
                      </td>
                      <td class="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                        {{ formatDateTimeSafe(order.created_at) }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="purchaseEntryEnabled && !hasEmbeddedPurchase"
          class="card border border-amber-200 bg-amber-50/80 p-6 dark:border-amber-900/50 dark:bg-amber-900/10"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" class="mt-0.5 text-amber-500" />
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('purchase.notConfiguredTitle') }}
              </h3>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                {{ t('purchase.notConfiguredDesc') }}
              </p>
            </div>
          </div>
        </div>

        <div v-if="hasEmbeddedPurchase" class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('purchase.embeddedTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('purchase.embeddedDescription') }}
                </p>
              </div>
              <a
                :href="purchaseUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="btn btn-secondary btn-sm"
              >
                <Icon name="externalLink" size="sm" class="mr-1.5" />
                {{ t('purchase.openInNewTab') }}
              </a>
            </div>
          </div>

          <div class="purchase-embed-shell">
            <iframe :src="purchaseUrl" class="purchase-embed-frame" allowfullscreen></iframe>
          </div>
        </div>
      </template>
    </div>
    <BaseDialog
      :show="showPaymentModal"
      :title="paymentModalTitle"
      width="narrow"
      :close-on-click-outside="true"
      @close="closePaymentModal"
    >
      <div v-if="paymentModalOrder && paymentModalPayment" class="space-y-4 payment-modal-content">
        <div class="rounded-[28px] border border-emerald-100 bg-gradient-to-b from-emerald-50 to-white p-4 text-center dark:border-emerald-900/40 dark:from-emerald-950/40 dark:to-dark-900">
          <div class="inline-flex items-center gap-2 rounded-full bg-emerald-100 px-3 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
            <Icon name="creditCard" size="sm" />
            {{ t('purchase.paymentModal.scanBadge') }}
          </div>
          <img
            v-if="paymentModalPayment.qrcode_url"
            :src="paymentModalPayment.qrcode_url"
            :alt="t('purchase.recharge.scanToPay')"
            class="mx-auto mt-4 h-64 w-64 rounded-3xl bg-white object-contain p-4 shadow-[0_20px_60px_rgba(15,23,42,0.12)]"
          />
          <div v-else class="mt-4 text-sm text-gray-500 dark:text-gray-400">
            {{ t('purchase.paymentModal.qrUnavailable') }}
          </div>
        </div>

        <div class="rounded-3xl bg-gray-50 px-4 py-4 text-sm text-gray-600 dark:bg-dark-700/60 dark:text-gray-300">
          <div class="flex items-start justify-between gap-4">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ getOrderSubtitle(paymentModalOrder) }}
              </div>
              <div class="mt-1 break-all font-mono text-xs text-gray-500 dark:text-gray-400">
                {{ paymentModalOrder.order_no }}
              </div>
            </div>
            <div class="text-right">
              <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('purchase.orders.amount') }}
              </div>
              <div class="mt-1 text-base font-semibold text-gray-900 dark:text-white">
                {{ formatMoney(paymentModalOrder.amount) }}
              </div>
            </div>
          </div>
          <div v-if="paymentModalPayment.expires_at" class="mt-3 rounded-2xl bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
            {{ t('purchase.paymentModal.expiresAt', { date: formatDateTimeSafe(paymentModalPayment.expires_at) }) }}
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <button
            v-if="paymentModalPayment.payment_url"
            type="button"
            class="btn btn-primary"
            @click="openPayment(paymentModalPayment.payment_url)"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" />
            {{ t('purchase.recharge.openPayment') }}
          </button>
          <button
            v-if="paymentModalOrder.status === 'pending'"
            type="button"
            class="btn btn-secondary"
            @click="refreshPendingOrder(paymentModalOrder.order_no, false)"
          >
            <Icon name="refresh" size="sm" class="mr-1.5" />
            {{ t('purchase.recharge.checkStatus') }}
          </button>
        </div>
      </div>
      <div v-else class="py-4 text-sm text-gray-500 dark:text-gray-400">
        {{ t('purchase.paymentModal.unavailable') }}
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { rechargeAPI, type RechargeOrder, type XunhuPaymentResult } from '@/api'
import subscriptionsAPI, { type SubscriptionPurchaseOption } from '@/api/subscriptions'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { calculateBalanceCredit } from '@/utils/balanceRecharge'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'
import { formatDateTime } from '@/utils/format'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const amountPresets = [30, 50, 100, 200]

const loading = ref(false)
const creatingOrder = ref(false)
const creatingSubscriptionGroupId = ref<number | null>(null)
const loadingOrders = ref(false)
const loadingProducts = ref(false)
const rechargeAmount = ref<number | null>(50)
const rechargeNotes = ref('')
const orders = ref<RechargeOrder[]>([])
const activeOrder = ref<RechargeOrder | null>(null)
const activePayment = ref<XunhuPaymentResult | null>(null)
const subscriptionProducts = ref<SubscriptionPurchaseOption[]>([])
const purchaseTheme = ref<'light' | 'dark'>('light')
const pendingOrderNo = ref('')
const showPaymentModal = ref(false)
const paymentModalOrder = ref<RechargeOrder | null>(null)
const paymentModalPayment = ref<XunhuPaymentResult | null>(null)
const paymentNow = ref(Date.now())

let themeObserver: MutationObserver | null = null
let pendingOrderTimer: ReturnType<typeof setInterval> | null = null
let paymentClockTimer: ReturnType<typeof setInterval> | null = null
const PAYMENT_SESSION_STORAGE_KEY = 'purchase_payment_sessions'

const purchaseEntryEnabled = computed(() => {
  return appStore.cachedPublicSettings?.purchase_subscription_enabled ?? false
})

const xunhuEnabled = computed(() => {
  return appStore.cachedPublicSettings?.xunhupay_enabled ?? false
})

const balanceRechargeRatio = computed(() => {
  const ratio = Number(appStore.cachedPublicSettings?.balance_recharge_ratio ?? 1)
  return Number.isFinite(ratio) && ratio > 0 ? ratio : 1
})

const estimatedBalanceCredit = computed(() => {
  return calculateBalanceCredit(Number(rechargeAmount.value || 0), balanceRechargeRatio.value)
})

const hasAnyFeature = computed(() => purchaseEntryEnabled.value || xunhuEnabled.value)

const purchaseUrl = computed(() => {
  const baseUrl = (appStore.cachedPublicSettings?.purchase_subscription_url || '').trim()
  return buildEmbeddedUrl(baseUrl, authStore.user?.id, authStore.token, purchaseTheme.value, locale.value)
})

const hasEmbeddedPurchase = computed(() => purchaseEntryEnabled.value && isValidUrl.value)

const isValidUrl = computed(() => {
  const url = purchaseUrl.value
  return url.startsWith('http://') || url.startsWith('https://')
})

const currentBalance = computed(() => authStore.user?.balance ?? 0)
const paymentModalTitle = computed(() => {
  if (!paymentModalOrder.value) {
    return t('purchase.paymentModal.title')
  }
  return `${t('purchase.paymentModal.title')} / ${getOrderSubtitle(paymentModalOrder.value)}`
})

type SubscriptionOrderMeta = {
  group_id: number
  group_name: string
  validity_days: number
  purchase_price: number
  currency: string
}

type PaymentSessionMap = Record<string, XunhuPaymentResult>

function formatMoney(value: number) {
  return new Intl.NumberFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(Number(value || 0))
}

function formatBalanceAmount(value: number) {
  return new Intl.NumberFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 4
  }).format(Number(value || 0))
}

function formatBalanceRatio(ratio: number) {
  return `1 CNY = ${formatBalanceAmount(ratio)} ${t('purchase.recharge.balanceUnit')}`
}

function formatDateTimeSafe(value?: string | null) {
  return value ? formatDateTime(value) : '-'
}

function isPaymentExpired(payment?: XunhuPaymentResult | null) {
  if (!payment?.expires_at) {
    return false
  }
  return new Date(payment.expires_at).getTime() <= paymentNow.value
}

function loadPaymentSessions(): PaymentSessionMap {
  if (typeof window === 'undefined') {
    return {}
  }
  try {
    const raw =
      localStorage.getItem(PAYMENT_SESSION_STORAGE_KEY) ||
      sessionStorage.getItem(PAYMENT_SESSION_STORAGE_KEY)
    if (!raw) {
      return {}
    }
    const parsed = JSON.parse(raw) as PaymentSessionMap
    const next: PaymentSessionMap = {}
    for (const [orderNo, payment] of Object.entries(parsed || {})) {
      if (!isPaymentExpired(payment)) {
        next[orderNo] = payment
      }
    }
    return next
  } catch {
    return {}
  }
}

const paymentSessions = ref<PaymentSessionMap>(loadPaymentSessions())

function persistPaymentSessions() {
  if (typeof window === 'undefined') {
    return
  }
  localStorage.setItem(PAYMENT_SESSION_STORAGE_KEY, JSON.stringify(paymentSessions.value))
  sessionStorage.removeItem(PAYMENT_SESSION_STORAGE_KEY)
}

function cachePaymentSession(orderNo: string, payment?: XunhuPaymentResult | null) {
  if (!orderNo || !payment) {
    return
  }
  paymentSessions.value = {
    ...paymentSessions.value,
    [orderNo]: payment
  }
  persistPaymentSessions()
}

function removePaymentSession(orderNo: string) {
  if (!orderNo || !paymentSessions.value[orderNo]) {
    return
  }
  const next = { ...paymentSessions.value }
  delete next[orderNo]
  paymentSessions.value = next
  persistPaymentSessions()
}

function getPaymentSession(orderNo?: string | null) {
  if (!orderNo) {
    return null
  }
  const payment = paymentSessions.value[orderNo]
  if (!payment) {
    return null
  }
  if (isPaymentExpired(payment)) {
    removePaymentSession(orderNo)
    return null
  }
  return payment
}

function parseSubscriptionOrderMeta(notes?: string | null): SubscriptionOrderMeta | null {
  if (!notes) {
    return null
  }
  try {
    const parsed = JSON.parse(notes) as Partial<SubscriptionOrderMeta>
    if (!parsed || typeof parsed.group_id !== 'number' || typeof parsed.group_name !== 'string') {
      return null
    }
    return {
      group_id: parsed.group_id,
      group_name: parsed.group_name,
      validity_days: Number(parsed.validity_days || 0),
      purchase_price: Number(parsed.purchase_price || 0),
      currency: String(parsed.currency || 'CNY')
    }
  } catch {
    return null
  }
}

function isSubscriptionOrder(order?: RechargeOrder | null) {
  return order?.source === 'subscription_purchase'
}

function getOrderSubtitle(order?: RechargeOrder | null) {
  if (!order) {
    return '-'
  }
  if (!isSubscriptionOrder(order)) {
    return order.channel
  }
  const meta = parseSubscriptionOrderMeta(order.notes)
  if (!meta) {
    return t('purchase.subscription.orderType')
  }
  return `${meta.group_name} · ${t('purchase.subscription.validityDaysShort', { days: meta.validity_days })}`
}

function formatSubscriptionLimitText(product: SubscriptionPurchaseOption) {
  const segments: string[] = []
  if (product.daily_limit_usd) {
    segments.push(`$${product.daily_limit_usd.toFixed(2)}/${t('purchase.subscription.day')}`)
  }
  if (product.weekly_limit_usd) {
    segments.push(`$${product.weekly_limit_usd.toFixed(2)}/${t('purchase.subscription.week')}`)
  }
  if (product.monthly_limit_usd) {
    segments.push(`$${product.monthly_limit_usd.toFixed(2)}/${t('purchase.subscription.month')}`)
  }
  return segments.join(' · ') || t('purchase.subscription.noLimit')
}

function getSubscriptionActionLabel(product: SubscriptionPurchaseOption) {
  return product.is_renewal
    ? t('purchase.subscription.renewNow')
    : t('purchase.subscription.buyNow')
}

function getStatusLabel(status: string) {
  const normalized = String(status || '').toLowerCase()
  const key = `purchase.statuses.${normalized}`
  const translated = t(key)
  return translated === key ? normalized || '-' : translated
}

function getStatusClass(status: string) {
  switch (String(status || '').toLowerCase()) {
    case 'paid':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'failed':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    case 'refunded':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

function isPendingStatus(status: string) {
  return String(status || '').toLowerCase() === 'pending'
}

function canReopenPayment(order?: RechargeOrder | null) {
  return !!order && isPendingStatus(order.status) && !!getPaymentSession(order.order_no)
}

function openPaymentModal(order: RechargeOrder, payment?: XunhuPaymentResult | null) {
  const resolvedPayment = payment || getPaymentSession(order.order_no)
  if (!resolvedPayment) {
    appStore.showWarning(t('purchase.paymentModal.unavailable'))
    return
  }
  paymentModalOrder.value = order
  paymentModalPayment.value = resolvedPayment
  showPaymentModal.value = true
}

function closePaymentModal() {
  showPaymentModal.value = false
}

function getPaymentRemainingMs(order?: RechargeOrder | null) {
  const payment = getPaymentSession(order?.order_no)
  if (!payment?.expires_at) {
    return null
  }
  const remaining = new Date(payment.expires_at).getTime() - paymentNow.value
  return remaining > 0 ? remaining : null
}

function formatPaymentRemaining(order?: RechargeOrder | null) {
  const remainingMs = getPaymentRemainingMs(order)
  if (remainingMs === null) {
    return t('purchase.paymentModal.expired')
  }

  const totalSeconds = Math.floor(remainingMs / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return t('purchase.paymentModal.remainingHms', {
      hours,
      minutes: String(minutes).padStart(2, '0'),
      seconds: String(seconds).padStart(2, '0')
    })
  }

  return t('purchase.paymentModal.remainingMs', {
    minutes,
    seconds: String(seconds).padStart(2, '0')
  })
}

function getPendingStatusLabel(order?: RechargeOrder | null) {
  if (!order || !isPendingStatus(order.status)) {
    return ''
  }
  return canReopenPayment(order)
    ? t('purchase.paymentModal.pendingLabel')
    : t('purchase.paymentModal.expiredLabel')
}

function getPendingStatusClass(order?: RechargeOrder | null) {
  if (!order || !isPendingStatus(order.status)) {
    return ''
  }
  return canReopenPayment(order)
    ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    : 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
}

function getPaymentRemainingClass(order?: RechargeOrder | null) {
  const remainingMs = getPaymentRemainingMs(order)
  if (remainingMs === null) {
    return 'text-rose-600 dark:text-rose-300'
  }
  if (remainingMs <= 5 * 60 * 1000) {
    return 'text-rose-600 dark:text-rose-300'
  }
  if (remainingMs <= 15 * 60 * 1000) {
    return 'text-amber-600 dark:text-amber-300'
  }
  return 'text-sky-600 dark:text-sky-300'
}

function tickPaymentClock() {
  paymentNow.value = Date.now()
  let changed = false
  const next = { ...paymentSessions.value }
  for (const [orderNo, payment] of Object.entries(next)) {
    if (isPaymentExpired(payment)) {
      delete next[orderNo]
      changed = true
    }
  }
  if (changed) {
    paymentSessions.value = next
    persistPaymentSessions()
  }
}

function upsertOrder(order: RechargeOrder) {
  const next = [...orders.value]
  const index = next.findIndex((item) => item.order_no === order.order_no)
  if (index >= 0) {
    next[index] = order
  } else {
    next.unshift(order)
  }
  orders.value = next.sort((a, b) => b.created_at.localeCompare(a.created_at)).slice(0, 10)
}

function stopPendingOrderPolling() {
  if (pendingOrderTimer) {
    clearInterval(pendingOrderTimer)
    pendingOrderTimer = null
  }
  pendingOrderNo.value = ''
}

function startPendingOrderPolling(orderNo: string) {
  if (!orderNo) {
    stopPendingOrderPolling()
    return
  }
  if (pendingOrderNo.value === orderNo && pendingOrderTimer) {
    return
  }

  stopPendingOrderPolling()
  pendingOrderNo.value = orderNo
  pendingOrderTimer = setInterval(() => {
    refreshPendingOrder(orderNo, true)
  }, 5000)
}

async function refreshPendingOrder(orderNo: string, silent = false) {
  try {
    const previousStatus =
      orders.value.find((item) => item.order_no === orderNo)?.status ?? activeOrder.value?.status ?? ''
    const reconcileResult = await rechargeAPI.reconcileRechargeOrder(orderNo)
    const order = reconcileResult.order ?? (await rechargeAPI.getRechargeOrder(orderNo))
    upsertOrder(order)
    if (activeOrder.value?.order_no === orderNo) {
      activeOrder.value = order
      if (isPendingStatus(order.status)) {
        activePayment.value = getPaymentSession(order.order_no)
      }
    }

    if (!isPendingStatus(order.status)) {
      stopPendingOrderPolling()
      removePaymentSession(orderNo)
      if (paymentModalOrder.value?.order_no === orderNo) {
        closePaymentModal()
      }
      if (order.status === 'paid') {
        await authStore.refreshUser().catch(() => undefined)
        await loadSubscriptionProducts(true)
        if (previousStatus !== 'paid') {
          appStore.showSuccess(
            isSubscriptionOrder(order)
              ? t('purchase.subscription.paidSuccess')
              : t('purchase.recharge.paidSuccess')
          )
        }
      } else if (order.status === 'failed' && previousStatus !== 'failed') {
        appStore.showError(
          isSubscriptionOrder(order)
            ? t('purchase.subscription.failed')
            : t('purchase.recharge.failed')
        )
      } else if (order.status === 'refunded' && previousStatus !== 'refunded') {
        appStore.showWarning(
          isSubscriptionOrder(order)
            ? t('purchase.subscription.refunded')
            : t('purchase.recharge.refunded')
        )
      }
    }
  } catch (error: any) {
    if (!silent) {
      appStore.showError(error.message || t('purchase.recharge.statusCheckFailed'))
    }
  }
}

async function loadSubscriptionProducts(silent = false) {
  if (!xunhuEnabled.value) {
    subscriptionProducts.value = []
    return
  }

  loadingProducts.value = true
  try {
    subscriptionProducts.value = await subscriptionsAPI.getPurchaseOptions()
  } catch (error: any) {
    if (!silent) {
      appStore.showError(error.message || t('purchase.subscription.loadFailed'))
    }
  } finally {
    loadingProducts.value = false
  }
}

async function loadOrders() {
  if (!xunhuEnabled.value) {
    return
  }

  loadingOrders.value = true
  try {
    const data = await rechargeAPI.listRechargeOrders(1, 10)
    orders.value = data.items
    const latestPending = data.items.find((item) => isPendingStatus(item.status))
    if (latestPending) {
      if (!activeOrder.value) {
        activeOrder.value = latestPending
        activePayment.value = getPaymentSession(latestPending.order_no)
      }
      startPendingOrderPolling(latestPending.order_no)
    } else {
      stopPendingOrderPolling()
    }
  } catch (error: any) {
    appStore.showError(error.message || t('purchase.orders.loadFailed'))
  } finally {
    loadingOrders.value = false
  }
}

async function createOrder() {
  const amount = Number(rechargeAmount.value || 0)
  if (!Number.isFinite(amount) || amount <= 0) {
    appStore.showError(t('purchase.recharge.amountInvalid'))
    return
  }

  creatingOrder.value = true
  try {
    const result = await rechargeAPI.createRechargeOrder({
      amount,
      channel: 'xunhupay',
      currency: 'CNY',
      title: t('purchase.recharge.defaultTitle'),
      notes: rechargeNotes.value.trim() || undefined
    })

    activeOrder.value = result.order
    activePayment.value = result.payment || null
    cachePaymentSession(result.order.order_no, result.payment)
    upsertOrder(result.order)
    if (isPendingStatus(result.order.status)) {
      startPendingOrderPolling(result.order.order_no)
    }
    openPaymentModal(result.order, result.payment)
    appStore.showSuccess(t('purchase.recharge.orderCreated'))
  } catch (error: any) {
    appStore.showError(error.message || t('purchase.recharge.createFailed'))
  } finally {
    creatingOrder.value = false
  }
}

async function createSubscriptionOrder(product: SubscriptionPurchaseOption) {
  if (!product?.group_id) {
    return
  }

  creatingSubscriptionGroupId.value = product.group_id
  try {
    const result = await subscriptionsAPI.createPurchaseOrder(product.group_id)
    activeOrder.value = result.order
    activePayment.value = result.payment || null
    cachePaymentSession(result.order.order_no, result.payment)
    upsertOrder(result.order)
    if (isPendingStatus(result.order.status)) {
      startPendingOrderPolling(result.order.order_no)
    }
    openPaymentModal(result.order, result.payment)
    appStore.showSuccess(
      product.is_renewal
        ? t('purchase.subscription.renewCreated')
        : t('purchase.subscription.orderCreated')
    )
  } catch (error: any) {
    appStore.showError(error.message || t('purchase.subscription.createFailed'))
  } finally {
    creatingSubscriptionGroupId.value = null
  }
}

function openPayment(url?: string) {
  if (!url) {
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

onMounted(async () => {
  purchaseTheme.value = detectTheme()
  tickPaymentClock()
  paymentClockTimer = setInterval(() => {
    tickPaymentClock()
  }, 1000)

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      purchaseTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    })
  }

  if (!appStore.publicSettingsLoaded) {
    loading.value = true
    try {
      await appStore.fetchPublicSettings()
    } finally {
      loading.value = false
    }
  }

  if (xunhuEnabled.value) {
    await Promise.all([loadOrders(), loadSubscriptionProducts()])
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
  if (paymentClockTimer) {
    clearInterval(paymentClockTimer)
    paymentClockTimer = null
  }
  stopPendingOrderPolling()
})
</script>

<style scoped>
.purchase-embed-shell {
  height: min(80vh, 960px);
  width: 100%;
  overflow: hidden;
  background: linear-gradient(180deg, rgba(248, 250, 252, 1) 0%, rgba(255, 255, 255, 1) 100%);
}

.purchase-embed-frame {
  display: block;
  height: 100%;
  width: 100%;
  border: 0;
  background: transparent;
}

.payment-modal-content :deep(.btn) {
  min-width: 140px;
}
</style>
