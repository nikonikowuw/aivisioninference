export const users = {
  title: "사용자 관리",
  table: {
    columns: {
      username: "사용자 이름",
      displayName: "표시 이름",
      email: "이메일",
      roles: "역할",
    },
  },
  form: {
    username: {
      label: "사용자 이름",
      placeholder: "사용자 이름을 입력하세요",
    },
    displayName: {
      label: "표시 이름",
      placeholder: "표시 이름을 입력하세요",
    },
    email: {
      label: "이메일",
      placeholder: "이메일을 입력하세요",
    },
    password: {
      label: "비밀번호",
      placeholder: "비밀번호를 입력하세요",
      hint: "변경하지 않으려면 비워 두세요",
    },
    status: {
      label: "상태",
      active: "활성",
      inactive: "비활성",
      readOnlyHint:
        "편집 중에 사용자를 활성화하거나 비활성화하려면 목록의 상태 스위치를 사용하십시오.",
    },
  },
  modal: {
    createTitle: "사용자 추가",
    editTitle: "사용자 편집",
  },
  message: {
    resetPasswordSuccess: "비밀번호가 성공적으로 초기화되었습니다",
    deleteConfirm: "이 사용자를 삭제하시겠습니까?",
    disableConfirm: "이 사용자를 비활성화하시겠습니까?",
    enableConfirm: "이 사용자를 활성화하시겠습니까?",
    batchDeleteConfirm: "선택한 사용자 {{count}}명을 삭제하시겠습니까?",
    batchEnableConfirm: "선택한 사용자 {{count}}명을 활성화하시겠습니까?",
    batchDisableConfirm: "선택한 사용자 {{count}}명을 비활성화하시겠습니까?",
  },
  actions: {
    create: '사용자 추가',
    edit: '편집',
    delete: '삭제',
    enable: '활성화',
    disable: '비활성화',
  },
  batch: {
    selected: "사용자 {{count}}명 선택됨",
    selectPage: "현재 페이지 사용자 선택",
    selectRow: "사용자 {{username}} 선택",
    delete: "일괄 삭제",
    enable: "일괄 활성화",
    disable: "일괄 비활성화",
  },
} as const;

export default users;
