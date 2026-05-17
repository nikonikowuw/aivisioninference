export const tasks = {
  title: 'Task Management',
  filter: {
    taskTypes: {
      email: 'Email',
      export: 'Export',
      import: 'Import',
      backup: 'Backup',
    },
  },
  table: {
    columns: {
      id: 'ID',
      type: 'Type',
      status: 'Status',
      error: 'Error',
      createdAt: 'Created At',
      updatedAt: 'Updated At',
      actions: 'Actions',
    },
    status: {
      pending: 'Pending',
      running: 'Running',
      completed: 'Completed',
      failed: 'Failed',
      cancelled: 'Cancelled',
    },
  },
  message: {
    cancelled: 'Task cancelled',
    cancelFailed: 'Failed to cancel task',
    cancelConfirm: 'Are you sure you want to cancel this task?',
  },
  actions: {
    cancel: 'Cancel Task',
  },
} as const;

export default tasks;
