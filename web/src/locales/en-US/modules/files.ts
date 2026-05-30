export const files = {
  title: "File Management",
  filter: {
    storageTypes: {
      local: "Local",
      oss: "OSS",
      pg: "PostgreSQL",
    },
  },
  table: {
    columns: {
      name: "Filename",
      type: "Type",
      size: "Size",
      storageType: "Storage Type",
      uploadTime: "Upload Time",
    },
  },
  upload: {
    uploading: "Uploading...",
    progress: "Upload Progress",
  },
  message: {
    deleteConfirm: "Are you sure you want to delete this file?",
    batchDeleteConfirm: "Delete the selected {{count}} file(s)?",
  },
  batch: {
    delete: "Batch Delete",
  },
  size: {
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    gigabytes: "GB",
  },
} as const;

export default files;
