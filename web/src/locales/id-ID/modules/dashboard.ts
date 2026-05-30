export const dashboard = {
  title: "Dasbor",
  stats: {
    totalUsers: "Total Pengguna",
    totalFiles: "Total File",
    activeTasks: "Tugas Aktif",
  },
  charts: {
    userStats: "Pertumbuhan & Aktivitas Pengguna",
    userNew: "Pengguna Baru",
    userActive: "Pengguna Aktif",
  },
  auditLog: {
    title: "Aktivitas Terbaru",
    empty: "Belum ada aktivitas terbaru",
    columns: {
      username: "Pengguna",
      action: "Tindakan",
      method: "Metode",
      time: "Waktu",
    },
  },
  message: {
    loadFailed: "Gagal memuat data dasbor",
  },
} as const;

export default dashboard;
