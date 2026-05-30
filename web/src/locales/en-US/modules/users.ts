export const users = {
  title: 'User Management',
  button: {
    create: 'Add User',
  },
  table: {
    columns: {
      id: 'ID',
      username: 'Username',
      displayName: 'Display Name',
      email: 'Email',
      status: 'Status',
      roles: 'Roles',
      actions: 'Actions',
    },
    status: {
      active: 'Active',
      inactive: 'Inactive',
    },
  },
  form: {
    username: {
      label: 'Username',
      placeholder: 'Enter username',
    },
    displayName: {
      label: 'Display Name',
      placeholder: 'Enter display name',
    },
    email: {
      label: 'Email',
      placeholder: 'Enter email',
    },
    password: {
      label: 'Password',
      placeholder: 'Enter password',
      hint: 'Leave blank to keep unchanged',
    },
    status: {
      label: 'Status',
      active: 'Active',
      inactive: 'Inactive',
      readOnlyHint: 'Use the status switch in the list to enable or disable users while editing.',
    },
  },
  modal: {
    createTitle: 'Add User',
    editTitle: 'Edit User',
  },
  message: {
    resetPasswordSuccess: 'Password reset successfully',
    deleteConfirm: 'Are you sure you want to delete this user?',
    disableConfirm: 'Are you sure you want to disable this user?',
    enableConfirm: 'Are you sure you want to enable this user?',
  },
  batch: {
    selected: '{{count}} user(s) selected',
    selectPage: 'Select users on this page',
    selectRow: 'Select user {{username}}',
    delete: 'Batch delete',
    enable: 'Batch enable',
    disable: 'Batch disable',
  },
  actions: {
    edit: 'Edit',
    delete: 'Delete',
    enable: 'Enable',
    disable: 'Disable',
  },
} as const;

export default users;
