export const roles = {
  title: '역할 관리',
  button: {
    create: '역할 추가',
    assignPermissions: '권한 할당',
  },
  table: {
    columns: {
      id: 'ID',
      name: '역할 이름',
      description: '설명',
      permissionCount: '권한',
      status: '상태',
      actions: '작업',
    },
    status: {
      active: '활성',
      inactive: '비활성',
    },
  },
  form: {
    name: {
      label: '역할 이름',
      placeholder: '역할 이름을 입력하세요',
    },
    description: {
      label: '설명',
      placeholder: '설명을 입력하세요',
    },
    level: {
      label: '레벨',
      placeholder: '낮은 숫자 = 높은 권한',
      helper: '숫자가 낮을수록 권한이 높습니다. 최종 권한 확인은 백엔드에서 수행됩니다.',
    },
  },
  modal: {
    createTitle: '역할 추가',
    editTitle: '역할 편집',
    assignPermissionsTitle: '권한 할당',
  },
  permissions: {
    empty: '권한 데이터 없음',
  },
  message: {
    assignPermissionsSuccess: '권한이 성공적으로 할당되었습니다',
    loadPermissionsFailed: '권한을 불러오지 못했습니다',
    assignPermissionsFailed: '권한 할당 실패',
    deleteConfirm: '이 역할을 삭제하시겠습니까?',
    levelInvalidTitle: '잘못된 레벨 입력',
    levelInvalidDescription: '{{min}}에서 {{max}} 사이의 정수를 입력하세요',
    batchDeleteConfirm: '선택한 역할 {{count}}개를 삭제하시겠습니까?',
  },
  batch: {
    delete: '일괄 삭제',
  },
  actions: {
    edit: '편집',
    delete: '삭제',
  },
} as const;

export default roles;
