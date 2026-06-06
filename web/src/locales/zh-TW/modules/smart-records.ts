export const smartRecords = {
  title: '告警記錄',
  fields: {
    recordId: '記錄ID',
    recordType: '記錄類型',
    deviceId: '設備ID',
    deviceName: '設備名稱',
    taskName: '任務名稱',
    alarmType: '告警類型',
    alarmLevel: '告警級別',
    snapshotImageUrl: '截圖',
    confidence: '置信度',
    createdAt: '創建時間',
    rawResult: '原始數據',
  },
  type: {
    alarm: '告警',
    recognition: '識別',
    capture: '抓拍',
  },
  actions: {
    export: '導出 CSV',
    viewDetail: '查看詳情',
  },
  filters: {
    recordType: '記錄類型',
    device: '設備',
    alarmType: '告警類型',
    alarmLevel: '告警級別',
    timeRange: '時間範圍',
  },
};
