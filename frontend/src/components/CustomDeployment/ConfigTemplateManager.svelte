<script lang="ts">
  import { onMount } from 'svelte';
  import { ListConfigTemplates, LoadConfigTemplate, DeleteConfigTemplate } from '../../../wailsjs/go/main/App.js';

  let { t, onLoadTemplate } = $props();
  
  // State
  let templates = $state([]);
  let isLoading = $state(false);
  let error = $state('');
  let selectedTemplate = $state(null);
  let showDeleteConfirm = $state(false);
  let templateToDelete = $state(null);
  let isDeleting = $state(false);

  onMount(async () => {
    await loadTemplates();
  });

  async function loadTemplates() {
    isLoading = true;
    error = '';
    try {
      const templateNames = await ListConfigTemplates();
      templates = templateNames || [];
    } catch (e) {
      error = e.message || String(e);
      templates = [];
    } finally {
      isLoading = false;
    }
  }

  async function handleLoadTemplate(templateName) {
    error = '';
    try {
      const config = await LoadConfigTemplate(templateName);
      if (config && onLoadTemplate) {
        onLoadTemplate(config, templateName);
      }
    } catch (e) {
      error = `加载配置模板失败: ${e.message || String(e)}`;
    }
  }

  function confirmDelete(templateName) {
    templateToDelete = templateName;
    showDeleteConfirm = true;
  }

  function cancelDelete() {
    templateToDelete = null;
    showDeleteConfirm = false;
  }

  async function handleDelete() {
    if (!templateToDelete) return;
    
    isDeleting = true;
    error = '';
    
    try {
      await DeleteConfigTemplate(templateToDelete);
      // Reload templates list
      await loadTemplates();
      showDeleteConfirm = false;
      templateToDelete = null;
    } catch (e) {
      error = `删除配置模板失败: ${e.message || String(e)}`;
    } finally {
      isDeleting = false;
    }
  }

  function handleRefresh() {
    loadTemplates();
  }
</script>

<div class="bg-white rounded-lg border border-gray-100 shadow-sm">
  <!-- Header -->
  <div class="flex items-center justify-between px-4 py-3 border-b border-gray-100">
    <h3 class="text-[14px] font-semibold text-gray-900">
      {t.savedConfigTemplates || '已保存的配置模板'}
    </h3>
    <button
      class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-50 rounded transition-colors"
      onclick={handleRefresh}
      disabled={isLoading}
      title={t.refresh || '刷新'}
    >
      <svg class="w-4 h-4" class:animate-spin={isLoading} fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
    </button>
  </div>

  <!-- Error Message -->
  {#if error}
    <div class="mx-4 mt-3 flex items-center gap-3 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[12px] text-red-700 flex-1">{error}</span>
      <button 
        class="text-red-400 hover:text-red-600" 
        onclick={() => error = ''}
        aria-label="关闭错误"
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Content -->
  <div class="p-4">
    {#if isLoading}
      <div class="flex items-center justify-center py-8">
        <div class="w-5 h-5 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
      </div>
    {:else if templates.length === 0}
      <div class="text-center py-8">
        <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
        </svg>
        <p class="text-[13px] text-gray-500">
          {t.noSavedTemplates || '暂无保存的配置模板'}
        </p>
        <p class="text-[12px] text-gray-400 mt-1">
          {t.saveTemplateHint || '完成配置后可以保存为模板以便重复使用'}
        </p>
      </div>
    {:else}
      <div class="space-y-2">
        {#each templates as templateName}
          <div class="flex items-center gap-3 p-3 border border-gray-100 rounded-lg hover:border-gray-200 hover:bg-gray-50 transition-colors group">
            <div class="flex-shrink-0">
              <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-[13px] font-medium text-gray-900 truncate">
                {templateName}
              </p>
            </div>
            <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-blue-600 hover:text-blue-700 hover:bg-blue-50 rounded transition-colors"
                onclick={() => handleLoadTemplate(templateName)}
              >
                {t.load || '加载'}
              </button>
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-red-600 hover:text-red-700 hover:bg-red-50 rounded transition-colors"
                onclick={() => confirmDelete(templateName)}
              >
                {t.delete || '删除'}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- Delete Confirmation Modal -->
{#if showDeleteConfirm}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50" onclick={cancelDelete}>
    <div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="p-6">
        <div class="flex items-start gap-4">
          <div class="flex-shrink-0">
            <svg class="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
          </div>
          <div class="flex-1">
            <h3 class="text-[15px] font-semibold text-gray-900 mb-2">
              {t.confirmDelete || '确认删除'}
            </h3>
            <p class="text-[13px] text-gray-600">
              {t.confirmDeleteMessage || '确定要删除配置模板'} <span class="font-medium text-gray-900">{templateToDelete}</span> {t.questionMark || '吗？'}
            </p>
            <p class="text-[12px] text-gray-500 mt-2">
              {t.deleteWarning || '此操作无法撤销。'}
            </p>
          </div>
        </div>
      </div>
      <div class="flex items-center justify-end gap-3 px-6 py-4 bg-gray-50 rounded-b-lg">
        <button
          class="px-4 py-2 text-[13px] font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors"
          onclick={cancelDelete}
          disabled={isDeleting}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 hover:bg-red-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          onclick={handleDelete}
          disabled={isDeleting}
        >
          {#if isDeleting}
            <span class="flex items-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {t.deleting || '删除中...'}
            </span>
          {:else}
            {t.delete || '删除'}
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Component-specific styles */
</style>
