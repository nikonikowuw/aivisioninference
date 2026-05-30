export const dashboard = {
  title: '仪表盘',
  stats: {
    totalUsers: '用户总数',
    totalFiles: '文件总数',
    activeTasks: '活跃任务',
  },
  charts: {
    userStats: '用户增长与活跃',
    userNew: '新增用户',
    userActive: '活跃用户',
  },
  auditLog: {
    title: '最近活动',
    empty: '暂无最近活动',
    columns: {
      username: '用户',
      action: '操作',
      method: '方式',
      time: '时间',
    },
  },
  message: {
    loadFailed: '加载仪表盘数据失败',
  },
} as const;

export default dashboard;
