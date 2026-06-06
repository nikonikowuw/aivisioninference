export const algorithmpackage = {
  "title": "算法包管理",
  "uploadZone": {
    "title": "点击或拖拽 ZIP 算法包到此区域上传",
    "hint": "支持 1GB 以内的算法包，系统会自动进行安全重打包（ZIP -> TAR）并执行自检流程",
    "onlyZip": "仅支持 ZIP 格式的算法包",
    "dropHere": "拖拽到此处..."
  },
  "uploading": "正在上传...",
  "checking": "正在自检...",
  "card": {
    "version": "版本",
    "domain": "业务领域",
    "hardware": "硬件平台",
    "checkStatus": "自检状态",
    "packageSize": "包大小",
    "md5": "MD5 校验和",
    "capabilities": "支持能力",
    "details": "详情展开",
    "delete": "删除算法包",
    "deleteConfirm": "确定要删除该算法包吗？此操作将永久移除相关文件和数据，且不可恢复。",
    "selfCheckBtn": "重新自检",
    "selfCheckSuccess": "自检成功",
    "selfCheckFailed": "自检失败",
    "selfCheckRunning": "自检进行中",
    "selfCheckPending": "排队自检中",
    "schema": "推理结果 Schema",
    "paramsSchema": "入参配置 Schema",
    "copied": "已复制到剪贴板",
    "noSchema": "未提供 Schema",
    "errorMsg": "自检错误详情",
    "empty": "暂无算法包，请在上方上传"
  },
  "message": {
    "uploadSuccess": "算法包上传成功，已触发自检信令",
    "uploadFailed": "算法包上传失败",
    "deleteSuccess": "算法包删除成功",
    "deleteFailed": "算法包删除失败",
    "selfCheckTriggered": "已发送自检指令",
    "selfCheckFailed": "发送自检指令失败",
    "operationFailed": "操作失败"
  },
  "search": {
    "placeholder": "搜索算法包名称、别名或业务领域..."
  },
  "stats": {
    "total": "总算法包",
    "passed": "自检通过",
    "failed": "自检失败",
    "active": "已启用"
  }
} as const;

export default algorithmpackage;
