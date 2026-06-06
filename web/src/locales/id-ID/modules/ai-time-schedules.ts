export const aiTimeSchedules = {
  title: 'Jadwal Waktu',
  fields: {
    name: 'Nama Jadwal',
    description: 'Deskripsi',
    dateRange: 'Periode Berlaku',
    timeWindows: 'Jendela Waktu',
    updatedAt: 'Diperbarui',
  },
  actions: {
    create: 'Jadwal Baru',
    edit: 'Edit',
    delete: 'Hapus',
    save: 'Simpan',
  },
  form: {
    namePlaceholder: 'contoh: Jam kerja, Pemantauan 24/7',
    descriptionPlaceholder: 'Opsional, jelaskan tujuan',
    startDate: 'Tanggal Mulai',
    endDate: 'Tanggal Berakhir',
    timeWindows: 'Jendela Waktu Harian',
    addTimeWindow: 'Tambah Jendela Waktu',
  },
  message: {
    deleteConfirm: 'Yakin ingin menghapus jadwal waktu ini? Tugas yang merujuknya tidak akan terpengaruh.',
    deleteSuccess: 'Berhasil dihapus',
    deleteFailed: 'Gagal menghapus',
    updateSuccess: 'Berhasil diperbarui',
    createSuccess: 'Berhasil dibuat',
    nameRequired: 'Masukkan nama jadwal',
    startDateRequired: 'Pilih tanggal mulai',
    endDateRequired: 'Pilih tanggal berakhir',
    dateInvalid: 'Tanggal berakhir tidak boleh sebelum tanggal mulai',
  },
  empty: 'Belum ada jadwal waktu',
} as const;
export default aiTimeSchedules;
