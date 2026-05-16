export const auditLogs = {
  title: 'Audit Logs',
  table: {
    columns: {
      id: 'ID',
      userId: 'User ID',
      action: 'Action',
      resourceType: 'Resource Type',
      resourceId: 'Resource ID',
      ip: 'IP',
      detail: 'Detail',
      time: 'Time',
    },
  },
} as const;

export default auditLogs;
