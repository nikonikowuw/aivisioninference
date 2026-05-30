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
      system: 'System',
    },
  },
  table: {
    columns: {
      username: 'Operator',
      actionType: 'Action',
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
  actionTypes: {
    action: {
      view: {
        users: 'View users',
        roles: 'View roles',
        permissions: 'View permissions',
        files: 'View files',
        'audit-logs': 'View audit logs',
        tasks: 'View tasks',
        system: 'View system config',
        dashboard: 'View dashboard',
      },
      create: {
        users: 'Create user',
        roles: 'Create role',
        permissions: 'Create permission',
        files: 'Upload file',
        tasks: 'Create task',
      },
      update: {
        users: 'Update user',
        roles: 'Update role',
        permissions: 'Update permission',
        files: 'Update file',
        tasks: 'Update task',
        system: 'Update system config',
      },
      delete: {
        users: 'Delete user',
        roles: 'Delete role',
        permissions: 'Delete permission',
        files: 'Delete file',
        tasks: 'Delete task',
      },
      login: 'User login',
      auth: 'Auth operation',
    },
  },
} as const;

export default auditLogs;
