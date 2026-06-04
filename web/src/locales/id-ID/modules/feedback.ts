export const feedback = {
  title: "Umpan Balik Pengguna",
  source: {
    user: "Dalam aplikasi",
    email: "Email",
  },
  status: {
    open: "Terbuka",
    processing: "Diproses",
    resolved: "Selesai",
    closed: "Ditutup",
  },
  table: {
    source: "Sumber",
    category: "Kategori",
    title: "Judul",
    content: "Konten",
    status: "Status",
    createdAt: "Dibuat Pada",
    actions: "Tindakan",
  },
  actions: {
    batchUpdateStatus: "Perbarui Status Massal",
    viewDetails: "Lihat Detail",
    copy: "Salin",
  },
  detail: {
    title: "Detail Umpan Balik",
    email: "Email Kontak",
    updatedAt: "Diperbarui Pada",
    noEmail: "Email Tidak Tersedia",
    copySuccess: "Alamat email disalin ke papan klip",
    copyFailed: "Gagal menyalin alamat email",
  },
  batch: {
    selected: "{{count}} dipilih",
  },
  message: {
    updated: "Status umpan balik diperbarui",
    batchDone: "Pembaruan massal selesai: {{success}} berhasil, {{failed}} gagal",
    batchUpdateConfirm: "Perbarui status {{count}} umpan balik yang dipilih?",
    operationFailed: "Operasi gagal",
  },
} as const;

export default feedback;
