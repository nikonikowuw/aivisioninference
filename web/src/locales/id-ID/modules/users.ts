export const users = {
  title: "Manajemen Pengguna",
  table: {
    columns: {
      username: "Nama Pengguna",
      displayName: "Nama Tampilan",
      email: "Email",
      roles: "Peran",
    },
  },
  form: {
    username: {
      label: "Nama Pengguna",
      placeholder: "Masukkan nama pengguna",
    },
    displayName: {
      label: "Nama Tampilan",
      placeholder: "Masukkan nama tampilan",
    },
    email: {
      label: "Email",
      placeholder: "Masukkan email",
    },
    password: {
      label: "Kata Sandi",
      placeholder: "Masukkan kata sandi",
      hint: "Biarkan kosong untuk tetap tidak berubah",
    },
    status: {
      label: "Status",
      active: "Aktif",
      inactive: "Tidak Aktif",
      readOnlyHint:
        "Gunakan sakelar status di daftar untuk mengaktifkan atau menonaktifkan pengguna saat mengedit.",
    },
  },
  modal: {
    createTitle: "Tambah Pengguna",
    editTitle: "Edit Pengguna",
  },
  message: {
    resetPasswordSuccess: "Kata sandi berhasil direset",
    deleteConfirm: "Apakah Anda yakin ingin menghapus pengguna ini?",
    disableConfirm: "Apakah Anda yakin ingin menonaktifkan pengguna ini?",
    enableConfirm: "Apakah Anda yakin ingin mengaktifkan pengguna ini?",
    batchDeleteConfirm: "Hapus {{count}} pengguna yang dipilih?",
    batchEnableConfirm: "Aktifkan {{count}} pengguna yang dipilih?",
    batchDisableConfirm: "Nonaktifkan {{count}} pengguna yang dipilih?",
  },
  batch: {
    selected: "{{count}} pengguna dipilih",
    selectPage: "Pilih pengguna di halaman ini",
    selectRow: "Pilih pengguna {{username}}",
    delete: "Hapus batch",
    enable: "Aktifkan batch",
    disable: "Nonaktifkan batch",
  },
} as const;

export default users;
