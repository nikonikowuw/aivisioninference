export const smartRecords = {
  title: '告警记录',
  fields: {
    recordId: '记录ID',
    recordType: '记录类型',
    deviceId: '设备ID',
    deviceName: '设备名称',
    taskName: '任务名称',
    alarmType: '告警类型',
    alarmLevel: '告警级别',
    snapshotImageUrl: '截图',
    confidence: '置信度',
    createdAt: '创建时间',
    rawResult: '原始数据',
  },
  type: {
    alarm: '告警',
    recognition: '识别',
    capture: '抓拍',
  },
  actions: {
    export: '导出 CSV',
    viewDetail: '查看详情',
  },
  filters: {
    recordType: '记录类型',
    device: '设备',
    alarmType: '告警类型',
    alarmLevel: '告警级别',
    timeRange: '时间范围',
  },
};
