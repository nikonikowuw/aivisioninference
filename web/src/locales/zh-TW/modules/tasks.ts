export const tasks = {
  title: '任務管理',
  filter: {
    taskTypes: {
      email: '郵件',
      export: '匯出',
      import: '匯入',
      backup: '備份',
    },
  },
  table: {
    columns: {
      id: 'ID',
      type: '類型',
      status: '狀態',
      error: '錯誤',
      createdAt: '建立時間',
      updatedAt: '更新時間',
      actions: '操作',
    },
    status: {
      pending: '等待中',
      running: '執行中',
      completed: '已完成',
      failed: '失敗',
      cancelled: '已取消',
    },
  },
  message: {
    cancelled: '已取消',
    cancelFailed: '取消失敗',
    cancelConfirm: '確定取消該任務？',
  },
  actions: {
    cancel: '取消任務',
  },
} as const;

export default tasks;
