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
  },
  message: {
    updated: 'Feedback status updated',
    updateFailed: 'Failed to update feedback status',
  },
} as const;

export default feedback;
