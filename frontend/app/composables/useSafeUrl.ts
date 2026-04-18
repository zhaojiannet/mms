// safeAssetUrl 用于渲染后端返回的资源 URL（如 store_logo_url）
// 防御场景：后端字段被污染为 `javascript:` / `data:text/html` / `//evil.com/...` 等
//
// 规则：
//   - 绝对路径 `/xxx` 允许
//   - http(s):// 的完整 URL 允许（未来可能返回 CDN URL）
//   - 其他（javascript:、data:、相对路径、协议相对 //x）拒绝，返回 ''
export function safeAssetUrl(url: unknown): string {
  if (typeof url !== 'string' || url === '') return ''

  // 绝对路径（相对站内）
  if (url.startsWith('/') && !url.startsWith('//')) return url

  // http(s) 完整 URL
  if (url.startsWith('http://') || url.startsWith('https://')) {
    try {
      // 再用 URL 构造函数过一遍，确保格式合法
      // eslint-disable-next-line no-new
      new URL(url)
      return url
    } catch {
      return ''
    }
  }

  return ''
}
