export const files = {
  title: "Manajemen File",
  filter: {
    storageTypes: {
      local: "Lokal",
      oss: "OSS",
      pg: "PostgreSQL",
    },
  },
  table: {
    columns: {
      name: "Nama File",
      type: "Tipe",
      size: "Ukuran",
      storageType: "Tipe Penyimpanan",
      uploadTime: "Waktu Unggah",
    },
  },
  upload: {
    uploading: "Mengunggah...",
    progress: "Progres Unggah",
  },
  message: {
    deleteConfirm: "Apakah Anda yakin ingin menghapus file ini?",
    exportFailed: "Ekspor gagal",
  },
  actions: {
    export: "Ekspor",
  },
  size: {
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    gigabytes: "GB",
  },
} as const;

export default files;
