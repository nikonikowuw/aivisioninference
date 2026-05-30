export const auditLogs = {
  title: '审计日志',
  filter: {
    results: {
      success: '成功',
      failed: '失败',
    },
    resourceTypes: {
      user: '用户',
      role: '角色',
      permission: '权限',
      file: '文件',
      task: '任务',
      system: '系统',
      feedback: '反馈',
    },
  },
  table: {
    columns: {
      username: '操作者',
      actionType: '操作类型',
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
  actionTypes: {
    action: {
      view: {
        users: '查看用户',
        roles: '查看角色',
        permissions: '查看权限',
        files: '查看文件',
        'audit-logs': '查看审计日志',
        tasks: '查看任务',
        system: '查看系统配置',
        dashboard: '查看仪表盘',
        feedback: '查看反馈',
      },
      create: {
        users: '创建用户',
        roles: '创建角色',
        permissions: '创建权限',
        files: '上传文件',
        tasks: '创建任务',
        system: '创建系统配置',
        feedback: '提交反馈',
      },
      update: {
        users: '更新用户',
        roles: '更新角色',
        permissions: '更新权限',
        files: '更新文件',
        tasks: '更新任务',
        system: '更新系统配置',
        feedback: '更新反馈状态',
      },
      delete: {
        users: '删除用户',
        roles: '删除角色',
        permissions: '删除权限',
        files: '删除文件',
        tasks: '删除任务',
      },
      login: '用户登录',
      logout: '用户登出',
      auth: '认证操作',
    },
  },
  actions: {
    exportSelected: '导出选中项',
  },
} as const;

export default auditLogs;
