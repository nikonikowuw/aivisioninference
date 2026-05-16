export const permissions = {
  title: 'Permission Management',
  button: {
    create: 'Add Permission',
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
      api: 'API',
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
} as const;

export default permissions;
