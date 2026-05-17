export const auditLogs = {
  title: 'Audit Logs',
  table: {
    columns: {
      username: 'Operator',
      action: 'Action',
      method: 'Method',
      path: 'Path',
      ip: 'IP',
      status: 'Status',
      duration: 'Duration',
      result: 'Result',
      time: 'Time',
    },
    durationMs: '{{value}} ms',
  },
} as const;

export default auditLogs;
