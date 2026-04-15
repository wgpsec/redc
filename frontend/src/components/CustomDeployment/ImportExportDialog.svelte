<script lang="ts">
  import { ExportConfigTemplate, ImportConfigTemplate } from '../../../wailsjs/go/main/App.js';
  import { selectFile, selectSaveFile } from '../../lib/file-dialog.js';
  import Modal from '../UI/Modal.svelte';

  let { t, show = false, mode = 'export', templateName = '', onClose, onImported } = $props();
  
  // State
  let isProcessing = $state(false);
  let error = $state('');
  let success = $state('');
  let selectedFile = $state('');
  let importTemplateName = $state('');
  let nameError = $state('');

  // Reset state when dialog opens
  $effect(() => {
    if (show) {
      error = '';
      success = '';
      selectedFile = '';
      importTemplateName = '';
      nameError = '';
      isProcessing = false;
    }
  });

  function validateTemplateName(name) {
    if (!name || name.trim() === '') {
      return t.templateNameRequired || '模板名称不能为空';
    }
    
    // Check for invalid characters
    const invalidChars = /[<>:"/\\|?*]/;
    if (invalidChars.test(name)) {
      return t.templateNameInvalidChars || '模板名称不能包含特殊字符 < > : " / \\ | ? *';
    }
    
    // Check length
    if (name.length > 100) {
      return t.templateNameTooLong || '模板名称不能超过100个字符';
    }
    
    return '';
  }

  function handleNameInput() {
    nameError = validateTemplateName(importTemplateName);
  }

  async function handleSelectFile() {
    error = '';
    try {
      const file = await selectFile(t.selectConfigFile || '选择配置文件');
      if (file) {
        selectedFile = file;
      }
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleExport() {
    error = '';
    success = '';
    isProcessing = true;
    
    try {
      // Open save file dialog
      const defaultFilename = `${templateName}.json`;
      const savePath = await selectSaveFile(
        t.exportConfigTemplate || '导出配置模板',
        defaultFilename
      );
      
      if (!savePath) {
        isProcessing = false;
        return;
      }
      
      // Export the template
      await ExportConfigTemplate(templateName, savePath);
      
      success = t.exportSuccess || '配置模板导出成功';
      
      // Close dialog after a short delay
      setTimeout(() => {
        if (onClose) {
          onClose();
        }
      }, 1500);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      isProcessing = false;
    }
  }

  async function handleImport() {
    // Validate template name
    const validationError = validateTemplateName(importTemplateName);
    if (validationError) {
      nameError = validationError;
      return;
    }

    if (!selectedFile) {
      error = t.pleaseSelectFile || '请选择要导入的文件';
      return;
    }

    error = '';
    success = '';
    isProcessing = true;
    
    try {
      await ImportConfigTemplate(importTemplateName.trim(), selectedFile);
      
      success = t.importSuccess || '配置模板导入成功';
      
      // Notify parent component
      if (onImported) {
        onImported(importTemplateName.trim());
      }
      
      // Close dialog after a short delay
      setTimeout(() => {
        if (onClose) {
          onClose();
        }
      }, 1500);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      isProcessing = false;
    }
  }

  function handleCancel() {
    if (onClose) {
      onClose();
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      handleCancel();
    }
  }
</script>

<Modal show={show} onclose={handleCancel}>
    <div class="bg-white rounded-lg border border-gray-200 max-w-md w-full mx-4" onclick={(e) => e.stopPropagation()} onkeydown={handleKeydown} role="dialog" aria-modal="true">
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <h3 class="text-[15px] font-semibold text-gray-900">
          {mode === 'export' ? (t.exportConfigTemplate || '导出配置模板') : (t.importConfigTemplate || '导入配置模板')}
        </h3>
        <button
          class="text-gray-400 hover:text-gray-600 transition-colors"
          onclick={handleCancel}
          disabled={isProcessing}
          aria-label={t.close}
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Content -->
      <div class="p-6">
        {#if error}
          <div class="mb-4 flex items-center gap-3 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
            <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            <span class="text-[12px] text-red-700 flex-1">{error}</span>
            <button 
              class="text-red-400 hover:text-red-600" 
              onclick={() => error = ''}
              aria-label={t.closeError}
            >
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        {/if}

        {#if success}
          <div class="mb-4 flex items-center gap-3 px-3 py-2 bg-green-50 border border-green-100 rounded-lg">
            <svg class="w-4 h-4 text-green-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span class="text-[12px] text-green-700 flex-1">{success}</span>
          </div>
        {/if}

        <div class="space-y-4">
          {#if mode === 'export'}
            <!-- Export Mode -->
            <div class="p-4 bg-gray-50 border border-gray-100 rounded-lg">
              <div class="flex items-start gap-3">
                <svg class="w-5 h-5 text-gray-400 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
                </svg>
                <div class="flex-1">
                  <p class="text-[13px] font-medium text-gray-900 mb-1">
                    {templateName}
                  </p>
                  <p class="text-[12px] text-gray-600">
                    {t.exportDescription || '将此配置模板导出为 JSON 文件，可以在其他设备上导入使用'}
                  </p>
                </div>
              </div>
            </div>
          {:else}
            <!-- Import Mode -->
            <div>
              <label for="import-template-name" class="block text-[13px] font-medium text-gray-700 mb-1.5">
                {t.templateName || '模板名称'}
                <span class="text-red-500">*</span>
              </label>
              <input
                id="import-template-name"
                type="text"
                class="w-full px-3 py-2 text-[13px] border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                class:border-red-300={nameError}
                class:focus:ring-red-500={nameError}
                placeholder={t.templateNamePlaceholder || '输入模板名称'}
                bind:value={importTemplateName}
                oninput={handleNameInput}
                disabled={isProcessing}
              />
              {#if nameError}
                <p class="mt-1.5 text-[12px] text-red-600">{nameError}</p>
              {/if}
            </div>

            <div>
              <label for="selectedFile" class="block text-[13px] font-medium text-gray-700 mb-1.5">
                {t.selectFile || '选择文件'}
                <span class="text-red-500">*</span>
              </label>
              <div class="flex items-center gap-2">
                <input
                  id="selectedFile"
                  type="text"
                  class="flex-1 px-3 py-2 text-[13px] border border-gray-300 rounded-lg bg-gray-50"
                  placeholder={t.noFileSelected || '未选择文件'}
                  value={selectedFile}
                  readonly
                  disabled={isProcessing}
                />
                <button
                  class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
                  onclick={handleSelectFile}
                  disabled={isProcessing}
                >
                  {t.browse || '浏览'}
                </button>
              </div>
            </div>

            <div class="flex items-start gap-2 p-3 bg-blue-50 border border-blue-100 rounded-lg">
              <svg class="w-4 h-4 text-blue-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p class="text-[12px] text-blue-700">
                {t.importFromJson || '从 JSON 文件导入配置模板。如果模板名称已存在，将会覆盖原有配置。'}
              </p>
            </div>
          {/if}
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-3 px-6 py-4 bg-gray-50 rounded-b-lg">
        <button
          class="px-4 py-2 text-[13px] font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors"
          onclick={handleCancel}
          disabled={isProcessing}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 hover:bg-blue-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          onclick={mode === 'export' ? handleExport : handleImport}
          disabled={isProcessing || (mode === 'import' && (!importTemplateName.trim() || !selectedFile || !!nameError))}
        >
          {#if isProcessing}
            <span class="flex items-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {mode === 'export' ? (t.exporting || '导出中...') : (t.importing || '导入中...')}
            </span>
          {:else}
            {mode === 'export' ? (t.export || '导出') : (t.import || '导入')}
          {/if}
        </button>
      </div>
    </div>
</Modal>

<style>
  /* Component-specific styles */
</style>
