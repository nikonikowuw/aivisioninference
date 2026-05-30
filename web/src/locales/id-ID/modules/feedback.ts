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
    refresh: "Perbarui",
  },
  message: {
    updated: "Status umpan balik diperbarui",
  },
} as const;

export default feedback;
