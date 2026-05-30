export const tasks = {
  title: "Task Management",
  filter: {
    taskTypes: {
      email: "Email",
      export: "Export",
      import: "Import",
      backup: "Backup",
    },
  },
  table: {
    columns: {
      type: "Type",
      error: "Error",
    },
    status: {
      pending: "Pending",
      running: "Running",
      completed: "Completed",
      failed: "Failed",
      cancelled: "Cancelled",
    },
  },
  message: {
    cancelled: "Task cancelled",
    cancelFailed: "Failed to cancel task",
    cancelConfirm: "Are you sure you want to cancel this task?",
    confirmCancel: "Yes, cancel it",
    batchCancelConfirm: "Cancel the selected {{count}} task(s)?",
  },
  actions: {
    cancel: "Cancel Task",
  },
} as const;

export default tasks;
