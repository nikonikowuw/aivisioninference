export const auditLogs = {
  title: '稽核日誌',
  table: {
    columns: {
      username: '操作者',
      action: '操作',
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
