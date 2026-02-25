<script>
  import { onMount } from 'svelte';
  import { 
    ListRemoteFiles, 
    CreateRemoteDirectory, 
    DeleteRemoteFile, 
    RenameRemoteFile,
    UploadFile,
    DownloadFile,
    SelectFile,
    SelectDirectory
  } from '../../../wailsjs/go/main/App.js';

  let { t, caseId, caseName, onClose } = $props();

  let currentPath = $state('/root');
  let files = $state([]);
  let loading = $state(false);
  let error = $state('');
  let selectedFile = $state(null);
  
  // 操作状态
  let showNewFolderDialog = $state(false);
  let showRenameDialog = $state(false);
  let showDeleteConfirm = $state(false);
  let newFolderName = $state('');
  let renameNewName = $state('');
  let renameTarget = $state(null);
  let deleteTarget = $state(null);
  
  // 上传/下载状态
  let uploading = $state(false);
  let downloading = $state(false);
  let uploadProgress = $state('');
  let downloadProgress = $state('');

  onMount(() => {
    loadFiles();
  });

  async function loadFiles() {
    loading = true;
    error = '';
    try {
      files = await ListRemoteFiles(caseId, currentPath);
      // 排序：目录在前，文件在后
      files.sort((a, b) => {
        if (a.isDir && !b.isDir) return -1;
        if (!a.isDir && b.isDir) return 1;
        return a.name.localeCompare(b.name);
      });
    } catch (e) {
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  function navigateTo(path) {
    currentPath = path;
    selectedFile = null;
    loadFiles();
  }

  function navigateUp() {
    if (currentPath === '/') return;
    const parts = currentPath.split('/').filter(p => p);
    parts.pop();
    navigateTo('/' + parts.join('/'));
  }

  function openFile(file) {
    if (file.isDir) {
      const newPath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      navigateTo(newPath);
    } else {
      selectedFile = file;
    }
  }

  async function handleCreateFolder() {
    if (!newFolderName.trim()) return;
    
    try {
      const newPath = currentPath === '/' ? `/${newFolderName}` : `${currentPath}/${newFolderName}`;
      await CreateRemoteDirectory(caseId, newPath);
      showNewFolderDialog = false;
      newFolderName = '';
      await loadFiles();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleRename() {
    if (!renameNewName.trim() || !renameTarget) return;
    
    try {
      const oldPath = currentPath === '/' ? `/${renameTarget.name}` : `${currentPath}/${renameTarget.name}`;
      const newPath = currentPath === '/' ? `/${renameNewName}` : `${currentPath}/${renameNewName}`;
      await RenameRemoteFile(caseId, oldPath, newPath);
      showRenameDialog = false;
      renameNewName = '';
      renameTarget = null;
      await loadFiles();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    
    try {
      const path = currentPath === '/' ? `/${deleteTarget.name}` : `${currentPath}/${deleteTarget.name}`;
      await DeleteRemoteFile(caseId, path);
      showDeleteConfirm = false;
      deleteTarget = null;
      selectedFile = null;
      await loadFiles();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleUpload() {
    try {
      const localPath = await SelectFile(t.selectFile || '选择文件');
      if (!localPath) return;
      
      uploading = true;
      uploadProgress = '上传中...';
      
      const result = await UploadFile(caseId, localPath, currentPath);
      if (result.success) {
        uploadProgress = '上传成功';
        await loadFiles();
        setTimeout(() => { uploadProgress = ''; uploading = false; }, 2000);
      } else {
        error = result.error;
        uploading = false;
        uploadProgress = '';
      }
    } catch (e) {
      error = e.message || String(e);
      uploading = false;
      uploadProgress = '';
    }
  }

  async function handleDownload(file) {
    try {
      const localDir = await SelectDirectory(t.selectDirectory || '选择保存目录');
      if (!localDir) return;
      
      downloading = true;
      downloadProgress = '下载中...';
      
      const remotePath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      const result = await DownloadFile(caseId, remotePath, localDir);
      if (result.success) {
        downloadProgress = '下载成功';
        setTimeout(() => { downloadProgress = ''; downloading = false; }, 2000);
      } else {
        error = result.error;
        downloading = false;
        downloadProgress = '';
      }
    } catch (e) {
      error = e.message || String(e);
      downloading = false;
      downloadProgress = '';
    }
  }

  function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
  }

  function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleString();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      if (showNewFolderDialog) showNewFolderDialog = false;
      else if (showRenameDialog) showRenameDialog = false;
      else if (showDeleteConfirm) showDeleteConfirm = false;
      else onClose();
    }
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onclick={handleBackdropClick}>
  <div class="bg-white rounded-xl shadow-xl w-full max-w-6xl h-[85vh] overflow-hidden flex flex-col" onclick={(e) => e.stopPropagation()}>
    <!-- Header -->
    <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center">
          <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.fileManager || '文件管理器'}</h3>
          <p class="text-[12px] text-gray-500 mt-0.5">{caseName} <span class="text-gray-400">({caseId?.substring(0, 8)})</span></p>
        </div>
      </div>
      <button
        class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
        onclick={onClose}
        aria-label="关闭"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Toolbar -->
    <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-2">
        <button
          class="px-3 py-1.5 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
          onclick={navigateUp}
          disabled={currentPath === '/'}
          aria-label="返回上级目录"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
        </button>
        <div class="px-3 py-1.5 text-[12px] font-mono text-gray-700 bg-gray-50 rounded-lg border border-gray-100">
          {currentPath}
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="px-3 py-1.5 text-[12px] font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 transition-colors flex items-center gap-1.5"
          onclick={() => showNewFolderDialog = true}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          {t.newFolder || '新建文件夹'}
        </button>
        <button
          class="px-3 py-1.5 text-[12px] font-medium text-white bg-emerald-500 rounded-lg hover:bg-emerald-600 transition-colors flex items-center gap-1.5 disabled:opacity-50"
          onclick={handleUpload}
          disabled={uploading}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
          </svg>
          {uploading ? uploadProgress : (t.upload || '上传')}
        </button>
        <button
          class="px-3 py-1.5 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
          onclick={loadFiles}
          aria-label="刷新文件列表"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Error Display -->
    {#if error}
      <div class="mx-5 mt-3 flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
        <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <span class="text-[12px] text-red-700 flex-1">{error}</span>
        <button class="text-red-400 hover:text-red-600" onclick={() => error = ''} aria-label="关闭错误">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/if}

    <!-- File List -->
    <div class="flex-1 overflow-auto">
      {#if loading}
        <div class="flex items-center justify-center h-full">
          <svg class="w-8 h-8 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
      {:else if files.length === 0}
        <div class="flex flex-col items-center justify-center h-full text-gray-400">
          <svg class="w-16 h-16 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          <p class="text-[13px]">{t.emptyDirectory || '空目录'}</p>
        </div>
      {:else}
        <table class="w-full text-[12px]">
          <thead class="bg-gray-50 border-b border-gray-100 sticky top-0">
            <tr>
              <th class="text-left px-5 py-3 font-semibold text-gray-600">{t.name || '名称'}</th>
              <th class="text-right px-5 py-3 font-semibold text-gray-600">{t.size || '大小'}</th>
              <th class="text-left px-5 py-3 font-semibold text-gray-600">{t.modified || '修改时间'}</th>
              <th class="text-right px-5 py-3 font-semibold text-gray-600">{t.actions || '操作'}</th>
            </tr>
          </thead>
          <tbody>
            {#each files as file}
              <tr 
                class="border-b border-gray-50 hover:bg-gray-50 transition-colors cursor-pointer"
                class:bg-blue-50={selectedFile === file}
                onclick={() => openFile(file)}
              >
                <td class="px-5 py-3">
                  <div class="flex items-center gap-2">
                    {#if file.isDir}
                      <svg class="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                      </svg>
                    {:else}
                      <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                    {/if}
                    <span class="font-medium text-gray-900">{file.name}</span>
                    {#if file.isLink}
                      <span class="text-[10px] px-1.5 py-0.5 bg-purple-100 text-purple-600 rounded">LINK</span>
                    {/if}
                  </div>
                </td>
                <td class="px-5 py-3 text-right text-gray-600">
                  {file.isDir ? '-' : formatSize(file.size)}
                </td>
                <td class="px-5 py-3 text-gray-600">
                  {formatDate(file.modTime)}
                </td>
                <td class="px-5 py-3 text-right">
                  <div class="flex items-center justify-end gap-1">
                    {#if !file.isDir}
                      <button
                        class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                        onclick={(e) => { e.stopPropagation(); handleDownload(file); }}
                        disabled={downloading}
                        title={t.download || '下载'}
                      >
                        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                        </svg>
                      </button>
                    {/if}
                    <button
                      class="p-1.5 text-gray-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                      onclick={(e) => { e.stopPropagation(); renameTarget = file; renameNewName = file.name; showRenameDialog = true; }}
                      title={t.rename || '重命名'}
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
                      </svg>
                    </button>
                    <button
                      class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                      onclick={(e) => { e.stopPropagation(); deleteTarget = file; showDeleteConfirm = true; }}
                      title={t.delete || '删除'}
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                      </svg>
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>

    <!-- Status Bar -->
    <div class="px-5 py-2 border-t border-gray-100 flex items-center justify-between text-[11px] text-gray-500 flex-shrink-0">
      <div>{files.length} {t.items || '项'}</div>
      {#if downloadProgress}
        <div class="text-blue-600">{downloadProgress}</div>
      {/if}
    </div>
  </div>
</div>

<!-- New Folder Dialog -->
{#if showNewFolderDialog}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]" onclick={() => showNewFolderDialog = false}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md p-5" onclick={(e) => e.stopPropagation()}>
      <h3 class="text-[14px] font-semibold text-gray-900 mb-4">{t.newFolder || '新建文件夹'}</h3>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        type="text"
        class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
        placeholder={t.folderName || '文件夹名称'}
        bind:value={newFolderName}
        onkeydown={(e) => { if (e.key === 'Enter') handleCreateFolder(); }}
        autofocus
      />
      <div class="flex items-center justify-end gap-2 mt-4">
        <button
          class="px-4 py-2 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
          onclick={() => { showNewFolderDialog = false; newFolderName = ''; }}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[12px] font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50"
          onclick={handleCreateFolder}
          disabled={!newFolderName.trim()}
        >
          {t.create || '创建'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Rename Dialog -->
{#if showRenameDialog}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]" onclick={() => showRenameDialog = false}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md p-5" onclick={(e) => e.stopPropagation()}>
      <h3 class="text-[14px] font-semibold text-gray-900 mb-4">{t.rename || '重命名'}</h3>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        type="text"
        class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500"
        placeholder={t.newName || '新名称'}
        bind:value={renameNewName}
        onkeydown={(e) => { if (e.key === 'Enter') handleRename(); }}
        autofocus
      />
      <div class="flex items-center justify-end gap-2 mt-4">
        <button
          class="px-4 py-2 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
          onclick={() => { showRenameDialog = false; renameNewName = ''; renameTarget = null; }}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[12px] font-medium text-white bg-amber-500 rounded-lg hover:bg-amber-600 transition-colors disabled:opacity-50"
          onclick={handleRename}
          disabled={!renameNewName.trim()}
        >
          {t.rename || '重命名'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Confirm Dialog -->
{#if showDeleteConfirm}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]" onclick={() => showDeleteConfirm = false}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md p-5" onclick={(e) => e.stopPropagation()}>
      <h3 class="text-[14px] font-semibold text-gray-900 mb-2">{t.confirmDelete || '确认删除'}</h3>
      <p class="text-[12px] text-gray-600 mb-4">
        {t.deleteConfirmMessage || '确定要删除'} <span class="font-medium text-gray-900">{deleteTarget?.name}</span> {t.questionMark || '吗？'}
        {#if deleteTarget?.isDir}
          <span class="text-red-600">{t.deleteDirectoryWarning || '（包括所有子文件和目录）'}</span>
        {/if}
      </p>
      <div class="flex items-center justify-end gap-2">
        <button
          class="px-4 py-2 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
          onclick={() => { showDeleteConfirm = false; deleteTarget = null; }}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[12px] font-medium text-white bg-red-500 rounded-lg hover:bg-red-600 transition-colors"
          onclick={handleDelete}
        >
          {t.delete || '删除'}
        </button>
      </div>
    </div>
  </div>
{/if}
