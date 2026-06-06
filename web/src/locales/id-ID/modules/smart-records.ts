export const smartRecords = {
  title: 'Catatan Alarm',
  fields: {
    recordId: 'ID Catatan',
    recordType: 'Jenis Catatan',
    deviceId: 'ID Perangkat',
    deviceName: 'Nama Perangkat',
    taskName: 'Nama Tugas',
    alarmType: 'Jenis Alarm',
    alarmLevel: 'Tingkat Alarm',
    snapshotImageUrl: 'Snapshot',
    confidence: 'Kepercayaan',
    createdAt: 'Waktu Dibuat',
    rawResult: 'Data Mentah',
  },
  type: {
    alarm: 'Alarm',
    recognition: 'Pengenalan',
    capture: 'Tangkapan',
  },
  actions: {
    export: 'Ekspor CSV',
    viewDetail: 'Lihat Detail',
  },
  filters: {
    recordType: 'Jenis Catatan',
    device: 'Perangkat',
    alarmType: 'Jenis Alarm',
    alarmLevel: 'Tingkat Alarm',
    timeRange: 'Rentang Waktu',
  },
};
