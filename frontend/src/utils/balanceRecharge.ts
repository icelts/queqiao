export function calculateBalanceCredit(amountCny: number, ratio: number): number {
  if (!Number.isFinite(amountCny) || amountCny <= 0) {
    return 0
  }
  if (!Number.isFinite(ratio) || ratio <= 0) {
    return 0
  }
  return Math.round(amountCny * ratio * 1e8) / 1e8
}
