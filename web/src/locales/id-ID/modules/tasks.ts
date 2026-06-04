export const tasks = {
  title: "Manajemen Tugas",
  filter: {
    taskTypes: {
      email: "Email",
      export: "Ekspor",
      import: "Impor",
      backup: "Cadangan",
    },
  },
  table: {
    columns: {
      type: "Tipe",
      error: "Kesalahan",
    },
    status: {
      pending: "Menunggu",
      running: "Berjalan",
      completed: "Selesai",
      failed: "Gagal",
      cancelled: "Dibatalkan",
    },
  },
  message: {
    cancelled: "Tugas dibatalkan",
    cancelFailed: "Gagal membatalkan tugas",
    cancelConfirm: "Apakah Anda yakin ingin membatalkan tugas ini?",
    batchCancelConfirm: "Batalkan {{count}} tugas yang dipilih?",
    confirmCancel: "Ya, batalkan",
  },
  actions: {
    cancel: "Batalkan Tugas",
  },
} as const;

export default tasks;
