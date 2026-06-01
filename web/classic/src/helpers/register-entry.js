export function isRegisterEnabled(status) {
  const raw =
    status?.register_enabled ??
    (status?.data && typeof status.data === 'object'
      ? status.data.register_enabled
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

export function hasRegisterStatusValue(status) {
  return Boolean(
    status?.register_enabled !== undefined ||
    (status?.data &&
      typeof status.data === 'object' &&
      status.data.register_enabled !== undefined)
  );
}

export function getCachedStatus() {
  try {
    if (typeof window === 'undefined') return null;
    const saved = window.localStorage.getItem('status');
    return saved ? JSON.parse(saved) : null;
  } catch {
    return null;
  }
}

export function getEffectiveStatus(status) {
  return status || getCachedStatus() || {};
}
