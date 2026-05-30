export const feedback = {
  title: "User Feedback",
  source: {
    user: "In-app",
    email: "Email",
  },
  status: {
    open: "Open",
    processing: "Processing",
    resolved: "Resolved",
    closed: "Closed",
  },
  table: {
    source: "Source",
    category: "Category",
    title: "Title",
    content: "Content",
    status: "Status",
    createdAt: "Created At",
    actions: "Actions",
  },
  actions: {
    batchUpdateStatus: "Batch Update Status",
    viewDetails: "View Details",
    copy: "Copy",
  },
  detail: {
    title: "Feedback Details",
    email: "Contact Email",
    updatedAt: "Updated At",
    noEmail: "No Email Provided",
    copySuccess: "Email copied to clipboard",
    copyFailed: "Failed to copy email",
  },
  batch: {
    selected: "{{count}} selected",
  },
  message: {
    updated: "Feedback status updated",
    batchDone:
      "Batch update completed: {{success}} succeeded, {{failed}} failed",
    batchUpdateConfirm:
      "Update the status of selected {{count}} feedback item(s)?",
    operationFailed: "Operation failed",
  },
} as const;

export default feedback;
