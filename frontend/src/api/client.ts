const base = import.meta.env.VITE_API_BASE_URL ?? ''

/** Go 空 slice 常序列化为 JSON null，列表接口统一归一成 [] */
export function asArray<T>(data: T[] | null | undefined): T[] {
  return Array.isArray(data) ? data : []
}

export async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base}/api/v1${path}`, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers as Record<string, string>) },
    ...init,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((err as { error?: string }).error ?? res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}
