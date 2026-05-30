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
    confirmCancel: 'Yes, cancel it',
    batchDone: 'Batch completed: {{success}} succeeded, {{failed}} failed',
    batchCancelConfirm: 'Cancel the selected {{count}} task(s)?',
    operationFailed: 'Operation failed',
  },
  batch: {
    selected: '{{count}} items selected',
  },
  actions: {
    cancel: 'Cancel Task',
    exportSelected: 'Export Selected',
  },
} as const;

export default tasks;
