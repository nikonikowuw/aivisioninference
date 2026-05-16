export const common = {
  button: {
    submit: 'Submit',
    cancel: 'Cancel',
    save: 'Save',
    create: 'Create',
    delete: 'Delete',
    edit: 'Edit',
    confirm: 'Confirm',
    close: 'Close',
    back: 'Back',
    next: 'Next',
    search: 'Search',
    refresh: 'Refresh',
    export: 'Export',
    import: 'Import',
  },
  status: {
    loading: 'Loading...',
    success: 'Success',
    error: 'Error',
    warning: 'Warning',
    info: 'Info',
  },
  message: {
    confirmDelete: 'Are you sure you want to delete?',
    confirmCancel: 'Are you sure you want to cancel?',
    operationSuccess: 'Operation successful',
    operationFailed: 'Operation failed',
    networkError: 'Network error, please try again later',
    unauthorized: 'Unauthorized, please sign in again',
    forbidden: 'You do not have permission to perform this action',
    notFound: 'The requested resource was not found',
    serverError: 'Server error',
    loadFailed: 'Failed to load data',
    requiredFields: 'Please fill in required fields',
  },
  user: {
    defaultName: 'User',
    logout: 'Logout',
  },
  pagination: {
    total: '{{total}} total',
    page: 'Page {{page}}',
    pageSize: '{{size}} per page',
  },
} as const;

export default common;
