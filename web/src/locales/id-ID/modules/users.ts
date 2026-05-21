export const users = {
  title: 'Manajemen Pengguna',
  button: {
    create: 'Tambah Pengguna',
  },
  table: {
    columns: {
      id: 'ID',
      username: 'Nama Pengguna',
      displayName: 'Nama Tampilan',
      email: 'Email',
      status: 'Status',
      roles: 'Peran',
      actions: 'Tindakan',
    },
    status: {
      active: 'Aktif',
      inactive: 'Tidak Aktif',
    },
  },
  form: {
    username: {
      label: 'Nama Pengguna',
      placeholder: 'Masukkan nama pengguna',
    },
    displayName: {
      label: 'Nama Tampilan',
      placeholder: 'Masukkan nama tampilan',
    },
    email: {
      label: 'Email',
      placeholder: 'Masukkan email',
    },
    password: {
      label: 'Kata Sandi',
      placeholder: 'Masukkan kata sandi',
      hint: 'Biarkan kosong untuk tetap tidak berubah',
    },
    status: {
      label: 'Status',
      active: 'Aktif',
      inactive: 'Tidak Aktif',
      readOnlyHint: 'Gunakan sakelar status di daftar untuk mengaktifkan atau menonaktifkan pengguna saat mengedit.',
    },
  },
  modal: {
    createTitle: 'Tambah Pengguna',
    editTitle: 'Edit Pengguna',
  },
  message: {
    createSuccess: 'Pengguna berhasil dibuat',
    updateSuccess: 'Pengguna berhasil diperbarui',
    deleteSuccess: 'Pengguna berhasil dihapus',
    resetPasswordSuccess: 'Kata sandi berhasil direset',
    deleteConfirm: 'Apakah Anda yakin ingin menghapus pengguna ini?',
    disableConfirm: 'Apakah Anda yakin ingin menonaktifkan pengguna ini?',
    enableConfirm: 'Apakah Anda yakin ingin mengaktifkan pengguna ini?',
    operationFailed: 'Operasi gagal',
    deleteFailed: 'Penghapusan gagal',
  },
  actions: {
    edit: 'Edit',
    delete: 'Hapus',
    enable: 'Aktifkan',
    disable: 'Nonaktifkan',
  },
} as const;

export default users;
