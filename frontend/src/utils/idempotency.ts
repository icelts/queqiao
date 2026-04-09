export function createIdempotencyKey(scope: string): string {
  const prefix = scope.trim() || 'request'
  const randomPart =
    typeof globalThis.crypto?.randomUUID === 'function'
      ? globalThis.crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(16).slice(2)}`

  return `${prefix}-${randomPart}`
}
