export const tasks = {
  title: 'Manajemen Tugas',
  filter: {
    taskTypes: {
      email: 'Email',
      export: 'Ekspor',
      import: 'Impor',
      backup: 'Cadangan',
    },
  },
  table: {
    columns: {
      id: 'ID',
      type: 'Tipe',
      status: 'Status',
      error: 'Kesalahan',
      createdAt: 'Dibuat Pada',
      updatedAt: 'Diperbarui Pada',
      actions: 'Tindakan',
    },
    status: {
      pending: 'Tertunda',
      running: 'Berjalan',
      completed: 'Selesai',
      failed: 'Gagal',
      cancelled: 'Dibatalkan',
    },
  },
  message: {
    cancelled: 'Tugas dibatalkan',
    cancelFailed: 'Gagal membatalkan tugas',
    cancelConfirm: 'Apakah Anda yakin ingin membatalkan tugas ini?',
  },
  actions: {
    cancel: 'Batalkan Tugas',
  },
} as const;

export default tasks;
