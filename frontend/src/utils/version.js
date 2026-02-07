/**
 * 版本号工具函数
 */

/**
 * 标准化版本号，移除 'v' 前缀并去除空格
 * @param {string|number} value - 版本号
 * @returns {string} 标准化后的版本号
 */
export function normalizeVersion(value) {
  if (!value) return '';
  return String(value).trim().replace(/^v/i, '');
}

/**
 * 比较两个版本号
 * @param {string|number} a - 第一个版本号
 * @param {string|number} b - 第二个版本号
 * @returns {number} 1 表示 a > b, -1 表示 a < b, 0 表示相等
 */
export function compareVersions(a, b) {
  const va = normalizeVersion(a).split('.').map(part => parseInt(part, 10));
  const vb = normalizeVersion(b).split('.').map(part => parseInt(part, 10));
  const maxLen = Math.max(va.length, vb.length);
  for (let i = 0; i < maxLen; i += 1) {
    const na = Number.isFinite(va[i]) ? va[i] : 0;
    const nb = Number.isFinite(vb[i]) ? vb[i] : 0;
    if (na > nb) return 1;
    if (na < nb) return -1;
  }
  return 0;
}

/**
 * 检查模板是否有更新
 * @param {Object} tmpl - 模板对象
 * @param {boolean} tmpl.installed - 是否已安装
 * @param {string} tmpl.latest - 最新版本号
 * @param {string} tmpl.localVersion - 本地版本号
 * @returns {boolean} 是否有更新
 */
export function hasUpdate(tmpl) {
  if (!tmpl || !tmpl.installed) return false;
  if (!tmpl.latest || !tmpl.localVersion) return false;
  return compareVersions(tmpl.latest, tmpl.localVersion) > 0;
}
