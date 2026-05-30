export const feedback = {
  title: 'User Feedback',
  source: {
    user: 'In-app',
    email: 'Email',
  },
  status: {
    open: 'Open',
    processing: 'Processing',
    resolved: 'Resolved',
    closed: 'Closed',
  },
  table: {
    source: 'Source',
    category: 'Category',
    title: 'Title',
    content: 'Content',
    status: 'Status',
    createdAt: 'Created At',
    actions: 'Actions',
  },
  actions: {
    refresh: 'Update',
    batchUpdateStatus: 'Batch Update Status',
    exportSelected: 'Export Selected',
  },
  message: {
    updated: 'Feedback status updated',
    updateFailed: 'Failed to update feedback status',
    batchDone: 'Batch completed: {{success}} succeeded, {{failed}} failed',
    batchUpdateConfirm: 'Update the status of selected {{count}} feedback item(s)?',
    operationFailed: 'Operation failed',
  },
  batch: {
    selected: '{{count}} items selected',
  },
} as const;

export default feedback;
