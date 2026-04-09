import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'
import { createIdempotencyKey } from '@/utils/idempotency'

export interface RechargeOrder {
  id: number
  user_id: number
  order_no: string
  external_order_id?: string | null
  channel: string
  source: string
  currency: string
  amount: number
  credited_amount: number
  status: string
  paid_at?: string | null
  refunded_at?: string | null
  callback_idempotency_key?: string
  callback_raw?: string | null
  notes?: string | null
  created_at: string
  updated_at: string
}

export interface XunhuPaymentResult {
  provider: string
  open_order_id?: string
  payment_url?: string
  qrcode_url?: string
  expires_at?: string
  response_raw?: Record<string, unknown>
}

export interface XunhuQueryPaymentResult {
  provider: string
  status: string
  order_no?: string
  open_order_id?: string
  transaction_id?: string
  total_fee?: number
  response_raw?: Record<string, unknown>
}

export interface CreateRechargeOrderRequest {
  amount: number
  channel?: string
  currency?: string
  title?: string
  notes?: string
}

export interface CreateRechargeOrderResponse {
  order: RechargeOrder
  payment?: XunhuPaymentResult
}

export interface ReconcileRechargeOrderResponse {
  order: RechargeOrder
  payment?: XunhuQueryPaymentResult
}

export async function createRechargeOrder(
  payload: CreateRechargeOrderRequest
): Promise<CreateRechargeOrderResponse> {
  const { data } = await apiClient.post<CreateRechargeOrderResponse>('/recharges/orders', payload, {
    headers: {
      'Idempotency-Key': createIdempotencyKey('recharge-order')
    }
  })
  return data
}

export async function listRechargeOrders(
  page: number = 1,
  pageSize: number = 10
): Promise<PaginatedResponse<RechargeOrder>> {
  const { data } = await apiClient.get<PaginatedResponse<RechargeOrder>>('/recharges/orders', {
    params: {
      page,
      page_size: pageSize
    }
  })
  return data
}

export async function getRechargeOrder(orderNo: string): Promise<RechargeOrder> {
  const { data } = await apiClient.get<RechargeOrder>(`/recharges/orders/${encodeURIComponent(orderNo)}`)
  return data
}

export async function reconcileRechargeOrder(orderNo: string): Promise<ReconcileRechargeOrderResponse> {
  const { data } = await apiClient.post<ReconcileRechargeOrderResponse>(
    `/recharges/orders/${encodeURIComponent(orderNo)}/reconcile`
  )
  return data
}

export const rechargeAPI = {
  createRechargeOrder,
  listRechargeOrders,
  getRechargeOrder,
  reconcileRechargeOrder
}

export default rechargeAPI
