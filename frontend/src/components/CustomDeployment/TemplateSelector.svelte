<script>
  let { t, templates = [], selectedTemplate = null, onSelect = () => {} } = $props();
  
  let searchQuery = $state('');

  // Filtered templates based on search
  let filteredTemplates = $derived(() => {
    let result = templates;

    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(tmpl => 
        tmpl.name.toLowerCase().includes(query) ||
        (tmpl.description && tmpl.description.toLowerCase().includes(query))
      );
    }

    return result;
  });

  function handleSelect(template) {
    onSelect(template);
  }

  function clearSelection() {
    onSelect(null);
  }
</script>

<div class="bg-white rounded-xl border border-gray-100 p-5">
  <div class="flex items-center justify-between mb-4">
    <h2 class="text-[15px] font-semibold text-gray-900">
      {t.selectTemplate || '选择模板'}
    </h2>
    {#if selectedTemplate}
      <button
        class="text-[12px] text-blue-600 hover:text-blue-800 underline"
        onclick={clearSelection}
      >
        {t.clearSelection || '清除选择'}
      </button>
    {/if}
  </div>

  <!-- Search and Filter -->
  <div class="flex gap-3 mb-4">
    <div class="flex-1">
      <input
        type="text"
        placeholder={t.search || '搜索模板...'}
        class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
        bind:value={searchQuery}
      />
    </div>
  </div>

  <!-- Template List -->
  <div class="space-y-2">
    {#if templates.length === 0}
      <!-- Empty state with guidance -->
      <div class="text-center py-12 px-6">
        <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
        <h3 class="text-[15px] font-semibold text-gray-900 mb-2">
          {t.noBaseTemplates || '暂无基础模板'}
        </h3>
        <p class="text-[13px] text-gray-600 mb-6 max-w-md mx-auto">
          {t.noBaseTemplatesDesc || '自定义部署需要基础模板。基础模板是可以自定义配置的通用模板，支持选择不同的云厂商、地域和实例规格。'}
        </p>
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 text-left max-w-lg mx-auto">
          <h4 class="text-[13px] font-semibold text-blue-900 mb-2">
            {t.howToCreateBaseTemplate || '如何创建基础模板？'}
          </h4>
          <ol class="text-[12px] text-blue-800 space-y-2 list-decimal list-inside">
            <li>{t.baseTemplateStep1 || '在模板仓库中拉取一个模板到本地'}</li>
            <li>{t.baseTemplateStep2 || '在本地模板管理中，编辑模板的 case.json 文件'}</li>
            <li>{t.baseTemplateStep3 || '添加 "template": "base" 字段'}</li>
            <li>{t.baseTemplateStep4 || '添加 "supported_providers": ["alicloud", "tencentcloud"] 等字段'}</li>
            <li>{t.baseTemplateStep5 || '在 variables.tf 中定义可配置的变量（provider, region, instance_type 等）'}</li>
            <li>{t.baseTemplateStep6 || '刷新此页面即可看到基础模板'}</li>
          </ol>
        </div>
        <div class="mt-6 flex items-center justify-center gap-3">
          <button
            class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
            onclick={() => window.dispatchEvent(new CustomEvent('switchTab', { detail: 'localTemplates' }))}
          >
            {t.goToLocalTemplates || '前往本地模板管理'}
          </button>
          <button
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
            onclick={() => window.dispatchEvent(new CustomEvent('switchTab', { detail: 'registry' }))}
          >
            {t.goToRegistry || '前往模板仓库'}
          </button>
        </div>
      </div>
    {:else if filteredTemplates().length === 0}
      <div class="text-center py-8 text-gray-400 text-[13px]">
        {t.noMatch || '未找到匹配的模板'}
      </div>
    {:else}
      {#each filteredTemplates() as template}
        <button
          class="w-full text-left p-4 rounded-lg border transition-all {selectedTemplate?.name === template.name ? 'border-gray-900 bg-gray-50' : 'border-gray-100 hover:border-gray-200 hover:bg-gray-50'}"
          onclick={() => handleSelect(template)}
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <div class="flex items-center gap-2">
                <h3 class="text-[14px] font-medium text-gray-900">{template.name}</h3>
                {#if template.isBase}
                  <span class="px-2 py-0.5 text-[10px] font-medium bg-blue-100 text-blue-700 rounded">
                    {t.baseTemplate || '基础模板'}
                  </span>
                {:else}
                  <span class="px-2 py-0.5 text-[10px] font-medium bg-gray-100 text-gray-700 rounded">
                    {t.predefined || '预定义'}
                  </span>
                {/if}
                {#if template.version}
                  <span class="text-[11px] text-gray-400">v{template.version}</span>
                {/if}
              </div>
              {#if template.description}
                <p class="text-[12px] text-gray-600 mt-1">{template.description}</p>
              {/if}
              {#if template.providers && template.providers.length > 0}
                <div class="flex items-center gap-2 mt-2">
                  <span class="text-[11px] text-gray-500">{t.supportedProviders || '支持云厂商'}:</span>
                  <div class="flex gap-1">
                    {#each template.providers as provider}
                      <span class="px-1.5 py-0.5 text-[10px] bg-gray-100 text-gray-600 rounded">
                        {provider}
                      </span>
                    {/each}
                  </div>
                </div>
              {/if}
            </div>
            {#if selectedTemplate?.name === template.name}
              <svg class="w-5 h-5 text-gray-900 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            {/if}
          </div>
        </button>
      {/each}
    {/if}
  </div>
</div>

<style>
  /* Component-specific styles if needed */
</style>
