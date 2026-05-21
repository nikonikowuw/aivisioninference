export const permissions = {
  title: '권한 관리',
  button: {
    create: '권한 추가',
  },
  table: {
    columns: {
      name: '권한 이름',
      code: '권한 코드',
      type: '유형',
    },
  },
  form: {
    name: {
      label: '권한 이름',
      placeholder: '권한 이름을 입력하세요',
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
      placeholder: '상위 권한 선택',
      none: '없음 (최상위)',
    },
  },
  modal: {
    createTitle: '권한 추가',
    editTitle: '권한 편집',
    deleteTitle: '권한 삭제',
  },
  message: {
    createSuccess: '권한이 성공적으로 생성되었습니다',
    createFailed: '권한 생성 실패',
    updateSuccess: '권한이 성공적으로 업데이트되었습니다',
    updateFailed: '권한 업데이트 실패',
    deleteSuccess: '권한이 성공적으로 삭제되었습니다',
    deleteFailed: '권한 삭제 실패',
    deleteConfirm: '이 권한과 모든 하위 권한을 삭제하시겠습니까?',
    emptyData: '권한 데이터 없음',
  },
} as const;

export default permissions;
