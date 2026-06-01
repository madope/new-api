export function isNoticeButtonEnabled(status) {
  const raw =
    status?.notice_button_enabled ??
    (status?.data && typeof status.data === 'object'
      ? status.data.notice_button_enabled
      : undefined);

  if (raw === undefined || raw === null) {
    return true;
  }
  if (typeof raw === 'boolean') {
    return raw;
  }
  if (typeof raw === 'number') {
    return raw !== 0;
  }
  if (typeof raw === 'string') {
    const normalized = raw.trim().toLowerCase();
    if (normalized === 'false' || normalized === '0') {
      return false;
    }
    if (normalized === 'true' || normalized === '1') {
      return true;
    }
  }
  return true;
}
