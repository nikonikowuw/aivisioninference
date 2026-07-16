/**
 * WebSocket message type & topic constants.
 *
 * 命令类型 (send 时的 type 字段)：
 *   subscribe   — 订阅一个 topic
 *   unsubscribe — 取消订阅一个 topic
 *
 * 事件类型 (received msg.type) / topic (payload.topic)：
 *   前端用 topic 订阅后端某类事件，后端广播时 type 即为该事件名。
 */
export const WS_CMD = {
  SUBSCRIBE: 'subscribe',
  UNSUBSCRIBE: 'unsubscribe',
} as const;

export const WS_TOPIC = {
  /** 节点状态变更（online/offline/error） */
  EDGE_NODE_STATUS: 'edge-node-status',
  /** 节点实时性能指标（CPU、内存、网速） */
  EDGE_NODE_METRICS: 'edge-node-metrics',
  /** 节点引擎指标（流数、编解码插槽、加速器利用率） */
  EDGE_NODE_ENGINE_METRICS: 'edge-node-engine-metrics',
  /** 节点算法部署状态变更 */
  EDGE_NODE_ALGO_STATUS: 'edge-node-algo-status',
  /** 通用指标 topic（detail 页面使用） */
  METRICS: 'metrics',
} as const;
