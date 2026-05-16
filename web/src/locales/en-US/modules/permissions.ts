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
      label: 'Parent ID',
      placeholder: 'Leave blank for top level',
    },
  },
  modal: {
    createTitle: 'Add Permission',
  },
  message: {
    createSuccess: 'Permission created successfully',
    createFailed: 'Failed to create permission',
    emptyData: 'No permission data',
  },
} as const;

export default permissions;
