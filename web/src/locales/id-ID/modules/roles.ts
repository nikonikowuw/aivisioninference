export const roles = {
  title: 'Manajemen Peran',
  button: {
    create: 'Tambah Peran',
    assignPermissions: 'Tetapkan Izin',
  },
  table: {
    columns: {
      id: 'ID',
      name: 'Nama Peran',
      description: 'Deskripsi',
      permissionCount: 'Izin',
      status: 'Status',
      actions: 'Tindakan',
    },
    status: {
      active: 'Aktif',
      inactive: 'Tidak Aktif',
    },
  },
  form: {
    name: {
      label: 'Nama Peran',
      placeholder: 'Masukkan nama peran',
    },
    description: {
      label: 'Deskripsi',
      placeholder: 'Masukkan deskripsi',
    },
    level: {
      label: 'Tingkat',
      placeholder: 'Angka lebih rendah = otoritas lebih tinggi',
      helper: 'Angka yang lebih rendah memberikan otoritas yang lebih tinggi; pemeriksaan izin akhir ditegakkan oleh backend.',
    },
  },
  modal: {
    createTitle: 'Tambah Peran',
    editTitle: 'Edit Peran',
    assignPermissionsTitle: 'Tetapkan Izin',
  },
  permissions: {
    empty: 'Tidak ada data izin',
  },
  message: {
    assignPermissionsSuccess: 'Izin berhasil ditetapkan',
    loadPermissionsFailed: 'Gagal memuat izin',
    assignPermissionsFailed: 'Gagal menetapkan izin',
    deleteConfirm: 'Apakah Anda yakin ingin menghapus peran ini?',
    levelInvalidTitle: 'Input tingkat tidak valid',
    levelInvalidDescription: 'Silakan masukkan angka bulat antara {{min}} dan {{max}}',
    batchDeleteConfirm: 'Hapus {{count}} peran yang dipilih?',
  },
  batch: {
    delete: 'Hapus massal',
  },
  actions: {
    edit: 'Edit',
    delete: 'Hapus',
  },
} as const;

export default roles;
