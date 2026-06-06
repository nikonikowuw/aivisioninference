export const algorithmpackage = {
  "title": "アルゴリズムパッケージ",
  "uploadZone": {
    "title": "クリックまたはZIPアルゴリズムパッケージをこの領域にドラッグしてアップロード",
    "hint": "1GBまでのパッケージに対応。システムが自動的に再パッケージ（ZIP→TAR）し、セルフチェックを実行します。",
    "onlyZip": "ZIPファイルのみ対応しています。",
    "dropHere": "ここにドロップ..."
  },
  "uploading": "アップロード中...",
  "checking": "セルフチェック中...",
  "card": {
    "version": "バージョン",
    "domain": "ドメイン",
    "hardware": "ハードウェアプラットフォーム",
    "checkStatus": "セルフチェックステータス",
    "packageSize": "パッケージサイズ",
    "md5": "MD5 チェックサム",
    "capabilities": "機能",
    "details": "詳細",
    "delete": "パッケージを削除",
    "deleteConfirm": "このアルゴリズムパッケージを削除してもよろしいですか？この操作はファイルを完全に削除し、元に戻せません。",
    "selfCheckBtn": "セルフチェック実行",
    "selfCheckSuccess": "セルフチェック成功",
    "selfCheckFailed": "セルフチェック失敗",
    "selfCheckRunning": "セルフチェック実行中",
    "selfCheckPending": "セルフチェック待機中",
    "schema": "結果スキーマ",
    "paramsSchema": "パラメータスキーマ",
    "copied": "クリップボードにコピーしました",
    "noSchema": "スキーマが提供されていません",
    "errorMsg": "セルフチェックエラー",
    "empty": "アルゴリズムパッケージはまだありません。上部からアップロードしてください。"
  },
  "message": {
    "uploadSuccess": "アルゴリズムパッケージのアップロードに成功しました。セルフチェックを開始します",
    "uploadFailed": "パッケージのアップロードに失敗しました",
    "deleteSuccess": "アルゴリズムパッケージの削除に成功しました",
    "deleteFailed": "パッケージの削除に失敗しました",
    "selfCheckTriggered": "セルフチェックコマンドを送信しました",
    "selfCheckFailed": "セルフチェックコマンドの送信に失敗しました",
    "operationFailed": "操作に失敗しました"
  },
  "search": {
    "placeholder": "アルゴリズム名、エイリアス、ドメインを検索..."
  },
  "stats": {
    "total": "総パッケージ数",
    "passed": "セルフチェック成功",
    "failed": "セルフチェック失敗",
    "active": "有効"
  }
} as const;

export default algorithmpackage;
