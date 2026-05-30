export const auditLogs = {
  title: 'Log Audit',
  filter: {
    results: {
      success: 'Berhasil',
      failed: 'Gagal',
    },
    resourceTypes: {
      user: 'Pengguna',
      role: 'Peran',
      permission: 'Izin',
      file: 'File',
      task: 'Tugas',
      system: 'Sistem',
    },
  },
  table: {
    columns: {
      username: 'Operator',
      actionType: 'Tindakan',
      resourceType: 'Tipe Sumber Daya',
      method: 'Metode',
      path: 'Path',
      ip: 'IP',
      status: 'Status',
      duration: 'Durasi',
      result: 'Hasil',
      time: 'Waktu',
    },
    durationMs: '{{value}} ms',
  },
  actionTypes: {
    action: {
      view: {
        users: 'Lihat pengguna',
        roles: 'Lihat peran',
        permissions: 'Lihat izin',
        files: 'Lihat file',
        'audit-logs': 'Lihat log audit',
        tasks: 'Lihat tugas',
        system: 'Lihat konfigurasi sistem',
        dashboard: 'Lihat dasbor',
      },
      create: {
        users: 'Buat pengguna',
        roles: 'Buat peran',
        permissions: 'Buat izin',
        files: 'Unggah file',
        tasks: 'Buat tugas',
      },
      update: {
        users: 'Perbarui pengguna',
        roles: 'Perbarui peran',
        permissions: 'Perbarui izin',
        files: 'Perbarui file',
        tasks: 'Perbarui tugas',
        system: 'Perbarui konfigurasi sistem',
      },
      delete: {
        users: 'Hapus pengguna',
        roles: 'Hapus peran',
        permissions: 'Hapus izin',
        files: 'Hapus file',
        tasks: 'Hapus tugas',
      },
      login: 'Login pengguna',
      auth: 'Operasi autentikasi',
    },
  },
} as const;

export default auditLogs;
