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
    },
  },
  modal: {
    createTitle: 'Add User',
    editTitle: 'Edit User',
  },
  message: {
    createSuccess: 'User created successfully',
    updateSuccess: 'User updated successfully',
    deleteSuccess: 'User deleted successfully',
    deleteConfirm: 'Are you sure you want to delete this user?',
    operationFailed: 'Operation failed',
    deleteFailed: 'Delete failed',
  },
  actions: {
    edit: 'Edit',
    delete: 'Delete',
  },
} as const;

export default users;
