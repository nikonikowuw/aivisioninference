export const files = {
  title: "ファイル管理",
  filter: {
    storageTypes: {
      local: "ローカル",
      oss: "OSS",
      pg: "PostgreSQL",
    },
  },
  table: {
    columns: {
      name: "ファイル名",
      type: "タイプ",
      size: "サイズ",
      storageType: "ストレージタイプ",
      uploadTime: "アップロード日時",
    },
  },
  upload: {
    uploading: "アップロード中...",
    progress: "アップロード進捗",
  },
  message: {
    deleteConfirm: "このファイルを削除してもよろしいですか？",
    exportFailed: "エクスポートに失敗しました",
  },
  actions: {
    export: "エクスポート",
  },
  size: {
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    gigabytes: "GB",
  },
} as const;

export default files;
