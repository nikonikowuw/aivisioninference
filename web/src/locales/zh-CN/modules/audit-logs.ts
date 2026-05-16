export const auditLogs = {
  title: '审计日志',
  table: {
    columns: {
      id: 'ID',
      userId: '用户ID',
      action: '操作',
      resourceType: '资源类型',
      resourceId: '资源ID',
      ip: 'IP',
      detail: '详情',
      time: '时间',
    },
  },
} as const;

export default auditLogs;
