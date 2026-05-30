export const auditLogs = {
  title: '감사 로그',
  filter: {
    results: {
      success: '성공',
      failed: '실패',
    },
    resourceTypes: {
      user: '사용자',
      role: '역할',
      permission: '권한',
      file: '파일',
      task: '작업',
      system: '시스템',
      feedback: '피드백',
    },
  },
  table: {
    columns: {
      username: '작업자',
      actionType: '동작',
      resourceType: '리소스 유형',
      method: '메서드',
      path: '경로',
      ip: 'IP',
      status: '상태',
      duration: '소요 시간',
      result: '결과',
      time: '시간',
    },
    durationMs: '{{value}} ms',
  },
  actionTypes: {
    action: {
      view: {
        users: '사용자 조회',
        roles: '역할 조회',
        permissions: '권한 조회',
        files: '파일 조회',
        'audit-logs': '감사 로그 조회',
        tasks: '작업 조회',
        system: '시스템 설정 조회',
        dashboard: '대시보드 조회',
        feedback: '피드백 조회',
      },
      create: {
        users: '사용자 생성',
        roles: '역할 생성',
        permissions: '권한 생성',
        files: '파일 업로드',
        tasks: '작업 생성',
        system: '시스템 설정 생성',
        feedback: '피드백 제출',
      },
      update: {
        users: '사용자 수정',
        roles: '역할 수정',
        permissions: '권한 수정',
        files: '파일 수정',
        tasks: '작업 수정',
        system: '시스템 설정 수정',
        feedback: '피드백 상태 수정',
      },
      delete: {
        users: '사용자 삭제',
        roles: '역할 삭제',
        permissions: '권한 삭제',
        files: '파일 삭제',
        tasks: '작업 삭제',
      },
      login: '사용자 로그인',
      logout: '사용자 로그아웃',
      auth: '인증 작업',
    },
  },
} as const;

export default auditLogs;
