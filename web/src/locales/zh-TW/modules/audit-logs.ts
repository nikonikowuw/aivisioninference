export const auditLogs = {
  title: '稽核日誌',
  filter: {
    actions: {
      login: '登入',
      logout: '登出',
      create: '建立',
      update: '更新',
      delete: '刪除',
    },
    resourceTypes: {
      user: '使用者',
      role: '角色',
      permission: '權限',
      file: '檔案',
      task: '任務',
    },
  },
  table: {
    columns: {
      username: '操作者',
      action: '操作',
      resourceType: '資源類型',
      method: '方法',
      path: '路徑',
      ip: 'IP',
      status: '狀態碼',
      duration: '耗時',
      result: '結果',
      time: '時間',
    },
    durationMs: '{{value}} ms',
  },
} as const;

export default auditLogs;
