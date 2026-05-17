export const auditLogs = {
  title: 'Audit Logs',
  filter: {
    results: {
      success: 'Success',
      failed: 'Failed',
    },
    resourceTypes: {
      user: 'User',
      role: 'Role',
      permission: 'Permission',
      file: 'File',
      task: 'Task',
    },
  },
  table: {
    columns: {
      username: 'Operator',
      resourceType: 'Resource Type',
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
