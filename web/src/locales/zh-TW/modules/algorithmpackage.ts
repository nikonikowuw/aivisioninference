export const algorithmpackage = {
  "title": "演算法套件管理",
  "uploadZone": {
    "title": "點擊或拖拽 ZIP 演算法套件到此區域上傳",
    "hint": "支援 1GB 以內的演算法套件，系統會自動進行安全重打包（ZIP -> TAR）並執行自我檢測流程",
    "onlyZip": "僅支援 ZIP 格式的演算法套件",
    "dropHere": "拖曳到此處..."
  },
  "uploading": "正在上傳...",
  "checking": "正在自我檢測...",
  "card": {
    "version": "版本",
    "domain": "業務領域",
    "hardware": "硬體平台",
    "checkStatus": "自我檢測狀態",
    "packageSize": "套件大小",
    "md5": "MD5 校驗和",
    "capabilities": "支援能力",
    "details": "詳情展開",
    "delete": "刪除演算法套件",
    "deleteConfirm": "確定要刪除該演算法套件嗎？此操作將永久移除相關檔案和資料，且不可復原。",
    "selfCheckBtn": "重新檢測",
    "selfCheckSuccess": "檢測成功",
    "selfCheckFailed": "檢測失敗",
    "selfCheckRunning": "檢測進行中",
    "selfCheckPending": "排隊檢測中",
    "schema": "推理結果 Schema",
    "paramsSchema": "參數配置 Schema",
    "copied": "已複製到剪貼簿",
    "noSchema": "未提供 Schema",
    "errorMsg": "自我檢測錯誤詳情",
    "empty": "暫無演算法套件，請在上方上傳"
  },
  "message": {
    "uploadSuccess": "演算法套件上傳成功，已觸發自我檢測信號",
    "uploadFailed": "演算法套件上傳失敗",
    "deleteSuccess": "演算法套件刪除成功",
    "deleteFailed": "演算法套件刪除失敗",
    "selfCheckTriggered": "已發送自我檢測指令",
    "selfCheckFailed": "發送自我檢測指令失敗",
    "operationFailed": "操作失敗"
  },
  "search": {
    "placeholder": "搜尋演算法套件名稱、別名或業務領域..."
  },
  "stats": {
    "total": "總演算法套件",
    "passed": "檢測通過",
    "failed": "檢測失敗",
    "active": "已啟用"
  }
} as const;

export default algorithmpackage;
