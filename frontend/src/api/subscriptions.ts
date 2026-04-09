/**
 * User Subscription API
 * API for regular users to view their own subscriptions and progress
 */

import { apiClient } from './client'
import type { UserSubscription, SubscriptionProgress } from '@/types'
import type { RechargeOrder, XunhuPaymentResult } from './recharge'
import { createIdempotencyKey } from '@/utils/idempotency'

/**
 * Subscription summary for user dashboard
 */
export interface SubscriptionSummary {
  active_count: number
  subscriptions: Array<{
    id: number
    group_name: string
    status: string
    daily_progress: number | null
    weekly_progress: number | null
    monthly_progress: number | null
    expires_at: string | null
    days_remaining: number | null
  }>
}

export interface SubscriptionPurchaseOption {
  group_id: number
  group_name: string
  description: string
  platform: string
  validity_days: number
  purchase_enabled: boolean
  purchase_price: number
  currency: string
  sort_order: number
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  monthly_limit_usd?: number | null
  current_subscription?: UserSubscription | null
  is_renewal: boolean
}

export interface SubscriptionPurchaseOrderResponse {
  product?: SubscriptionPurchaseOption
  order: RechargeOrder
  payment?: XunhuPaymentResult
}

/**
 * Get list of current user's subscriptions
 */
export async function getMySubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions')
  return response.data
}

/**
 * Get current user's active subscriptions
 */
export async function getActiveSubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions/active')
  return response.data
}

/**
 * Get progress for all user's active subscriptions
 */
export async function getSubscriptionsProgress(): Promise<SubscriptionProgress[]> {
  const response = await apiClient.get<SubscriptionProgress[]>('/subscriptions/progress')
  return response.data
}

/**
 * Get subscription summary for dashboard display
 */
export async function getSubscriptionSummary(): Promise<SubscriptionSummary> {
  const response = await apiClient.get<SubscriptionSummary>('/subscriptions/summary')
  return response.data
}

/**
 * Get progress for a specific subscription
 */
export async function getSubscriptionProgress(
  subscriptionId: number
): Promise<SubscriptionProgress> {
  const response = await apiClient.get<SubscriptionProgress>(
    `/subscriptions/${subscriptionId}/progress`
  )
  return response.data
}

export async function getPurchaseOptions(): Promise<SubscriptionPurchaseOption[]> {
  const response = await apiClient.get<SubscriptionPurchaseOption[]>('/subscriptions/purchase-options')
  return response.data
}

export async function createPurchaseOrder(groupId: number): Promise<SubscriptionPurchaseOrderResponse> {
  const response = await apiClient.post<SubscriptionPurchaseOrderResponse>(
    '/subscriptions/purchase-orders',
    {
      group_id: groupId
    },
    {
      headers: {
        'Idempotency-Key': createIdempotencyKey('subscription-purchase')
      }
    }
  )
  return response.data
}

export default {
  getMySubscriptions,
  getActiveSubscriptions,
  getSubscriptionsProgress,
  getSubscriptionSummary,
  getSubscriptionProgress,
  getPurchaseOptions,
  createPurchaseOrder
}
