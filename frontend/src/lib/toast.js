// Global toast notification store
// Usage: import { toast } from '../lib/toast.js';
//   toast.success('保存成功');
//   toast.error('操作失败: ...');
//   toast.warning('请注意...');
//   toast.info('提示信息');

let _toasts = [];
let _id = 0;
let _listeners = [];

function _notify() {
  const snapshot = [..._toasts];
  _listeners.forEach(fn => fn(snapshot));
}

function addToast(type, message, duration = 3500) {
  const id = ++_id;
  _toasts = [..._toasts, { id, type, message }];
  _notify();
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration);
  }
}

function removeToast(id) {
  _toasts = _toasts.filter(t => t.id !== id);
  _notify();
}

export const toast = {
  success: (msg, duration) => addToast('success', msg, duration),
  error: (msg, duration) => addToast('error', msg, duration ?? 5000),
  warning: (msg, duration) => addToast('warning', msg, duration),
  info: (msg, duration) => addToast('info', msg, duration),
};

export function onToastChange(fn) {
  _listeners.push(fn);
  return () => { _listeners = _listeners.filter(l => l !== fn); };
}

export function getToasts() {
  return [..._toasts];
}

export { removeToast };
