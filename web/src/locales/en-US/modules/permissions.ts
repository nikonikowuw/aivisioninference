export const permissions = {
  title: 'Permission Management',
  button: {
    create: 'Add Permission',
  },
  actions: {
    edit: 'Edit',
    delete: 'Delete',
  },
  table: {
    columns: {
      name: 'Permission Name',
      code: 'Permission Code',
      type: 'Type',
    },
  },
  form: {
    name: {
      label: 'Permission Name',
      placeholder: 'Enter permission name',
    },
    code: {
      label: 'Permission Code',
      placeholder: 'e.g., user:create',
    },
    type: {
      label: 'Type',
      menu: 'Menu',
      button: 'Button',
    },
    parentId: {
      label: 'Parent Permission',
      placeholder: 'Select parent permission',
      none: 'None (top level)',
    },
  },
  modal: {
    createTitle: 'Add Permission',
    editTitle: 'Edit Permission',
    deleteTitle: 'Delete Permission',
  },
  message: {
    createSuccess: 'Permission created successfully',
    createFailed: 'Failed to create permission',
    updateSuccess: 'Permission updated successfully',
    updateFailed: 'Failed to update permission',
    deleteSuccess: 'Permission deleted successfully',
    deleteFailed: 'Failed to delete permission',
    deleteConfirm: 'Are you sure you want to delete this permission and all its children?',
    emptyData: 'No permission data',
  },
  codes: {
    dashboard: {
      view: 'View Dashboard',
    },
    user: {
      list: 'User List',
      create: 'Add User',
      edit: 'Edit User',
      delete: 'Delete User',
      view: 'View User',
      'reset-password': 'Reset Password',
      'upload-avatar': 'Upload Avatar',
    },
    role: {
      list: 'Role List',
      create: 'Add Role',
      edit: 'Edit Role',
      delete: 'Delete Role',
      'assign-permissions': 'Assign Permissions',
      'view-permissions': 'View Role Permissions',
      view: 'View Role',
    },
    permission: {
      list: 'Permission List',
      create: 'Add Permission',
      edit: 'Edit Permission',
      delete: 'Delete Permission',
      view: 'View Permission',
    },
    file: {
      list: 'File List',
      upload: 'Upload File',
      check: 'Check File',
      'upload-progress': 'Upload Progress',
      delete: 'Delete File',
      view: 'View File',
      download: 'Download File',
    },
    audit: {
      view: 'View Audit Log',
    },
    'audit-log': {
      view: 'View Audit Log',
    },
    task: {
      list: 'Task List',
      create: 'Add Task',
      cancel: 'Cancel Task',
      view: 'View Task',
      update: 'Edit Task',
      delete: 'Delete Task',
    },
    'brand-config': {
      view: 'View brand configuration',
      edit: 'Edit brand configuration',
      'upload-logo': 'Upload brand logo',
    },
    'mail-config': {
      view: 'View Mail Config',
      edit: 'Edit Mail Config',
      'test-smtp': 'Test SMTP',
      'test-imap': 'Test IMAP',
      'sync-imap': 'Sync Feedback Emails',
      update: 'Edit Mail Config',
      test: 'Test Mail Config',
      sync: 'Sync Mail',
    },
    feedback: {
      view: 'View Feedback',
      'update-status': 'Update Feedback Status',
      update: 'Edit Feedback',
    },
  },
} as const;

export default permissions;
