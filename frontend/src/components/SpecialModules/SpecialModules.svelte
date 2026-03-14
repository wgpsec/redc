<script>
  import { onMount } from 'svelte';
  import { loadUserdataTemplates } from '../../lib/userdataTemplates.js';
  
  let { t, onTabChange } = $props();
  let templates = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let activeCategory = $state('all');
  let selectedTemplate = $state(null);
  let copied = $state(false);

  onMount(async () => {
    templates = await loadUserdataTemplates();
    loading = false;
  });

  const categories = $derived(() => {
    const cats = new Map();
    cats.set('all', { key: 'all', label: t.userdataAll || '全部', count: templates.length });
    for (const tmpl of templates) {
      const cat = tmpl.category || 'other';
      if (!cats.has(cat)) {
        cats.set(cat, { key: cat, label: categoryLabel(cat), count: 0 });
      }
      cats.get(cat).count++;
    }
    return [...cats.values()];
  });

  const filteredTemplates = $derived(() => {
    let list = templates;
    if (activeCategory !== 'all') {
      list = list.filter(t => t.category === activeCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(t =>
        (t.nameZh || '').toLowerCase().includes(q) ||
        t.name.toLowerCase().includes(q) ||
        (t.description || '').toLowerCase().includes(q) ||
        (t.cveId || '').toLowerCase().includes(q)
      );
    }
    return list;
  });

  function categoryLabel(cat) {
    const map = {
      vulhub: 'Vulhub',
      c2: 'C2',
      ai: 'AI',
      basic: t.userdataCatBasic || '基础环境',
      security: t.userdataCatSecurity || '安全工具',
      other: t.userdataCatOther || '其他'
    };
    return map[cat] || cat;
  }

  // SVG path data for category icons
  const categoryIconPaths = {
    vulhub: 'M12 12.75c1.148 0 2.278.08 3.383.237 1.037.146 1.866.966 1.866 2.013 0 3.728-2.35 6.75-5.25 6.75S6.75 18.728 6.75 15c0-1.046.83-1.867 1.866-2.013A24.204 24.204 0 0112 12.75zm0 0c2.883 0 5.647.508 8.207 1.44a23.91 23.91 0 01-1.152 6.135M12 12.75c-2.883 0-5.647.508-8.208 1.44.125 2.104.745 4.2 1.153 6.135M12 12.75a2.25 2.25 0 002.248-2.354M12 12.75a2.25 2.25 0 01-2.248-2.354M12 8.25c.995 0 1.971-.08 2.922-.236.403-.066.74-.358.795-.762a3.778 3.778 0 00-.399-2.25M12 8.25c-.995 0-1.97-.08-2.922-.236a1.023 1.023 0 01-.795-.762 3.778 3.778 0 01.399-2.25M12 2.25c-1.135 0-2.15.752-2.678 1.916-.164.36-.308.73-.43 1.106C9.758 4.484 10.85 4 12 4c1.15 0 2.242.484 3.108 1.272a8.337 8.337 0 00-.43-1.106C14.149 3.002 13.135 2.25 12 2.25z',
    c2: 'M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z',
    ai: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z',
    basic: 'M21 7.5l-2.25-1.313M21 7.5v2.25m0-2.25l-2.25 1.313M3 7.5l2.25-1.313M3 7.5l2.25 1.313M3 7.5v2.25m9 3l2.25-1.313M12 12.75l-2.25-1.313M12 12.75V15m0 6.75l2.25-1.313M12 21.75V19.5m0 2.25l-2.25-1.313m0-16.875L12 2.25l2.25 1.313M21 14.25v2.25l-2.25 1.313m-13.5 0L3 16.5v-2.25',
    security: 'M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z',
    other: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z',
  };

  function getCategoryIconPath(cat) {
    return categoryIconPaths[cat] || categoryIconPaths.other;
  }

  function selectTemplate(tmpl) {
    selectedTemplate = tmpl;
    copied = false;
  }

  async function copyScript() {
    if (!selectedTemplate) return;
    try {
      await navigator.clipboard.writeText(selectedTemplate.script);
      copied = true;
      setTimeout(() => copied = false, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }
</script>

<div class="h-full flex flex-col space-y-4">
  {#if loading}
    <div class="flex items-center justify-center gap-2 h-32">
      <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
      <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
    </div>
  {:else if templates.length === 0}
    <!-- Empty state -->
    <div class="bg-white rounded-xl border border-gray-100 p-8 text-center">
      <svg class="w-10 h-10 mx-auto text-gray-200 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
      </svg>
      <p class="text-[13px] text-gray-600 mb-1">{t.noUserdataTemplatesHint || '暂无 Userdata 脚本模板'}</p>
      <p class="text-[12px] text-gray-400 mb-4">{t.noUserdataHint2 || '请先从模板仓库拉取包含 userdata 脚本的模板'}</p>
      <button
        class="px-4 py-2 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 cursor-pointer transition-colors"
        onclick={() => onTabChange && onTabChange('registry')}
      >
        {t.noUserdataTemplatesHintButton || '前往模板仓库'}
      </button>
    </div>
  {:else}
    <!-- Search -->
    <div>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder={t.userdataSearchPlaceholder || '搜索脚本名称、CVE 编号...'}
        class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
      />
    </div>

    <!-- Category filter -->
    <div class="flex gap-1.5 flex-wrap">
      {#each categories() as cat}
        <button
          onclick={() => { activeCategory = cat.key; selectedTemplate = null; }}
          class="px-3 py-1.5 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {activeCategory === cat.key ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
        >
          {cat.label}
          <span class="ml-1 opacity-60">{cat.count}</span>
        </button>
      {/each}
    </div>

    <!-- Content: list + detail -->
    <div class="flex-1 flex gap-4 min-h-0">
      <!-- Left: script list -->
      <div class="w-1/3 min-w-[200px] bg-white rounded-xl border border-gray-100 overflow-hidden flex flex-col">
        <div class="px-3 py-2 border-b border-gray-50 flex items-center gap-1.5 text-[11px] text-gray-400">
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" /></svg>
          {filteredTemplates().length} {t.userdataItems || '个脚本'}
        </div>
        <div class="flex-1 overflow-y-auto">
          {#if filteredTemplates().length === 0}
            <div class="p-4 text-center text-[12px] text-gray-400">{t.noResults || '无匹配结果'}</div>
          {:else}
            {#each filteredTemplates() as tmpl}
              <button
                onclick={() => selectTemplate(tmpl)}
                class="w-full px-3 py-2.5 text-left border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer {selectedTemplate?.name === tmpl.name ? 'bg-gray-50 border-l-2 border-l-gray-900' : 'border-l-2 border-l-transparent'}"
              >
                <div class="flex items-center gap-2">
                  <svg class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="{getCategoryIconPath(tmpl.category)}" />
                  </svg>
                  <div class="flex-1 min-w-0">
                    <div class="text-[13px] font-medium text-gray-900 truncate">{tmpl.nameZh || tmpl.name}</div>
                    <div class="flex items-center gap-1.5 mt-0.5">
                      <span class="text-[11px] text-gray-400">{tmpl.type || 'bash'}</span>
                      {#if tmpl.cveId}
                        <span class="text-[11px] text-amber-600 font-medium">{tmpl.cveId}</span>
                      {/if}
                    </div>
                  </div>
                </div>
              </button>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Right: detail -->
      <div class="flex-1 bg-white rounded-xl border border-gray-100 overflow-hidden flex flex-col">
        {#if selectedTemplate}
          <!-- Header -->
          <div class="px-4 py-3 border-b border-gray-100">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-[13px] font-medium text-gray-900">{selectedTemplate.nameZh || selectedTemplate.name}</h3>
                <div class="flex items-center gap-2 mt-1">
                  <span class="text-[11px] px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{categoryLabel(selectedTemplate.category)}</span>
                  {#if selectedTemplate.cveId}
                    <span class="text-[11px] px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded font-medium">{selectedTemplate.cveId}</span>
                  {/if}
                  {#if selectedTemplate.level}
                    <span class="text-[11px] px-1.5 py-0.5 rounded {selectedTemplate.level === 'critical' ? 'bg-red-50 text-red-600' : 'bg-amber-50 text-amber-600'}">
                      {selectedTemplate.level === 'critical' ? (t.severityCritical || '严重') : (t.severityHigh || '高危')}
                    </span>
                  {/if}
                </div>
              </div>
              <button
                onclick={copyScript}
                class="px-3 py-1.5 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {copied ? 'bg-emerald-500 text-white' : 'bg-gray-900 text-white hover:bg-gray-800'}"
              >
                {copied ? (t.copiedSuccess || '已复制') : (t.copyScript || '复制脚本')}
              </button>
            </div>
          </div>

          <!-- Meta info -->
          {#if selectedTemplate.description || selectedTemplate.environment}
            <div class="px-4 py-3 border-b border-gray-50 space-y-2">
              {#if selectedTemplate.description}
                <p class="text-[12px] text-gray-600">{selectedTemplate.description}</p>
              {/if}
              {#if selectedTemplate.environment}
                <div class="flex gap-3 text-[12px] text-gray-500">
                  {#if selectedTemplate.environment.port}
                    <span>端口: <span class="text-gray-700">{selectedTemplate.environment.port}</span></span>
                  {/if}
                  {#if selectedTemplate.environment.image}
                    <span>镜像: <span class="text-gray-700 font-mono">{selectedTemplate.environment.image}</span></span>
                  {/if}
                </div>
              {/if}
              {#if selectedTemplate.installNotes}
                <p class="flex items-center gap-1 text-[12px] text-amber-600">
                  <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" /></svg>
                  {selectedTemplate.installNotes}
                </p>
              {/if}
            </div>
          {/if}

          <!-- Script -->
          <div class="flex-1 overflow-auto">
            <pre class="px-4 py-3 text-[12px] text-gray-100 bg-gray-900 font-mono leading-relaxed h-full overflow-auto m-0 rounded-none">{selectedTemplate.script}</pre>
          </div>
        {:else}
          <!-- No selection -->
          <div class="flex-1 flex items-center justify-center">
            <div class="text-center">
              <svg class="w-10 h-10 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
              <p class="text-[13px] text-gray-400">{t.userdataSelectHint || '选择一个脚本查看详情'}</p>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
