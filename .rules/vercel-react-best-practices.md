# React / Vite 性能与工程质量规范

本规范适用于 `web/` 下 Vite SPA。当前项目使用 React 19、TypeScript、React Router v6、Chakra UI、i18next、TanStack Table、ECharts/ApexCharts、HLS/FLV；不使用 Next.js、RSC、Server Action 或 API Route。

## 1. 数据请求与 API 边界

- API 调用统一走 `web/src/services/api.ts` 及同目录封装，不在组件内散落 `fetch`。
- 无依赖关系的异步请求禁止顺序 `await`，应使用 `Promise.all` 或 `Promise.allSettled`。
- 请求必须携带项目统一鉴权、语言头和错误处理逻辑；不要绕过 `request<T>()` 重写一套响应解析。
- 服务端统一响应 `{ code, message, data }`，前端展示错误时必须走 i18n 映射，不能直接暴露原始异常。
- 列表页优先使用后端分页、筛选、排序；不要在浏览器端拉全量数据再过滤。

## 2. 路由与代码拆分

- 页面级组件应沿用 `web/src/router/` 的动态 import、`React.lazy` 和 `Suspense` 模式。
- 菜单驱动路由以 `authApi.me()` 返回的 `User.menus` 为准，前端 `menu code → component` 映射必须保持稳定。
- 重型依赖必须路由级或组件级懒加载，包括图表、视频播放器、富文本、裁剪、Markdown、文件预览等。
- Suspense fallback 应使用项目统一 loading 组件或骨架屏，避免空白页。
- 动态导入失败时应提供错误态或重试入口，尤其是监控页、图表页和系统配置页。

## 3. Bundle 与依赖控制

- 新增依赖前必须确认现有 Chakra、react-icons、date-fns、ECharts/ApexCharts、工具函数是否已经覆盖需求。
- 禁止为了单个小功能引入大型库；能用浏览器 API 或现有工具完成时不新增依赖。
- 避免从大包根入口导入过多内容；优先使用可 tree-shake 的具名导入或子路径导入。
- 视频、图表、上传、裁剪等依赖不能进入所有页面的首屏 bundle。
- 构建验证以 `cd web && npm run build` 为准，必要时检查 Vite chunk 输出。

## 4. React 渲染正确性

- 禁止在组件函数内部声明子组件；子组件应提升到模块级，避免 remount 和状态丢失。
- 可由 props/state 直接计算出的值不要再同步到 state，也不要用 `useEffect` 做冗余同步。
- 高成本计算使用 `useMemo`；高成本 `useState` 初始值使用惰性初始化 `useState(() => compute())`。
- 不需要触发渲染的可变值使用 `useRef`，如 timer id、WebSocket 实例、播放器实例、滚动坐标、重连次数。
- JSX 条件渲染不要写 `items.length && <List />`，应使用 `items.length > 0 ? <List /> : null` 或显式布尔转换。
- effect 必须清理订阅、定时器、WebSocket、播放器、AbortController 和事件监听。

## 5. 高频交互与大数据视图

- 搜索框、筛选器、大列表联动可使用 `useDeferredValue` 或 `useTransition` 降低阻塞。
- 表格超过 50 行真实 DOM 时优先分页；确需长列表时使用虚拟滚动。
- TanStack Table 状态应保持受控且最小化，避免每次渲染重建 columns/data。
- 图表 option 应使用 memo 化或封装 hook，避免每次 render 重建大型对象导致重绘。
- 视频播放器实例应稳定保存，src 变化、销毁和重连逻辑必须可控。

## 6. Chakra UI 性能实践

- 高频列表项中避免创建大量匿名 style object；可复用组件 variant、`sx` 片段或 memo 化配置。
- 颜色模式切换使用 theme token 和 `useColorModeValue`，不要在组件内散落复杂条件样式。
- Modal、Drawer、Popover 等浮层组件应按需挂载，关闭后释放重型子组件资源。
- Toast 调用应集中封装，避免不同页面重复创建不一致的错误展示逻辑。

## 7. 测试与验证

- 前端测试使用 Vitest + Testing Library + jsdom。
- 修改服务层、路由生成、权限工具、视频组件或复杂表单时，应补充或更新相邻测试。
- 常用命令：
  - `cd web && npm run test`
  - `cd web && npm run build`
  - `cd web && npm run dev`
  - `cd web && npm run preview`

## 8. 不适用项

- 本项目不使用 Next.js，因此不要套用 Server Action、API Route、RSC、`next/dynamic`、Next `<Script>`、Streaming SSR 等规则。
- 如果未来迁移到 Next.js，应新增独立规则文档，而不是在当前 Vite SPA 规范中混写。
