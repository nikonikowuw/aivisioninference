export const aiTimeSchedules = {
  title: '時間設定',
  fields: {
    name: '設定名',
    description: '説明',
    dateRange: '有効期間',
    timeWindows: '時間ウィンドウ',
    updatedAt: '更新日時',
  },
  actions: {
    create: '新規設定',
    edit: '編集',
    delete: '削除',
    save: '保存',
  },
  form: {
    namePlaceholder: '例：平日昼間、24時間監視',
    descriptionPlaceholder: '任意、用途の説明',
    startDate: '開始日',
    endDate: '終了日',
    timeWindows: '毎日の時間ウィンドウ',
    addTimeWindow: '時間帯を追加',
  },
  message: {
    deleteConfirm: 'この時間設定を削除しますか？参照しているタスクには影響しません。',
    deleteSuccess: '削除しました',
    deleteFailed: '削除に失敗しました',
    updateSuccess: '更新しました',
    createSuccess: '作成しました',
    nameRequired: '設定名を入力してください',
    startDateRequired: '開始日を選択してください',
    endDateRequired: '終了日を選択してください',
    dateInvalid: '終了日は開始日より前にできません',
  },
  empty: '時間設定がありません',
} as const;
export default aiTimeSchedules;
