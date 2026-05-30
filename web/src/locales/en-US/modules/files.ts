export const files = {
  title: 'File Management',
  button: {
    upload: 'Upload File',
  },
  filter: {
    storageTypes: {
      local: 'Local',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: 'Filename',
      type: 'Type',
      size: 'Size',
      storageType: 'Storage Type',
      uploadTime: 'Upload Time',
      actions: 'Actions',
    },
  },
  upload: {
    uploading: 'Uploading...',
    progress: 'Upload Progress',
  },
  message: {
    uploadSuccess: 'File uploaded successfully',
    uploadFailed: 'Upload failed',
    deleteSuccess: 'File deleted successfully',
    deleteFailed: 'Delete failed',
    deleteConfirm: 'Are you sure you want to delete this file?',
    exportFailed: 'Export failed',
  },
  actions: {
    delete: 'Delete',
    download: 'Download',
    export: 'Export',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
