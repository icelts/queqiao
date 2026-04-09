import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface AdminRechargeOrder {
  id: number
  user_id: number
  user_email?: string
  username?: string
  order_no: string
  external_order_id?: string
  channel: string
  currency: string
  amount: number
  credited_amount: number
  status: 'pending' | 'paid' | 'failed' | 'refunded' | string
  paid_at?: string
  refunded_at?: string
  callback_idempotency_key?: string
  notes?: string
  commission_count: number
  total_commission_amount: number
  recorded_commission_amount: number
  reversed_commission_amount: number
  created_at: string
  updated_at: string
}

export interface AdminRechargeOrderStats {
  total_orders: number
  pending_orders: number
  paid_orders: number
  failed_orders: number
  refunded_orders: number
  total_paid_amount: number
  total_refunded_amount: number
  total_commission_amount: number
}

export interface AdminRechargeOrderFilters {
  status?: string
  channel?: string
  search?: string
  start_date?: string
  end_date?: string
  with_commission?: boolean
  refunded_only?: boolean
}

export interface AdminRechargeOrderDetailResponse {
  order: AdminRechargeOrder
  callback_raw?: string
}

export async function listRechargeOrders(
  page: number = 1,
  pageSize: number = 20,
  filters?: AdminRechargeOrderFilters
): Promise<PaginatedResponse<AdminRechargeOrder>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminRechargeOrder>>(
    '/admin/referrals/recharge-orders',
    {
      params: {
        page,
        page_size: pageSize,
        ...filters
      }
    }
  )
  return data
}

export async function getRechargeOrderStats(
  filters?: AdminRechargeOrderFilters
): Promise<AdminRechargeOrderStats> {
  const { data } = await apiClient.get<AdminRechargeOrderStats>(
    '/admin/referrals/recharge-orders/stats',
    { params: filters }
  )
  return data
}

export async function getRechargeOrderDetail(id: number): Promise<AdminRechargeOrderDetailResponse> {
  const { data } = await apiClient.get<AdminRechargeOrderDetailResponse>(
    `/admin/referrals/recharge-orders/${id}`
  )
  return data
}

const rechargeOrdersAPI = {
  listRechargeOrders,
  getRechargeOrderStats,
  getRechargeOrderDetail
}

export default rechargeOrdersAPI
