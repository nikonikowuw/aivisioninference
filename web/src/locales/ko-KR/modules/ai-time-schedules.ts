export const aiTimeSchedules = {
  title: '시간 설정',
  fields: {
    name: '설정 이름',
    description: '설명',
    dateRange: '유효 기간',
    timeWindows: '시간 윈도우',
    updatedAt: '업데이트 시간',
  },
  actions: {
    create: '새 설정',
    edit: '편집',
    delete: '삭제',
    save: '저장',
  },
  form: {
    namePlaceholder: '예: 평일 주간, 24시간 모니터링',
    descriptionPlaceholder: '선택사항, 용도 설명',
    startDate: '시작일',
    endDate: '종료일',
    timeWindows: '매일 시간 윈도우',
    addTimeWindow: '시간대 추가',
  },
  message: {
    deleteConfirm: '이 시간 설정을 삭제하시겠습니까? 참조하는 태스크에는 영향을 주지 않습니다.',
    deleteSuccess: '삭제 완료',
    deleteFailed: '삭제 실패',
    updateSuccess: '업데이트 완료',
    createSuccess: '생성 완료',
    nameRequired: '설정 이름을 입력하세요',
    startDateRequired: '시작일을 선택하세요',
    endDateRequired: '종료일을 선택하세요',
    dateInvalid: '종료일은 시작일보다 이전일 수 없습니다',
  },
  empty: '시간 설정이 없습니다',
} as const;
export default aiTimeSchedules;
