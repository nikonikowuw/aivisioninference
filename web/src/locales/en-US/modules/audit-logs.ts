export const auditLogs = {
  title: 'Audit Logs',
  filter: {
    actions: {
      login: 'Login',
      logout: 'Logout',
      create: 'Create',
      update: 'Update',
      delete: 'Delete',
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
      action: 'Action',
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
