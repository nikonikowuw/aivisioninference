export const auditLogs = {
  title: '审计日志',
  filter: {
    actions: {
      login: '登录',
      logout: '登出',
      create: '创建',
      update: '更新',
      delete: '删除',
    },
    resourceTypes: {
      user: '用户',
      role: '角色',
      permission: '权限',
      file: '文件',
      task: '任务',
    },
  },
  table: {
    columns: {
      username: '操作者',
      action: '操作',
      resourceType: '资源类型',
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
