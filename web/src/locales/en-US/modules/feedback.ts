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
  },
  message: {
    updated: 'Feedback status updated',
    updateFailed: 'Failed to update feedback status',
    batchUpdateConfirm: 'Update the status of selected {{count}} feedback item(s)?',
  },
} as const;

export default feedback;
