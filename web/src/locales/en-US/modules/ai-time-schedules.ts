export const aiTimeSchedules = {
  title: 'Time Schedules',
  fields: {
    name: 'Schedule Name',
    description: 'Description',
    dateRange: 'Valid Period',
    timeWindows: 'Time Windows',
    updatedAt: 'Updated At',
  },
  actions: {
    create: 'New Schedule',
    edit: 'Edit',
    delete: 'Delete',
    save: 'Save',
  },
  form: {
    namePlaceholder: 'e.g.: Weekday daytime, 24/7 monitoring',
    descriptionPlaceholder: 'Optional, describe the purpose',
    startDate: 'Start Date',
    endDate: 'End Date',
    timeWindows: 'Daily Time Windows',
    addTimeWindow: 'Add Time Window',
  },
  message: {
    deleteConfirm: 'Are you sure to delete this time schedule? Tasks referencing it will not be affected.',
    deleteSuccess: 'Deleted successfully',
    deleteFailed: 'Delete failed',
    updateSuccess: 'Updated successfully',
    createSuccess: 'Created successfully',
    nameRequired: 'Please enter schedule name',
    startDateRequired: 'Please select start date',
    endDateRequired: 'Please select end date',
    dateInvalid: 'End date cannot be earlier than start date',
  },
  empty: 'No time schedules yet',
} as const;
export default aiTimeSchedules;
