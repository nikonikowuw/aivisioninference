export const smartRecords = {
  title: 'Alarm Records',
  fields: {
    recordId: 'Record ID',
    recordType: 'Record Type',
    deviceId: 'Device ID',
    deviceName: 'Device Name',
    taskName: 'Task Name',
    alarmType: 'Alarm Type',
    alarmLevel: 'Alarm Level',
    snapshotImageUrl: 'Snapshot',
    confidence: 'Confidence',
    createdAt: 'Created At',
    rawResult: 'Raw Data',
  },
  type: {
    alarm: 'Alarm',
    recognition: 'Recognition',
    capture: 'Capture',
  },
  actions: {
    export: 'Export CSV',
    viewDetail: 'View Detail',
  },
  filters: {
    recordType: 'Record Type',
    device: 'Device',
    alarmType: 'Alarm Type',
    alarmLevel: 'Alarm Level',
    timeRange: 'Time Range',
  },
};
