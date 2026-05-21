export const permissions = {
  title: 'Manajemen Izin',
  button: {
    create: 'Tambah Izin',
  },
  table: {
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
      placeholder: 'misal, user:create',
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
} as const;

export default permissions;
