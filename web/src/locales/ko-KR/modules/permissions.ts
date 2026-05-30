export const permissions = {
  title: '권한 관리',
  button: {
    create: '권한 추가',
  },
  actions: {
    edit: '수정',
    delete: '삭제',
  },
  table: {
    totalCount: '총 {{count}}개 항목',
    columns: {
      name: '권한명',
      code: '권한 코드',
      type: '유형',
    },
  },
  form: {
    name: {
      label: '권한명',
      placeholder: '권한명을 입력하세요',
    },
    code: {
      label: '권한 코드',
      placeholder: '예: user:create',
    },
    type: {
      label: '유형',
      menu: '메뉴',
      button: '버튼',
    },
    parentId: {
      label: '상위 권한',
      placeholder: '상위 권한을 선택하세요',
      none: '없음 (최상위)',
    },
  },
  modal: {
    createTitle: '권한 추가',
    editTitle: '권한 수정',
    deleteTitle: '권한 삭제',
  },
  message: {
    createSuccess: '성공적으로 생성되었습니다',
    createFailed: '생성에 실패했습니다',
    updateSuccess: '성공적으로 수정되었습니다',
    updateFailed: '수정에 실패했습니다',
    deleteSuccess: '성공적으로 삭제되었습니다',
    deleteFailed: '삭제에 실패했습니다',
    deleteConfirm: '이 권한과 모든 하위 권한을 삭제하시겠습니까?',
    emptyData: '권한 데이터가 없습니다',
  },
  codes: {
    dashboard: {
      view: '대시보드 조회',
    },
    user: {
      list: '사용자 목록',
      create: '사용자 생성',
      edit: '사용자 수정',
      delete: '사용자 삭제',
      view: '사용자 상세 조회',
      'reset-password': '비밀번호 재설정',
      'upload-avatar': '아바타 업로드',
    },
    role: {
      list: '역할 목록',
      create: '역할 생성',
      edit: '역할 수정',
      delete: '역할 삭제',
      'assign-permissions': '권한 할당',
      'view-permissions': '역할 권한 조회',
      view: '역할 상세 조회',
    },
    permission: {
      list: '권한 목록',
      create: '권한 생성',
      edit: '권한 수정',
      delete: '권한 삭제',
      view: '권한 조회',
    },
    file: {
      list: '파일 목록',
      upload: '파일 업로드',
      check: '파일 확인',
      'upload-progress': '업로드 진행률',
      delete: '파일 삭제',
      view: '파일 조회',
      download: '파일 다운로드',
    },
    audit: {
      view: '감사 로그 조회',
    },
    'audit-log': {
      view: '감사 로그 조회',
    },
    task: {
      list: '작업 목록',
      create: '작업 생성',
      cancel: '작업 취소',
      view: '작업 조회',
      update: '작업 수정',
      delete: '작업 삭제',
    },
    'brand-config': {
      view: '브랜드 설정 보기',
      edit: '브랜드 설정 편집',
      'upload-logo': '브랜드 Logo 업로드',
    },
    'mail-config': {
      view: '메일 설정 조회',
      edit: '메일 설정 수정',
      'test-smtp': 'SMTP 테스트',
      'test-imap': 'IMAP 테스트',
      'sync-imap': '피드백 메일 동기화',
      update: '메일 설정 수정',
      test: '메일 설정 테스트',
      sync: '메일 동기화',
    },
    feedback: {
      view: '피드백 조회',
      'update-status': '피드백 상태 업데이트',
      update: '피드백 수정',
    },
  },
} as const;

export default permissions;
