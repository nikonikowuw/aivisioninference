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
      feedback: 'Feedback',
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
        feedback: 'View feedback',
      },
      create: {
        users: 'Create user',
        roles: 'Create role',
        permissions: 'Create permission',
        files: 'Upload file',
        tasks: 'Create task',
        system: 'Create system config',
        feedback: 'Submit feedback',
      },
      update: {
        users: 'Update user',
        roles: 'Update role',
        permissions: 'Update permission',
        files: 'Update file',
        tasks: 'Update task',
        system: 'Update system config',
        feedback: 'Update feedback status',
      },
      delete: {
        users: 'Delete user',
        roles: 'Delete role',
        permissions: 'Delete permission',
        files: 'Delete file',
        tasks: 'Delete task',
      },
      login: 'User login',
      logout: 'User logout',
      auth: 'Auth operation',
    },
  },
  batch: {
    selected: '{{count}} items selected',
  },
} as const;

export default auditLogs;
