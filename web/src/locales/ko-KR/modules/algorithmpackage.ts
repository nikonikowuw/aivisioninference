export const algorithmpackage = {
  "title": "알고리즘 패키지",
  "uploadZone": {
    "title": "클릭하거나 ZIP 알고리즘 패키지를 이 영역으로 끌어다 놓아 업로드",
    "hint": "최대 1GB 패키지를 지원합니다. 시스템이 자동으로 재패키징(ZIP→TAR)하고 자체 점검을 실행합니다.",
    "onlyZip": "ZIP 파일만 지원됩니다.",
    "dropHere": "여기에 놓기..."
  },
  "uploading": "업로드 중...",
  "checking": "자체 점검 중...",
  "card": {
    "version": "버전",
    "domain": "도메인",
    "hardware": "하드웨어 플랫폼",
    "checkStatus": "자체 점검 상태",
    "packageSize": "패키지 크기",
    "md5": "MD5 체크섬",
    "capabilities": "기능",
    "details": "세부 정보",
    "delete": "패키지 삭제",
    "deleteConfirm": "이 알고리즘 패키지를 삭제하시겠습니까? 이 작업은 파일을 영구적으로 제거하며 되돌릴 수 없습니다.",
    "selfCheckBtn": "자체 점검 실행",
    "selfCheckSuccess": "자체 점검 통과",
    "selfCheckFailed": "자체 점검 실패",
    "selfCheckRunning": "자체 점검 중",
    "selfCheckPending": "자체 점검 대기 중",
    "schema": "결과 스키마",
    "paramsSchema": "매개변수 스키마",
    "copied": "클립보드에 복사되었습니다",
    "noSchema": "제공된 스키마가 없습니다",
    "errorMsg": "자체 점검 오류",
    "empty": "아직 알고리즘 패키지가 없습니다. 위에서 업로드하세요."
  },
  "message": {
    "uploadSuccess": "알고리즘 패키지 업로드 성공, 자체 점검이 트리거되었습니다",
    "uploadFailed": "패키지 업로드 실패",
    "deleteSuccess": "알고리즘 패키지 삭제 성공",
    "deleteFailed": "패키지 삭제 실패",
    "selfCheckTriggered": "자체 점검 명령이 전송되었습니다",
    "selfCheckFailed": "자체 점검 명령 전송 실패",
    "operationFailed": "작업 실패"
  },
  "search": {
    "placeholder": "알고리즘 이름, 별칭 또는 도메인 검색..."
  },
  "stats": {
    "total": "전체 패키지",
    "passed": "자체 점검 통과",
    "failed": "자체 점검 실패",
    "active": "활성"
  }
} as const;

export default algorithmpackage;
