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
      status: 'Status',
      actions: 'Actions',
    },
    status: {
      active: 'Active',
      inactive: 'Inactive',
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
      helper: 'Lower numbers grant higher authority; final permission checks are enforced by the backend.',
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
    batchDone: 'Batch completed: {{success}} succeeded, {{failed}} failed',
    batchDeleteConfirm: 'Delete the selected {{count}} role(s)?',
  },
  batch: {
    selected: '{{count}} items selected',
    delete: 'Batch Delete',
  },
  actions: {
    edit: 'Edit',
    delete: 'Delete',
  },
} as const;

export default roles;
