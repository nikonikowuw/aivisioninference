export const files = {
  title: 'ファイル管理',
  button: {
    upload: 'ファイルをアップロード',
  },
  filter: {
    storageTypes: {
      local: 'ローカル',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: 'ファイル名',
      type: 'タイプ',
      size: 'サイズ',
      storageType: 'ストレージタイプ',
      uploadTime: 'アップロード日時',
      actions: '操作',
    },
  },
  upload: {
    uploading: 'アップロード中...',
    progress: 'アップロード進捗',
  },
  message: {
    uploadSuccess: 'ファイルをアップロードしました',
    uploadFailed: 'アップロードに失敗しました',
    deleteSuccess: 'ファイルを削除しました',
    deleteFailed: '削除に失敗しました',
    deleteConfirm: 'このファイルを削除してもよろしいですか？',
    exportFailed: 'エクスポートに失敗しました',
  },
  actions: {
    delete: '削除',
    download: 'ダウンロード',
    export: 'エクスポート',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
