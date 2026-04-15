<script lang="ts">
  import { SaveConfigTemplate } from '../../../wailsjs/go/main/App.js';
  import Modal from '../UI/Modal.svelte';

  let { t, show = false, config, onClose, onSaved } = $props();
  
  // State
  let templateName = $state('');
  let isSaving = $state(false);
  let error = $state('');
  let nameError = $state('');

  // Reset state when dialog opens
  $effect(() => {
    if (show) {
      templateName = '';
      error = '';
      nameError = '';
      isSaving = false;
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
    nameError = validateTemplateName(templateName);
  }

  async function handleSave() {
    // Validate template name
    const validationError = validateTemplateName(templateName);
    if (validationError) {
      nameError = validationError;
      return;
    }

    isSaving = true;
    error = '';
    
    try {
      // Convert config to backend format (camelCase to snake_case)
      const configToSave: any = {
        name: config.name || '',
        template_name: config.templateName || '',
        provider: config.provider || '',
        region: config.region || '',
        instance_type: config.instanceType || '', // Convert camelCase to snake_case
        userdata: config.userdata || '',
        variables: config.variables || {},
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      
      await SaveConfigTemplate(templateName.trim(), configToSave);
      
      // Notify parent component
      if (onSaved) {
        onSaved(templateName.trim());
      }
      
      // Close dialog
      if (onClose) {
        onClose();
      }
    } catch (e) {
      error = e.message || String(e);
    } finally {
      isSaving = false;
    }
  }

  function handleCancel() {
    if (onClose) {
      onClose();
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !isSaving) {
      handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  }
</script>

<Modal show={show} onclose={handleCancel}>
    <div class="bg-white rounded-lg border border-gray-200 max-w-md w-full mx-4" onclick={(e) => e.stopPropagation()}>
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <h3 class="text-[15px] font-semibold text-gray-900">
          {t.saveConfigTemplate || '保存配置模板'}
        </h3>
        <button
          class="text-gray-400 hover:text-gray-600 transition-colors"
          onclick={handleCancel}
          disabled={isSaving}
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

        <div class="space-y-4">
          <!-- Template Name Input -->
          <div>
            <label for="template-name" class="block text-[13px] font-medium text-gray-700 mb-1.5">
              {t.templateName || '模板名称'}
              <span class="text-red-500">*</span>
            </label>
            <!-- svelte-ignore a11y_autofocus -->
            <input
              id="template-name"
              type="text"
              class="w-full px-3 py-2 text-[13px] border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              class:border-red-300={nameError}
              class:focus:ring-red-500={nameError}
              placeholder={t.templateNamePlaceholder || '输入模板名称'}
              bind:value={templateName}
              oninput={handleNameInput}
              onkeydown={handleKeydown}
              disabled={isSaving}
              autofocus
            />
            {#if nameError}
              <p class="mt-1.5 text-[12px] text-red-600">{nameError}</p>
            {/if}
          </div>

          <!-- Info Message -->
          <div class="flex items-start gap-2 p-3 bg-blue-50 border border-blue-100 rounded-lg">
            <svg class="w-4 h-4 text-blue-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p class="text-[12px] text-blue-700">
              {t.saveTemplateInfo || '保存后可以在配置模板管理器中快速加载此配置'}
            </p>
          </div>

          <!-- Config Summary -->
          <div class="p-3 bg-gray-50 border border-gray-100 rounded-lg">
            <p class="text-[12px] font-medium text-gray-700 mb-2">
              {t.configSummary || '配置摘要'}
            </p>
            <div class="space-y-1 text-[12px] text-gray-600">
              {#if config.name}
                <div class="flex items-center gap-2">
                  <span class="text-gray-500">{t.deploymentName || '部署名称'}:</span>
                  <span class="font-medium text-gray-900">{config.name}</span>
                </div>
              {/if}
              {#if config.provider}
                <div class="flex items-center gap-2">
                  <span class="text-gray-500">{t.provider || '云厂商'}:</span>
                  <span class="font-medium text-gray-900">{config.provider}</span>
                </div>
              {/if}
              {#if config.region}
                <div class="flex items-center gap-2">
                  <span class="text-gray-500">{t.region || '地域'}:</span>
                  <span class="font-medium text-gray-900">{config.region}</span>
                </div>
              {/if}
              {#if config.instanceType}
                <div class="flex items-center gap-2">
                  <span class="text-gray-500">{t.instanceType || '实例规格'}:</span>
                  <span class="font-medium text-gray-900">{config.instanceType}</span>
                </div>
              {/if}
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-3 px-6 py-4 bg-gray-50 rounded-b-lg">
        <button
          class="px-4 py-2 text-[13px] font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors"
          onclick={handleCancel}
          disabled={isSaving}
        >
          {t.cancel || '取消'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 hover:bg-blue-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          onclick={handleSave}
          disabled={isSaving || !!nameError || !templateName.trim()}
        >
          {#if isSaving}
            <span class="flex items-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {t.saving || '保存中...'}
            </span>
          {:else}
            {t.save || '保存'}
          {/if}
        </button>
      </div>
    </div>
</Modal>

<style>
  /* Component-specific styles */
</style>
