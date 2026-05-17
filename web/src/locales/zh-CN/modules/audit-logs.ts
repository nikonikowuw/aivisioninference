export const auditLogs = {
  title: '审计日志',
  table: {
    columns: {
      username: '操作者',
      action: '操作',
      method: '方法',
      path: '路径',
      ip: 'IP',
      status: '状态码',
      duration: '耗时',
      result: '结果',
      time: '时间',
    },
    durationMs: '{{value}} ms',
  },
} as const;

export default auditLogs;
