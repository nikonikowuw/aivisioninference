export const roles = {
  title: "Role Management",
  button: {
    assignPermissions: "Assign Permissions",
  },
  table: {
    columns: {
      name: "Role Name",
      description: "Description",
      permissionCount: "Permissions",
    },
  },
  form: {
    name: {
      label: "Role Name",
      placeholder: "Enter role name",
    },
    description: {
      label: "Description",
      placeholder: "Enter description",
    },
    level: {
      label: "Level",
      placeholder: "Lower number = higher authority",
      helper:
        "Lower numbers grant higher authority; final permission checks are enforced by the backend.",
    },
  },
  modal: {
    createTitle: "Add Role",
    editTitle: "Edit Role",
    assignPermissionsTitle: "Assign Permissions",
  },
  permissions: {
    empty: "No permission data",
  },
  message: {
    assignPermissionsSuccess: "Permissions assigned successfully",
    loadPermissionsFailed: "Failed to load permissions",
    assignPermissionsFailed: "Failed to assign permissions",
    deleteConfirm: "Are you sure you want to delete this role?",
    levelInvalidTitle: "Invalid level input",
    levelInvalidDescription:
      "Please enter an integer between {{min}} and {{max}}",
    batchDeleteConfirm: "Delete the selected {{count}} role(s)?",
  },
  batch: {
    selected: "{{count}} role(s) selected",
    delete: "Batch Delete",
  },
} as const;

export default roles;
