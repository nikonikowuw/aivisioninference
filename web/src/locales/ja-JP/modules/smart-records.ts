export const smartRecords = {
  title: 'アラーム記録',
  fields: {
    recordId: '記録ID',
    recordType: '記録タイプ',
    deviceId: 'デバイスID',
    deviceName: 'デバイス名',
    taskName: 'タスク名',
    alarmType: 'アラームタイプ',
    alarmLevel: 'アラームレベル',
    snapshotImageUrl: 'スナップショット',
    confidence: '信頼度',
    createdAt: '作成日時',
    rawResult: '生データ',
  },
  type: {
    alarm: 'アラーム',
    recognition: '認識',
    capture: 'キャプチャ',
  },
  actions: {
    export: 'CSVエクスポート',
    viewDetail: '詳細表示',
  },
  filters: {
    recordType: '記録タイプ',
    device: 'デバイス',
    alarmType: 'アラームタイプ',
    alarmLevel: 'アラームレベル',
    timeRange: '時間範囲',
  },
};
