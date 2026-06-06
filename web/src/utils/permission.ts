/**
 * 判断当前用户是否拥有指定权限码。
 * 支持通配符 `*`（超级管理员），命中任一即返回 true。
 *
 * @param codes  登录返回的 permission_codes 列表
 * @param code   需要校验的权限编码
 */
export function hasPermission(codes: string[] | null | undefined, code: string): boolean {
  if (!codes || codes.length === 0) return false;
  if (codes.includes('*')) return true;
  return codes.includes(code);
}

/**
 * 判断当前用户是否拥有指定权限码集合中的任意一个。
 * 用于 Tab 可见性、批量按钮等场景。
 */
export function hasAnyPermission(codes: string[] | null | undefined, required: string[]): boolean {
  if (!codes || codes.length === 0) return false;
  if (codes.includes('*')) return true;
  return required.some((code) => codes.includes(code));
}
