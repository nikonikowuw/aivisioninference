export const permissions = {
  title: "Permission Management",
  table: {
    totalCount: "{{count}} items",
    columns: {
      name: "Permission Name",
      code: "Permission Code",
      type: "Type",
    },
  },
  form: {
    name: {
      label: "Permission Name",
      placeholder: "Enter permission name",
    },
    code: {
      label: "Permission Code",
      placeholder: "e.g., user:create",
    },
    type: {
      label: "Type",
      menu: "Menu",
      button: "Button",
    },
    parentId: {
      label: "Parent Permission",
      placeholder: "Select parent permission",
      none: "None (top level)",
    },
  },
  modal: {
    createTitle: "Add Permission",
    editTitle: "Edit Permission",
    deleteTitle: "Delete Permission",
  },
  message: {
    deleteConfirm:
      "Are you sure you want to delete this permission and all its children?",
    emptyData: "No permission data",
  },
  codes: {
    dashboard: {
      view: "View Dashboard",
    },
    user: {
      list: "User List",
      create: "Add User",
      edit: "Edit User",
      delete: "Delete User",
      view: "View User",
      "reset-password": "Reset Password",
      "upload-avatar": "Upload Avatar",
      export: "Export Users",
      import: "Import Users",
      "batch-delete": "Batch Delete Users",
      "batch-status": "Batch Update User Status",
    },
    role: {
      list: "Role List",
      create: "Add Role",
      edit: "Edit Role",
      delete: "Delete Role",
      "assign-permissions": "Assign Permissions",
      "view-permissions": "View Role Permissions",
      view: "View Role",
      "batch-delete": "Batch Delete Roles",
      export: "Export Roles",
    },
    permission: {
      list: "Permission List",
      create: "Add Permission",
      edit: "Edit Permission",
      delete: "Delete Permission",
      view: "View Permission",
    },
    file: {
      list: "File List",
      upload: "Upload File",
      check: "Check File",
      "upload-progress": "Upload Progress",
      delete: "Delete File",
      view: "View File",
      download: "Download File",
      export: "Export Files",
      "batch-delete": "Batch Delete Files",
    },
    audit: {
      view: "View Audit Log",
      export: "Export Audit Logs",
    },
    "audit-log": {
      view: "View Audit Log",
      export: "Export Audit Logs",
    },
    task: {
      list: "Task List",
      create: "Add Task",
      cancel: "Cancel Task",
      view: "View Task",
      update: "Edit Task",
      delete: "Delete Task",
      export: "Export Tasks",
      "batch-cancel": "Batch Cancel Tasks",
    },
    "brand-config": {
      view: "View brand configuration",
      edit: "Edit brand configuration",
      "upload-logo": "Upload brand logo",
    },
    "mail-config": {
      view: "View Mail Config",
      edit: "Edit Mail Config",
      "test-smtp": "Test SMTP",
      "test-imap": "Test IMAP",
      "sync-imap": "Sync Feedback Emails",
      update: "Edit Mail Config",
      test: "Test Mail Config",
      sync: "Sync Mail",
    },
    feedback: {
      view: "View Feedback",
      "update-status": "Update Feedback Status",
      update: "Edit Feedback",
      export: "Export Feedback",
      "batch-status": "Batch Update Feedback Status",
    },
  },
} as const;

export default permissions;
