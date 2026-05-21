export const tasks = {
  title: '작업 관리',
  filter: {
    taskTypes: {
      email: '이메일',
      export: '내보내기',
      import: '가져오기',
      backup: '백업',
    },
  },
  table: {
    columns: {
      id: 'ID',
      type: '유형',
      status: '상태',
      error: '오류',
      createdAt: '생성 시간',
      updatedAt: '수정 시간',
      actions: '작업',
    },
    status: {
      pending: '대기 중',
      running: '실행 중',
      completed: '완료됨',
      failed: '실패함',
      cancelled: '취소됨',
    },
  },
  message: {
    cancelled: '작업이 취소되었습니다',
    cancelFailed: '작업 취소 실패',
    cancelConfirm: '이 작업을 취소하시겠습니까?',
  },
  actions: {
    cancel: '작업 취소',
  },
} as const;

export default tasks;
