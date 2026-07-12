import { apiBaseUrl } from '@/utils/desktop'

const base = apiBaseUrl()

/** Go 空 slice 常序列化为 JSON null，列表接口统一归一成 [] */
export function asArray<T>(data: T[] | null | undefined): T[] {
  return Array.isArray(data) ? data : []
}

export async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${base}/api/v1${path}`
  let res: Response
  try {
    res = await fetch(url, {
      headers: { 'Content-Type': 'application/json', ...(init?.headers as Record<string, string>) },
      ...init,
    })
  } catch (networkErr) {
    throw new Error(`网络请求失败: ${url} — ${networkErr instanceof Error ? networkErr.message : '未知错误'}`)
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((err as { error?: string }).error ?? res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}
