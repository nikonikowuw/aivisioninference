export const files = {
  title: 'File Management',
  button: {
    upload: 'Upload File',
  },
  table: {
    columns: {
      id: 'ID',
      name: 'Filename',
      type: 'Type',
      size: 'Size',
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
  },
  actions: {
    delete: 'Delete',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
