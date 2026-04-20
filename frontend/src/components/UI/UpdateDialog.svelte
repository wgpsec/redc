<script>
  import Modal from './Modal.svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

  let {
    t,
    show = false,
    state = {},
    onDownload = () => {},
    onRestart = () => {},
    onClose = () => {},
  } = $props();

  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }
</script>

<Modal {show} onclose={onClose}>
  <div class="bg-white rounded-xl border border-gray-100 w-full max-w-lg p-6 shadow-2xl">

    {#if state.status === 'downloading'}
      <!-- Phase 2: Downloading -->
      <h3 class="text-[15px] font-semibold text-gray-900 mb-4">{t.updateDownloading || '正在下载...'}</h3>
      <div class="mb-3">
        <div class="w-full h-2.5 bg-gray-100 rounded-full overflow-hidden">
          <div
            class="h-full bg-emerald-500 rounded-full transition-all duration-300"
            style="width: {Math.min(state.progress || 0, 100)}%"
          ></div>
        </div>
      </div>
      <p class="text-[13px] text-gray-500 mb-4">
        {Math.round(state.progress || 0)}% — {formatBytes(state.downloaded || 0)} / {formatBytes(state.assetSize || 0)}
      </p>
      <div class="flex justify-end">
        <button
          class="px-4 py-2 text-[13px] text-gray-500 hover:text-gray-700 transition-colors cursor-pointer"
          onclick={onClose}
        >
          {t.updateCancel || '取消'}
        </button>
      </div>

    {:else if state.status === 'ready'}
      <!-- Phase 3: Ready to restart -->
      <div class="flex flex-col items-center text-center py-4">
        <div class="w-12 h-12 rounded-full bg-emerald-50 flex items-center justify-center mb-4">
          <svg class="w-6 h-6 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        </div>
        <p class="text-[14px] text-gray-900 font-medium mb-1">{state.latestVer}</p>
        <p class="text-[13px] text-gray-500 mb-6">{t.updateReady || '下载完成，重启以应用更新'}</p>
        <div class="flex items-center gap-3">
          <button
            class="px-5 py-2 text-[13px] text-gray-500 hover:text-gray-700 transition-colors cursor-pointer"
            onclick={onClose}
          >
            {t.updateRestartLater || '稍后重启'}
          </button>
          <button
            class="px-5 py-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[13px] font-medium rounded-lg transition-colors cursor-pointer"
            onclick={onRestart}
          >
            {t.updateRestartNow || '立即重启'}
          </button>
        </div>
      </div>

    {:else if state.status === 'error'}
      <!-- Error state -->
      <h3 class="text-[15px] font-semibold text-gray-900 mb-2">{t.updateError || '更新失败'}</h3>
      <p class="text-[13px] text-red-500 mb-4">{state.error}</p>
      <div class="flex justify-end gap-3">
        <button
          class="px-4 py-2 text-[13px] text-gray-500 hover:text-gray-700 transition-colors cursor-pointer"
          onclick={onClose}
        >
          {t.updateLater || '稍后'}
        </button>
        <button
          class="px-4 py-2 border border-gray-300 text-[13px] text-gray-700 hover:bg-gray-50 rounded-lg transition-colors cursor-pointer"
          onclick={() => BrowserOpenURL(state.downloadURL || 'https://github.com/wgpsec/redc/releases')}
        >
          {t.updateGoToGithub || '前往 GitHub'}
        </button>
      </div>

    {:else}
      <!-- Phase 1: Update available (default) -->
      <h3 class="text-[15px] font-semibold text-gray-900 mb-4">{t.updateAvailable || '发现新版本'}</h3>

      <!-- Version comparison -->
      <div class="flex items-center gap-3 mb-4">
        <span class="text-[13px] text-gray-400 bg-gray-50 px-2.5 py-1 rounded">{state.currentVer}</span>
        <svg class="w-4 h-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
        </svg>
        <span class="text-[13px] text-emerald-600 font-medium bg-emerald-50 px-2.5 py-1 rounded">{state.latestVer}</span>
      </div>

      <!-- Release Notes -->
      {#if state.releaseNotes}
        <div class="mb-4">
          <p class="text-[12px] font-medium text-gray-500 mb-1.5">{t.updateReleaseNotes || '更新日志'}</p>
          <div class="max-h-[240px] overflow-y-auto bg-gray-50 rounded-lg p-3 text-[12px] text-gray-600 leading-relaxed whitespace-pre-wrap">
            {state.releaseNotes}
          </div>
        </div>
      {/if}

      <!-- Actions -->
      <div class="flex items-center justify-end gap-3">
        <button
          class="px-4 py-2 text-[13px] text-gray-500 hover:text-gray-700 transition-colors cursor-pointer"
          onclick={onClose}
        >
          {t.updateLater || '稍后'}
        </button>
        <button
          class="px-4 py-2 border border-gray-300 text-[13px] text-gray-700 hover:bg-gray-50 rounded-lg transition-colors cursor-pointer"
          onclick={() => BrowserOpenURL(state.downloadURL || 'https://github.com/wgpsec/redc/releases')}
        >
          {t.updateGoToGithub || '前往 GitHub'}
        </button>
        {#if state.assetURL}
          <button
            class="px-5 py-2 bg-emerald-500 hover:bg-emerald-600 text-white text-[13px] font-medium rounded-lg transition-colors cursor-pointer"
            onclick={onDownload}
          >
            {t.updateAutoDownload || '自动下载安装'}
          </button>
        {/if}
      </div>
    {/if}
  </div>
</Modal>
