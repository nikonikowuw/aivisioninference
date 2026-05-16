export const files = {
  title: '文件管理',
  button: {
    upload: '上传文件',
  },
  table: {
    columns: {
      id: 'ID',
      name: '文件名',
      type: '类型',
      size: '大小',
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
  },
  actions: {
    delete: '删除',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
