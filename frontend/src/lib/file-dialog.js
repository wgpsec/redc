/**
 * File dialog helpers that work in both Wails desktop and HTTP browser mode.
 *
 * In Wails mode: delegates to native OS dialogs via SelectFile/SelectDirectory/SelectSaveFile.
 * In HTTP mode: uses browser <input type="file"> / download for file operations.
 */

import { SelectFile as WailsSelectFile, SelectDirectory as WailsSelectDirectory, SelectSaveFile as WailsSelectSaveFile } from '../../wailsjs/go/main/App.js';

function isWebMode() {
  return !!window.__redcWebMode__;
}

/**
 * Opens a file picker. Returns the selected file path (Wails) or uploads content (HTTP).
 * In HTTP mode, returns a virtual path and stores file content for later retrieval.
 */
export async function selectFile(title) {
  if (!isWebMode()) {
    return WailsSelectFile(title);
  }

  return new Promise((resolve) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.style.display = 'none';
    input.onchange = async () => {
      if (input.files && input.files.length > 0) {
        const file = input.files[0];
        // Upload file to server via API and get server-side path
        try {
          const path = await uploadFileToServer(file);
          resolve(path);
        } catch (e) {
          console.error('File upload failed:', e);
          resolve('');
        }
      } else {
        resolve('');
      }
      document.body.removeChild(input);
    };
    input.oncancel = () => {
      resolve('');
      document.body.removeChild(input);
    };
    document.body.appendChild(input);
    input.click();
  });
}

/**
 * Directory selection — not supported in browser mode.
 */
export async function selectDirectory(title) {
  if (!isWebMode()) {
    return WailsSelectDirectory(title);
  }
  throw new Error('目录选择在浏览器模式下不可用 / Directory selection is not available in browser mode');
}

/**
 * Save file dialog. In HTTP mode, triggers a browser download.
 */
export async function selectSaveFile(title, defaultFilename) {
  if (!isWebMode()) {
    return WailsSelectSaveFile(title, defaultFilename);
  }
  // In web mode, return a server temp path — the caller will write to it, then we download
  return `__web_download__/${defaultFilename}`;
}

/**
 * Upload a file to the HTTP server, returns the server-side temp path.
 */
async function uploadFileToServer(file) {
  const token = localStorage.getItem('redc_token') || '';
  const formData = new FormData();
  formData.append('file', file);

  /** @type {Record<string, string>} */
  const headers = {};
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const resp = await fetch('/api/upload', {
    method: 'POST',
    headers,
    body: formData,
  });

  if (!resp.ok) {
    throw new Error(`Upload failed: ${resp.status}`);
  }

  const data = await resp.json();
  return data.path;
}
