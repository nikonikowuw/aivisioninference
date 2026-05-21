export const files = {
  title: 'Manajemen File',
  button: {
    upload: 'Unggah File',
  },
  filter: {
    storageTypes: {
      local: 'Lokal',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: 'Nama File',
      type: 'Tipe',
      size: 'Ukuran',
      storageType: 'Tipe Penyimpanan',
      uploadTime: 'Waktu Unggah',
      actions: 'Tindakan',
    },
  },
  upload: {
    uploading: 'Mengunggah...',
    progress: 'Progres Unggah',
  },
  message: {
    uploadSuccess: 'File berhasil diunggah',
    uploadFailed: 'Unggahan gagal',
    deleteSuccess: 'File berhasil dihapus',
    deleteFailed: 'Penghapusan gagal',
    deleteConfirm: 'Apakah Anda yakin ingin menghapus file ini?',
  },
  actions: {
    delete: 'Hapus',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
