/**
 * 菜单相关类型定义
 *
 * 替代 auth.ts 中 `menus: any[]` 的类型丢失问题。
 * 与 api/role.ts 中 MenuItem 保持一致，包含完整服务端字段。
 */

export interface MenuItem {
  id: number
  name: string
  path: string
  icon: string
  parent_id: number
  sort_order: number
  type: number
  children?: MenuItem[]
}
