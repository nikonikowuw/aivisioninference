export const files = {
  title: '文件管理',
  button: {
    upload: '上传文件',
  },
  filter: {
    storageTypes: {
      local: '本地',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: '文件名',
      type: '类型',
      size: '大小',
      storageType: '存储类型',
      uploadTime: '上传时间',
      actions: '操作',
    },
  },
  upload: {
    uploading: '上传中...',
    progress: '上传进度',
  },
  message: {
    uploadSuccess: '上传成功',
    uploadFailed: '上传失败',
    deleteSuccess: '删除成功',
    deleteFailed: '删除失败',
    deleteConfirm: '确定删除该文件？',
    batchDone: '批量操作完成：成功 {{success}}，失败 {{failed}}',
    batchDeleteConfirm: '确定删除选中的 {{count}} 个文件？',
  },
  batch: {
    selected: '已选中 {{count}} 项',
    delete: '批量删除',
  },
  actions: {
    delete: '删除',
    download: '下载',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
