export const permissions = {
  title: "權限管理",
  table: {
    totalCount: "共 {{count}} 項",
    columns: {
      name: "權限名",
      code: "權限代碼",
      type: "類型",
    },
  },
  form: {
    name: {
      label: "權限名",
      placeholder: "請輸入權限名",
    },
    code: {
      label: "權限代碼",
      placeholder: "例如：user:create",
    },
    type: {
      label: "類型",
      menu: "選單",
      button: "按鈕",
    },
    parentId: {
      label: "父級權限",
      placeholder: "選擇父級權限",
      none: "無（頂級節點）",
    },
  },
  modal: {
    createTitle: "新增權限",
    editTitle: "編輯權限",
    deleteTitle: "刪除權限",
  },
  message: {
    deleteConfirm: "確定要刪除該權限及其所有子權限嗎？",
    emptyData: "暫無權限數據",
  },
  codes: {
    dashboard: {
      view: "檢視儀表板",
    },
    user: {
      list: "使用者列表",
      create: "新增使用者",
      edit: "編輯使用者",
      delete: "刪除使用者",
      view: "檢視使用者",
      "reset-password": "重置密碼",
      "upload-avatar": "上傳頭像",
      export: "匯出使用者",
      import: "匯入使用者",
      "batch-delete": "批量刪除使用者",
      "batch-status": "批量更新使用者狀態",
    },
    role: {
      list: "角色列表",
      create: "新增角色",
      edit: "編輯角色",
      delete: "刪除角色",
      "assign-permissions": "分配權限",
      "view-permissions": "檢視角色權限",
      view: "檢視角色",
      "batch-delete": "批量刪除角色",
      export: "匯出角色",
    },
    permission: {
      list: "權限列表",
      create: "新增權限",
      edit: "編輯權限",
      delete: "刪除權限",
      view: "檢視權限",
    },
    file: {
      list: "檔案列表",
      upload: "上傳檔案",
      check: "校驗檔案",
      "upload-progress": "上傳進度",
      delete: "刪除檔案",
      view: "檢視檔案",
      download: "下載檔案",
      export: "匯出檔案",
      "batch-delete": "批量刪除檔案",
    },
    audit: {
      view: "檢視稽核日誌",
      export: "匯出稽核日誌",
    },
    "audit-log": {
      view: "檢視稽核日誌",
      export: "匯出稽核日誌",
    },
    task: {
      list: "任務列表",
      create: "新增任務",
      cancel: "取消任務",
      view: "檢視任務",
      update: "編輯任務",
      delete: "刪除任務",
      export: "匯出任務",
      "batch-cancel": "批量取消任務",
    },
    "brand-config": {
      view: "檢視品牌設定",
      edit: "編輯品牌設定",
      "upload-logo": "上傳品牌 Logo",
    },
    "mail-config": {
      view: "檢視郵件設定",
      edit: "編輯郵件設定",
      "test-smtp": "測試SMTP",
      "test-imap": "測試IMAP",
      "sync-imap": "同步回饋郵件",
      update: "編輯郵件設定",
      test: "測試郵件設定",
      sync: "同步郵件",
    },
    feedback: {
      view: "檢視回饋",
      "update-status": "更新回饋狀態",
      update: "編輯回饋",
      export: "匯出回饋",
      "batch-status": "批量更新回饋狀態",
    },
  },
} as const;

export default permissions;
