export const files = {
  title: "檔案管理",
  filter: {
    storageTypes: {
      local: "本機",
      oss: "OSS",
      pg: "PostgreSQL",
    },
  },
  table: {
    columns: {
      name: "檔案名稱",
      type: "類型",
      size: "大小",
      storageType: "儲存類型",
      uploadTime: "上傳時間",
    },
  },
  upload: {
    uploading: "上傳中...",
    progress: "上傳進度",
  },
  message: {
    deleteConfirm: "確定刪除該檔案？",
    batchDeleteConfirm: "確定刪除已選擇的 {{count}} 個檔案？",
    exportFailed: "匯出失敗",
  },
  batch: {
    delete: "批次刪除",
  },
  actions: {
    export: "匯出",
  },
  size: {
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    gigabytes: "GB",
  },
} as const;

export default files;
