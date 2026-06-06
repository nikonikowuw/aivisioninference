export const smartRecords = {
  title: "Rekaman Cerdas",
  subtitle: "Pencarian terpadu rekaman pengenalan, alarm, dan tangkapan",
  tabs: {
    recognition: "Rekaman Pengenalan",
    alarm: "Rekaman Alarm",
    capture: "Rekaman Tangkapan",
  },
  filters: {
    deviceName: "Perangkat",
    devicePlaceholder: "Pilih perangkat",
    allDevices: "Semua Perangkat",
    taskName: "Tugas",
    taskPlaceholder: "Pilih tugas",
    allTasks: "Semua Tugas",
    alarmType: "Jenis Alarm",
    alarmLevel: "Level Alarm",
    categoryCode: "Kategori",
    allCategories: "Semua Kategori",
    minConfidence: "Kepercayaan Minimum",
    minSimilarity: "Kesamaan Minimum",
  },
  table: {
    snapshot: "Gambar Tangkapan",
    target: "Gambar Target",
    deviceName: "Nama Perangkat",
    captureTime: "Waktu Tangkapan",
    person: "Nama Orang",
    personImage: "Gambar Wajah Terdaftar",
    similarity: "Kesamaan",
    alarmType: "Jenis Alarm",
    alarmLevel: "Level Alarm",
    status: "Status Penanganan",
    category: "Kategori",
    confidence: "Kepercayaan",
    taskName: "Nama Tugas",
  },
  alarmLevel: {
    critical: "Kritis",
    high: "Tinggi",
    medium: "Sedang",
    low: "Rendah",
  },
  alarmType: {
    intrusion: "Intrusi Wilayah",
    cross_line: "Alarm Garis Batas",
    region: "Alarm Wilayah",
    unknown: "Tidak Diketahui",
  },
  status: {
    unhandled: "Belum Ditangani",
  },
  actions: {
    batchDelete: "Hapus Massal",
  },
  message: {
    batchDeleteConfirm: "Yakin ingin menghapus {{count}} rekaman yang dipilih?",
  },
  empty: {
    noData: "Tidak ada rekaman cerdas",
    noPermission: "Tidak memiliki izin melihat rekaman cerdas",
  },
} as const;

export default smartRecords;
