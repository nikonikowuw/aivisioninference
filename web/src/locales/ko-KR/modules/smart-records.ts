export const smartRecords = {
  title: '알람 기록',
  fields: {
    recordId: '기록 ID',
    recordType: '기록 유형',
    deviceId: '장치 ID',
    deviceName: '장치 이름',
    taskName: '작업 이름',
    alarmType: '알람 유형',
    alarmLevel: '알람 레벨',
    snapshotImageUrl: '스냅샷',
    confidence: '신뢰도',
    createdAt: '생성 시간',
    rawResult: '원시 데이터',
  },
  type: {
    alarm: '알람',
    recognition: '인식',
    capture: '캡처',
  },
  actions: {
    export: 'CSV 내보내기',
    viewDetail: '상세 보기',
  },
  filters: {
    recordType: '기록 유형',
    device: '장치',
    alarmType: '알람 유형',
    alarmLevel: '알람 레벨',
    timeRange: '시간 범위',
  },
};
