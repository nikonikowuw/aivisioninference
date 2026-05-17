export const files = {
  title: '檔案管理',
  button: {
    upload: '上傳檔案',
  },
  filter: {
    storageTypes: {
      local: '本機',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: '檔案名稱',
      type: '類型',
      size: '大小',
      storageType: '儲存類型',
      uploadTime: '上傳時間',
      actions: '操作',
    },
  },
  upload: {
    uploading: '上傳中...',
    progress: '上傳進度',
  },
  message: {
    uploadSuccess: '上傳成功',
    uploadFailed: '上傳失敗',
    deleteSuccess: '刪除成功',
    deleteFailed: '刪除失敗',
    deleteConfirm: '確定刪除該檔案？',
  },
  actions: {
    delete: '刪除',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
