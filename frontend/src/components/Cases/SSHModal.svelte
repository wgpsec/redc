<script>
  import { onMount } from 'svelte';
  import { ExecCommand, ExecUserdata, UploadFile, DownloadFile, SelectFile, SelectDirectory, SelectSaveFile } from '../../../wailsjs/go/main/App.js';
  import WebTerminal from './WebTerminal.svelte';
  import FileManager from './FileManager.svelte';
  import { loadUserdataTemplates, getGroupedTemplates, userdataCategoryNames } from '../../lib/userdataTemplates.js';

  let { t, caseId, caseName, onClose } = $props();

  let activeTab = $state('exec');
  let command = $state('');
  let execResult = $state(null);
  let execLoading = $state(false);
  
  let uploadLocalPath = $state('');
  let uploadRemotePath = $state('/root/');
  let uploadLoading = $state(false);
  let uploadResult = $state(null);
  
  let downloadRemotePath = $state('');
  let downloadLocalPath = $state('');
  let downloadLoading = $state(false);
  let downloadResult = $state(null);

  // 新增：userdata 执行状态
  let userdataTemplates = $state([]);
  let userdataTemplatesLoading = $state(true);
  let selectedTemplate = $state(null);
  let userdataExecLoading = $state(false);
  let userdataExecResult = $state(null);
  let expandedCategories = $state({});

  // Grouped templates by category
  let groupedUserdataTemplates = $derived(() => {
    return getGroupedTemplates(userdataTemplates);
  });

  onMount(async () => {
    userdataTemplates = await loadUserdataTemplates();
    userdataTemplatesLoading = false;
  });

  // 新增：终端和文件管理器状态
  let showTerminal = $state(false);
  let showFileManager = $state(false);

  async function handleExec() {
    if (!command.trim()) return;
    
    execLoading = true;
    execResult = null;
    
    try {
      const result = await ExecCommand(caseId, command);
      execResult = result;
    } catch (e) {
      execResult = { success: false, error: e.message || String(e) };
    } finally {
      execLoading = false;
    }
  }

  async function handleSelectUploadFile() {
    try {
      const path = await SelectFile(t.selectFile || '选择文件');
      if (path) {
        uploadLocalPath = path;
      }
    } catch (e) {
      console.error('选择文件失败:', e);
    }
  }

  async function handleUpload() {
    if (!uploadLocalPath || !uploadRemotePath) return;
    
    uploadLoading = true;
    uploadResult = null;
    
    try {
      const result = await UploadFile(caseId, uploadLocalPath, uploadRemotePath);
      uploadResult = result;
      if (result.success) {
        uploadLocalPath = '';
      }
    } catch (e) {
      uploadResult = { success: false, error: e.message || String(e) };
    } finally {
      uploadLoading = false;
    }
  }

  async function handleSelectDownloadDir() {
    try {
      const path = await SelectDirectory(t.selectDirectory || '选择保存目录');
      if (path) {
        downloadLocalPath = path;
      }
    } catch (e) {
      console.error('选择目录失败:', e);
    }
  }

  async function handleDownload() {
    if (!downloadRemotePath || !downloadLocalPath) return;
    
    downloadLoading = true;
    downloadResult = null;
    
    try {
      const result = await DownloadFile(caseId, downloadRemotePath, downloadLocalPath);
      downloadResult = result;
      if (result.success) {
        downloadRemotePath = '';
      }
    } catch (e) {
      downloadResult = { success: false, error: e.message || String(e) };
    } finally {
      downloadLoading = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  async function handleExecUserdata() {
    if (!selectedTemplate?.script) return;
    
    userdataExecLoading = true;
    userdataExecResult = null;
    
    try {
      const result = await ExecUserdata(caseId, selectedTemplate.script);
      userdataExecResult = result;
    } catch (e) {
      userdataExecResult = { success: false, error: e.message || String(e) };
    } finally {
      userdataExecLoading = false;
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
  <div class="bg-white rounded-xl shadow-xl w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col" onclick={(e) => e.stopPropagation()}>
    <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between flex-shrink-0">
      <div>
        <h3 class="text-[15px] font-semibold text-gray-900">{t.sshOperations || 'SSH 运维'}</h3>
        <p class="text-[12px] text-gray-500 mt-0.5">{caseName} <span class="text-gray-400">({caseId?.substring(0, 8)})</span></p>
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
          class="px-3 py-2 text-[12px] font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 transition-colors flex items-center gap-2"
          onclick={() => showTerminal = true}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
          {t.openTerminal || '打开终端'}
        </button>
        <button
          class="px-3 py-2 text-[12px] font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
          onclick={() => showFileManager = true}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          {t.openFileManager || '文件管理器'}
        </button>
      </div>
    </div>

    <div class="flex border-b border-gray-100 flex-shrink-0">
      <button
        class="flex-1 px-4 py-3 text-[13px] font-medium transition-colors relative"
        class:text-gray-900={activeTab === 'exec'}
        class:text-gray-500={activeTab !== 'exec'}
        class:hover:text-gray-700={activeTab !== 'exec'}
        onclick={() => activeTab = 'exec'}
      >
        <div class="flex items-center justify-center gap-2">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
          {t.execCommand || '执行命令'}
        </div>
        {#if activeTab === 'exec'}
          <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-gray-900"></div>
        {/if}
      </button>
      <button
        class="flex-1 px-4 py-3 text-[13px] font-medium transition-colors relative"
        class:text-gray-900={activeTab === 'upload'}
        class:text-gray-500={activeTab !== 'upload'}
        class:hover:text-gray-700={activeTab !== 'upload'}
        onclick={() => activeTab = 'upload'}
      >
        <div class="flex items-center justify-center gap-2">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
          </svg>
          {t.uploadFile || '上传文件'}
        </div>
        {#if activeTab === 'upload'}
          <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-gray-900"></div>
        {/if}
      </button>
      <button
        class="flex-1 px-4 py-3 text-[13px] font-medium transition-colors relative"
        class:text-gray-900={activeTab === 'download'}
        class:text-gray-500={activeTab !== 'download'}
        class:hover:text-gray-700={activeTab !== 'download'}
        onclick={() => activeTab = 'download'}
      >
        <div class="flex items-center justify-center gap-2">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
          </svg>
          {t.downloadFile || '下载文件'}
        </div>
        {#if activeTab === 'download'}
          <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-gray-900"></div>
        {/if}
      </button>
      <button
        class="flex-1 px-4 py-3 text-[13px] font-medium transition-colors relative"
        class:text-gray-900={activeTab === 'userdata'}
        class:text-gray-500={activeTab !== 'userdata'}
        class:hover:text-gray-700={activeTab !== 'userdata'}
        onclick={() => activeTab = 'userdata'}
      >
        <div class="flex items-center justify-center gap-2">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 01-1.043 3.296 3.745 3.745 0 01-3.296 1.043A3.745 3.745 0 0112 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 01-3.296-1.043 3.745 3.745 0 01-1.043-3.296A3.745 3.745 0 013 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 011.043-3.296 3.746 3.746 0 013.296-1.043A3.746 3.746 0 0112 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 013.296 1.043 3.746 3.746 0 011.043 3.296A3.745 3.745 0 0121 12z" />
          </svg>
          {t.execUserdata || '执行 Userdata'}
        </div>
        {#if activeTab === 'userdata'}
          <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-gray-900"></div>
        {/if}
      </button>
    </div>

    <div class="flex-1 overflow-auto p-5">
      {#if activeTab === 'exec'}
        <div class="space-y-4">
          <div>
            <label for="sshCommand" class="block text-[12px] font-medium text-gray-700 mb-2">{t.command || '命令'}</label>
            <div class="flex gap-2">
              <input
                id="sshCommand"
                type="text"
                class="flex-1 px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                placeholder="whoami"
                bind:value={command}
                onkeydown={(e) => { if (e.key === 'Enter') handleExec(); }}
              />
              <button
                class="px-4 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                onclick={handleExec}
                disabled={execLoading || !command.trim()}
              >
                {#if execLoading}
                  <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                {:else}
                  {t.execute || '执行'}
                {/if}
              </button>
            </div>
          </div>

          {#if execResult}
            <div class="space-y-3">
              {#if execResult.stdout}
                <div>
                  <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">stdout</div>
                  <pre class="bg-gray-900 text-green-400 text-[12px] p-3 rounded-lg overflow-auto max-h-48 font-mono">{execResult.stdout}</pre>
                </div>
              {/if}
              {#if execResult.stderr}
                <div>
                  <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">stderr</div>
                  <pre class="bg-gray-900 text-red-400 text-[12px] p-3 rounded-lg overflow-auto max-h-48 font-mono">{execResult.stderr}</pre>
                </div>
              {/if}
              {#if execResult.error}
                <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
                  <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <span class="text-[12px] text-red-700">{execResult.error}</span>
                </div>
              {/if}
              {#if execResult.success}
                <div class="flex items-center gap-2 px-3 py-2 bg-emerald-50 border border-emerald-100 rounded-lg">
                  <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                  <span class="text-[12px] text-emerald-700">{t.execSuccess || '执行成功'} (exit code: {execResult.exitCode})</span>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {:else if activeTab === 'upload'}
        <div class="space-y-4">
          <div>
            <label for="localFile" class="block text-[12px] font-medium text-gray-700 mb-2">{t.localFile || '本地文件'}</label>
            <div class="flex gap-2">
              <input
                id="localFile"
                type="text"
                class="flex-1 px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent bg-gray-50"
                placeholder={t.selectFilePlaceholder || '点击选择文件...'}
                bind:value={uploadLocalPath}
                readonly
              />
              <button
                class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                onclick={handleSelectUploadFile}
              >
                {t.browse || '浏览'}
              </button>
            </div>
          </div>

          <div>
            <label for="remotePath" class="block text-[12px] font-medium text-gray-700 mb-2">{t.remotePath || '远程路径'}</label>
            <input
              id="remotePath"
              type="text"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              placeholder="/root/"
              bind:value={uploadRemotePath}
            />
          </div>

          <button
            class="w-full px-4 py-2.5 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            onclick={handleUpload}
            disabled={uploadLoading || !uploadLocalPath || !uploadRemotePath}
          >
            {#if uploadLoading}
              <span class="flex items-center justify-center gap-2">
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                {t.uploading || '上传中...'}
              </span>
            {:else}
              {t.upload || '上传'}
            {/if}
          </button>

          {#if uploadResult}
            {#if uploadResult.success}
              <div class="flex items-center gap-2 px-3 py-2 bg-emerald-50 border border-emerald-100 rounded-lg">
                <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <span class="text-[12px] text-emerald-700">{t.uploadSuccess || '上传成功'}</span>
              </div>
            {:else}
              <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
                <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span class="text-[12px] text-red-700">{uploadResult.error}</span>
              </div>
            {/if}
          {/if}
        </div>
      {:else if activeTab === 'download'}
        <div class="space-y-4">
          <div>
            <label for="remoteFile" class="block text-[12px] font-medium text-gray-700 mb-2">{t.remoteFile || '远程文件'}</label>
            <input
              id="remoteFile"
              type="text"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              placeholder="/root/config.txt"
              bind:value={downloadRemotePath}
            />
          </div>

          <div>
            <label for="localDir" class="block text-[12px] font-medium text-gray-700 mb-2">{t.localDirectory || '本地目录'}</label>
            <div class="flex gap-2">
              <input
                id="localDir"
                type="text"
                class="flex-1 px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent bg-gray-50"
                placeholder={t.selectDirectoryPlaceholder || '点击选择目录...'}
                bind:value={downloadLocalPath}
                readonly
              />
              <button
                class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                onclick={handleSelectDownloadDir}
              >
                {t.browse || '浏览'}
              </button>
            </div>
          </div>

          <button
            class="w-full px-4 py-2.5 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            onclick={handleDownload}
            disabled={downloadLoading || !downloadRemotePath || !downloadLocalPath}
          >
            {#if downloadLoading}
              <span class="flex items-center justify-center gap-2">
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                {t.downloading || '下载中...'}
              </span>
            {:else}
              {t.download || '下载'}
            {/if}
          </button>

          {#if downloadResult}
            {#if downloadResult.success}
              <div class="flex items-center gap-2 px-3 py-2 bg-emerald-50 border border-emerald-100 rounded-lg">
                <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <span class="text-[12px] text-emerald-700">{t.downloadSuccess || '下载成功'}</span>
              </div>
            {:else}
              <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
                <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span class="text-[12px] text-red-700">{downloadResult.error}</span>
              </div>
            {/if}
          {/if}
        </div>
      {:else if activeTab === 'userdata'}
        <div class="space-y-4">
          {#if userdataTemplatesLoading}
            <div class="text-center py-8 text-gray-500 text-[13px]">
              {t.loading || '加载中...'}
            </div>
          {:else if userdataTemplates.length === 0}
            <div class="text-center py-8 text-gray-500 text-[13px]">
              {t.noTemplates || '暂无可用模板'}
            </div>
          {:else}
            <div class="space-y-2 max-h-64 overflow-y-auto">
              {#each Object.entries(groupedUserdataTemplates()) as [category, categoryTemplates]}
                <div class="border border-gray-100 rounded-lg overflow-hidden">
                  <button
                    class="w-full flex items-center justify-between px-3 py-2 bg-gray-50 hover:bg-gray-100 transition-colors"
                    onclick={() => expandedCategories[category] = !expandedCategories[category]}
                  >
                    <span class="text-[12px] font-medium text-gray-700">
                      {userdataCategoryNames[category] || category}
                      <span class="ml-1 text-gray-500">({categoryTemplates.length})</span>
                    </span>
                    <svg class="w-4 h-4 text-gray-500 transition-transform {expandedCategories[category] ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {#if expandedCategories[category]}
                    <div class="space-y-1 p-2 bg-white">
                      {#each categoryTemplates as template}
                        <button
                          class="w-full text-left px-3 py-2 text-[12px] bg-white hover:bg-blue-50 border border-gray-100 hover:border-blue-300 rounded transition-colors {selectedTemplate === template ? 'border-blue-500 bg-blue-50' : ''}"
                          onclick={() => selectedTemplate = template}
                        >
                          <span class="font-medium text-gray-900">{template.nameZh || template.name}</span>
                          {#if template.description}
                            <span class="block text-gray-500 text-[10px] truncate">{template.description}</span>
                          {/if}
                        </button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>

            {#if selectedTemplate}
              <div>
                <span class="block text-[12px] font-medium text-gray-700 mb-2">{t.scriptPreview || '脚本预览'}</span>
                <pre class="bg-gray-900 text-gray-100 text-[12px] p-3 rounded-lg overflow-auto max-h-48 font-mono">{selectedTemplate.script}</pre>
              </div>

              {#if selectedTemplate.description}
                <div class="text-[12px] text-gray-500">{selectedTemplate.description}</div>
              {/if}

              <button
                class="w-full px-4 py-2.5 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                onclick={handleExecUserdata}
                disabled={userdataExecLoading || !selectedTemplate?.script}
              >
                {#if userdataExecLoading}
                  <span class="flex items-center justify-center gap-2">
                    <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    {t.executing || '执行中...'}
                  </span>
                {:else}
                  {t.execUserdata || '执行 Userdata'}
                {/if}
              </button>
            {/if}

            {#if userdataExecResult}
              <div class="space-y-3">
                {#if userdataExecResult.stdout}
                  <div>
                    <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">stdout</div>
                    <pre class="bg-gray-900 text-green-400 text-[12px] p-3 rounded-lg overflow-auto max-h-48 font-mono">{userdataExecResult.stdout}</pre>
                  </div>
                {/if}
                {#if userdataExecResult.stderr}
                  <div>
                    <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">stderr</div>
                    <pre class="bg-gray-900 text-red-400 text-[12px] p-3 rounded-lg overflow-auto max-h-48 font-mono">{userdataExecResult.stderr}</pre>
                  </div>
                {/if}
                {#if userdataExecResult.error}
                  <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
                    <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <span class="text-[12px] text-red-700">{userdataExecResult.error}</span>
                  </div>
                {/if}
                {#if userdataExecResult.success}
                  <div class="flex items-center gap-2 px-3 py-2 bg-emerald-50 border border-emerald-100 rounded-lg">
                    <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                    <span class="text-[12px] text-emerald-700">{t.execSuccess || '执行成功'} (exit code: {userdataExecResult.exitCode})</span>
                  </div>
                {/if}
              </div>
            {/if}
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<!-- Web Terminal -->
{#if showTerminal}
  <WebTerminal {t} {caseId} {caseName} onClose={() => showTerminal = false} />
{/if}

<!-- File Manager -->
{#if showFileManager}
  <FileManager {t} {caseId} {caseName} onClose={() => showFileManager = false} />
{/if}
