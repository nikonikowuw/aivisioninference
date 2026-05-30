export const tasks = {
  title: "任务管理",
  filter: {
    taskTypes: {
      email: "邮件",
      export: "导出",
      import: "导入",
      backup: "备份",
    },
  },
  table: {
    columns: {
      type: "类型",
      error: "错误",
    },
    status: {
      pending: "等待中",
      running: "运行中",
      completed: "已完成",
      failed: "失败",
      cancelled: "已取消",
    },
  },
  message: {
    cancelled: "已取消",
    cancelFailed: "取消失败",
    cancelConfirm: "确定取消该任务？",
    confirmCancel: "确认取消",
    batchCancelConfirm: "确定取消选中的 {{count}} 个任务？",
  },
  actions: {
    cancel: "取消任务",
  },
} as const;

export default tasks;
