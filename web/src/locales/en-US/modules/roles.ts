export const roles = {
  title: 'Role Management',
  button: {
    create: 'Add Role',
    assignPermissions: 'Assign Permissions',
  },
  table: {
    columns: {
      id: 'ID',
      name: 'Role Name',
      description: 'Description',
      permissionCount: 'Permissions',
      actions: 'Actions',
    },
  },
  form: {
    name: {
      label: 'Role Name',
      placeholder: 'Enter role name',
    },
    description: {
      label: 'Description',
      placeholder: 'Enter description',
    },
    level: {
      label: 'Level',
      placeholder: 'Lower number = higher authority',
    },
  },
  modal: {
    createTitle: 'Add Role',
    editTitle: 'Edit Role',
    assignPermissionsTitle: 'Assign Permissions',
  },
  permissions: {
    empty: 'No permission data',
  },
  message: {
    createSuccess: 'Role created successfully',
    updateSuccess: 'Role updated successfully',
    deleteSuccess: 'Role deleted successfully',
    assignPermissionsSuccess: 'Permissions assigned successfully',
    loadPermissionsFailed: 'Failed to load permissions',
    assignPermissionsFailed: 'Failed to assign permissions',
    deleteConfirm: 'Are you sure you want to delete this role?',
    operationFailed: 'Operation failed',
    deleteFailed: 'Delete failed',
    levelInvalidTitle: 'Invalid level input',
    levelInvalidDescription: 'Please enter an integer between {{min}} and {{max}}',
  },
  actions: {
    edit: 'Edit',
    delete: 'Delete',
  },
} as const;

export default roles;
