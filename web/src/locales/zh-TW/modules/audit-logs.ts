export const auditLogs = {
  title: '稽核日誌',
  table: {
    columns: {
      id: 'ID',
      userId: '使用者ID',
      action: '操作',
      resourceType: '資源類型',
      resourceId: '資源ID',
      ip: 'IP',
      detail: '詳情',
      time: '時間',
    },
  },
} as const;

export default auditLogs;
