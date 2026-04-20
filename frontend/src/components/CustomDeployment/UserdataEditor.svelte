<script lang="ts">
  import { onMount } from 'svelte';
  import { loadUserdataTemplates, getTemplatesByType } from '../../lib/userdataTemplates.js';
  
  // Props
  let { 
    t, 
    value = '', 
    onChange = () => {},
    language = 'bash',
    disabled = false
  } = $props();

  // State
  let selectedLanguage = $state('bash');
  let showTemplates = $state(false);
  let templates = $state([]);
  let templatesLoading = $state(true);

  onMount(async () => {
    templates = await loadUserdataTemplates();
    templatesLoading = false;
  });

  function handleChange(event) {
    const newValue = event.currentTarget.value;
    onChange(newValue);
  }

  function handleLanguageChange(newLanguage) {
    selectedLanguage = newLanguage;
  }

  function applyTemplate(template) {
    onChange(template.script);
    showTemplates = false;
  }

  function clearContent() {
    onChange('');
  }

  // Get current templates based on selected language
  let currentTemplates = $derived(() => {
    return getTemplatesByType(templates, selectedLanguage);
  });

  // Group templates by category
  let groupedTemplates = $derived(() => {
    const groups: Record<string, any[]> = {};
    const current = currentTemplates();
    for (const template of current) {
      const cat = template.category || 'other';
      if (!groups[cat]) {
        groups[cat] = [];
      }
      groups[cat].push(template);
    }
    return groups;
  });

  // Category display names
  const categoryNames: Record<string, string> = $derived({
    basic: t.userdataCatBasic || '基础环境',
    ai: t.userdataCatAI || 'AI 应用',
    security: t.userdataCatSecurity || '安全工具',
    vulhub: t.userdataCatVulhub || '漏洞环境',
    other: t.userdataCatOther || '其他'
  });

  // Toggle category expanded state
  let expandedCategories = $state<Record<string, boolean>>({});

  // Character count
  let charCount = $derived(() => {
    return value.length;
  });
</script>

<div class="userdata-editor">
  <div class="flex items-center justify-between mb-1.5">
    <label for="userdata-textarea" class="block text-[12px] font-medium text-gray-700">
      {t.userdata || 'Userdata'}
      <span class="text-gray-400 ml-1">({t.optional || '可选'})</span>
    </label>
    
    <div class="flex items-center gap-2">
      <!-- Language selector -->
      <div class="flex gap-1 bg-gray-100 rounded-lg p-0.5">
        <button
          class="px-2 py-1 text-[10px] font-medium rounded transition-colors {selectedLanguage === 'bash' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}"
          onclick={() => handleLanguageChange('bash')}
        >
          Bash
        </button>
        <button
          class="px-2 py-1 text-[10px] font-medium rounded transition-colors {selectedLanguage === 'powershell' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'}"
          onclick={() => handleLanguageChange('powershell')}
        >
          PowerShell
        </button>
      </div>

      <!-- Templates button -->
      <button
        class="px-2 py-1 text-[10px] font-medium text-blue-600 hover:text-blue-800 bg-blue-50 hover:bg-blue-100 rounded transition-colors"
        onclick={() => showTemplates = !showTemplates}
      >
        {t.templates || '模板'}
      </button>

      <!-- Clear button -->
      {#if value}
        <button
          class="px-2 py-1 text-[10px] font-medium text-red-600 hover:text-red-800 bg-red-50 hover:bg-red-100 rounded transition-colors"
          onclick={clearContent}
        >
          {t.clear || '清空'}
        </button>
      {/if}
    </div>
  </div>

  <!-- Templates dropdown -->
  {#if showTemplates}
    <div class="mb-2 p-3 bg-gray-50 border border-gray-100 rounded-lg max-h-80 overflow-y-auto">
      <p class="text-[11px] font-medium text-gray-700 mb-2">
        {t.selectTemplate || '选择模板'}:
      </p>
      <div class="space-y-2">
        {#each Object.entries(groupedTemplates()) as [category, categoryTemplates]}
          <div class="border border-gray-100 rounded-lg overflow-hidden">
            <button
              class="w-full flex items-center justify-between px-3 py-2 bg-gray-100 hover:bg-gray-200 transition-colors"
              onclick={() => expandedCategories[category] = !expandedCategories[category]}
            >
              <span class="text-[12px] font-medium text-gray-700">
                {categoryNames[category] || category}
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
                    class="w-full text-left px-3 py-2 text-[12px] bg-white hover:bg-blue-50 border border-gray-100 hover:border-blue-300 rounded transition-colors"
                    onclick={() => applyTemplate(template)}
                  >
                    <span class="font-medium text-gray-900">{template.nameZh}</span>
                    <span class="text-gray-500 ml-2">({template.name})</span>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Code editor textarea -->
  <textarea
    id="userdata-textarea"
    class="w-full h-64 px-3 py-2 text-[12px] font-mono bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow resize-y"
    placeholder={selectedLanguage === 'bash'
      ? (t.userdataPlaceholderBash || '#!/bin/bash\n# 在此输入初始化脚本...\n\napt-get update -y\napt-get install -y nginx')
      : (t.userdataPlaceholderPowershell || '<powershell>\n# 在此输入初始化脚本...\n\nInstall-WindowsFeature -name Web-Server\n</powershell>')}
    {disabled}
    {value}
    oninput={handleChange}
  ></textarea>

  <!-- Footer info -->
  <div class="flex items-center justify-between mt-1.5">
    <p class="text-[10px] text-gray-500">
      {t.userdataHint || '实例启动时自动执行的脚本'}
    </p>
    <p class="text-[10px] text-gray-500">
      {charCount()} {t.characters || '字符'}
    </p>
  </div>

  <!-- Syntax hints -->
  <div class="mt-2 p-2 bg-blue-50 border border-blue-100 rounded-lg">
    <p class="text-[10px] font-medium text-blue-900 mb-1">
      {t.tips || '提示'}:
    </p>
    {#if selectedLanguage === 'bash'}
      <ul class="text-[10px] text-blue-700 space-y-0.5 ml-3">
        <li>• {t.bashTip1 || '脚本必须以 #!/bin/bash 开头'}</li>
        <li>• {t.bashTip2 || '使用 -y 参数避免交互式确认'}</li>
        <li>• {t.bashTip3 || '可以使用 echo 输出日志到 /var/log/cloud-init-output.log'}</li>
      </ul>
    {:else}
      <ul class="text-[10px] text-blue-700 space-y-0.5 ml-3">
        <li>• {t.psTip1 || '脚本必须包含在 <powershell></powershell> 标签中'}</li>
        <li>• {t.psTip2 || '使用 Write-Host 输出日志'}</li>
        <li>• {t.psTip3 || '某些云厂商可能需要 base64 编码'}</li>
      </ul>
    {/if}
  </div>
</div>

<style>
  textarea {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
    line-height: 1.5;
    tab-size: 2;
  }

  textarea:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* Custom scrollbar for textarea */
  textarea::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  textarea::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 4px;
  }

  textarea::-webkit-scrollbar-thumb {
    background: #888;
    border-radius: 4px;
  }

  textarea::-webkit-scrollbar-thumb:hover {
    background: #555;
  }
</style>
