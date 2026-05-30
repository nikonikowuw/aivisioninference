export const permissions = {
  title: 'Manajemen Izin',
  button: {
    create: 'Tambah Izin',
  },
  actions: {
    edit: 'Edit',
    delete: 'Hapus',
  },
  table: {
    totalCount: '{{count}} item',
    columns: {
      name: 'Nama Izin',
      code: 'Kode Izin',
      type: 'Tipe',
    },
  },
  form: {
    name: {
      label: 'Nama Izin',
      placeholder: 'Masukkan nama izin',
    },
    code: {
      label: 'Kode Izin',
      placeholder: 'contoh: user:create',
    },
    type: {
      label: 'Tipe',
      menu: 'Menu',
      button: 'Tombol',
    },
    parentId: {
      label: 'Izin Induk',
      placeholder: 'Pilih izin induk',
      none: 'Tidak ada (tingkat atas)',
    },
  },
  modal: {
    createTitle: 'Tambah Izin',
    editTitle: 'Edit Izin',
    deleteTitle: 'Hapus Izin',
  },
  message: {
    createSuccess: 'Izin berhasil dibuat',
    createFailed: 'Gagal membuat izin',
    updateSuccess: 'Izin berhasil diperbarui',
    updateFailed: 'Gagal memperbarui izin',
    deleteSuccess: 'Izin berhasil dihapus',
    deleteFailed: 'Gagal menghapus izin',
    deleteConfirm: 'Apakah Anda yakin ingin menghapus izin ini dan semua anaknya?',
    emptyData: 'Tidak ada data izin',
  },
  codes: {
    dashboard: {
      view: 'Lihat Dasbor',
    },
    user: {
      list: 'Daftar Pengguna',
      create: 'Tambah Pengguna',
      edit: 'Edit Pengguna',
      delete: 'Hapus Pengguna',
      view: 'Lihat Pengguna',
      'reset-password': 'Reset Kata Sandi',
      'upload-avatar': 'Unggah Avatar',
    },
    role: {
      list: 'Daftar Peran',
      create: 'Tambah Peran',
      edit: 'Edit Peran',
      delete: 'Hapus Peran',
      'assign-permissions': 'Tetapkan Izin',
      'view-permissions': 'Lihat Izin Peran',
      view: 'Lihat Peran',
    },
    permission: {
      list: 'Daftar Izin',
      create: 'Tambah Izin',
      edit: 'Edit Izin',
      delete: 'Hapus Izin',
      view: 'Lihat Izin',
    },
    file: {
      list: 'Daftar File',
      upload: 'Unggah File',
      check: 'Periksa File',
      'upload-progress': 'Progres Unggahan',
      delete: 'Hapus File',
      view: 'Lihat File',
      download: 'Unduh File',
    },
    audit: {
      view: 'Lihat Log Audit',
    },
    'audit-log': {
      view: 'Lihat Log Audit',
    },
    task: {
      list: 'Daftar Tugas',
      create: 'Tambah Tugas',
      cancel: 'Batalkan Tugas',
      view: 'Lihat Tugas',
      update: 'Edit Tugas',
      delete: 'Hapus Tugas',
    },
    'brand-config': {
      view: 'Lihat konfigurasi merek',
      edit: 'Edit konfigurasi merek',
      'upload-logo': 'Unggah logo merek',
    },
    'mail-config': {
      view: 'Lihat Konfigurasi Email',
      edit: 'Edit Konfigurasi Email',
      'test-smtp': 'Uji SMTP',
      'test-imap': 'Uji IMAP',
      'sync-imap': 'Sinkronisasi Email Umpan Balik',
      update: 'Edit Konfigurasi Email',
      test: 'Uji Konfigurasi Email',
      sync: 'Sinkronisasi Email',
    },
    feedback: {
      view: 'Lihat Umpan Balik',
      'update-status': 'Perbarui Status Umpan Balik',
      update: 'Edit Umpan Balik',
    },
  },
} as const;

export default permissions;
