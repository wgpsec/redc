<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetF8xCatalog, GetF8xCategories, GetF8xPresets, BuildF8xCommand, RefreshF8xCatalog } from '../../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import Modal from '../UI/Modal.svelte';

  let { t, onTabChange } = $props();

  let catalog = $state([]);
  let categories = $state([]);
  let presets = $state([]);
  let loading = $state(true);
  let catalogError = $state('');

  let activeCategory = $state('all');
  let searchQuery = $state('');
  let selectedModules = $state(new Set());
  let expandedIncludes = $state(new Set());

  // Install confirm dialog (single tool)
  let installConfirm = $state({ show: false, mod: null });

  // SSH session picker modal
  let sessionPicker = $state({ show: false, flags: [], toolName: '' });
  let sshSessions = $derived(window.__sshSessions || []);

  onMount(async () => {
    try {
      const [cat, cats, pre] = await Promise.all([
        GetF8xCatalog(),
        GetF8xCategories(),
        GetF8xPresets(),
      ]);
      catalog = cat || [];
      categories = cats || [];
      presets = pre || [];
      if (catalog.length === 0) {
        catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
      }
    } catch (e) {
      console.error('Failed to load f8x catalog:', e);
      catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
    }
    loading = false;
  });

  const filteredModules = $derived(() => {
    let list = catalog;
    if (activeCategory !== 'all') {
      list = list.filter(m => m.category === activeCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(m =>
        m.name.toLowerCase().includes(q) ||
        m.nameZh.toLowerCase().includes(q) ||
        (m.description || '').toLowerCase().includes(q) ||
        (m.descriptionZh || '').toLowerCase().includes(q) ||
        (m.tags || []).some(tag => tag.includes(q)) ||
        (m.includes || []).some(t => t.name.toLowerCase().includes(q))
      );
    }
    return list;
  });

  function toggleModule(id) {
    const next = new Set(selectedModules);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedModules = next;
  }

  function selectPreset(preset) {
    const next = new Set();
    for (const flag of preset.flags) {
      const mod = catalog.find(m => m.flag === flag);
      if (mod) next.add(mod.id);
    }
    selectedModules = next;
  }

  // --- New SSH session-based install flow ---

  function getActiveSshSessions() {
    return window.__sshSessions || [];
  }

  function openSessionPicker(flags, toolName = '') {
    const sessions = getActiveSshSessions();
    if (sessions.length === 0) {
      // No active SSH sessions, prompt to go to SSH terminal
      sessionPicker = { show: true, flags, toolName, noSessions: true };
      return;
    }
    sessionPicker = { show: true, flags, toolName, noSessions: false };
  }

  async function executeOnSession(sessionId) {
    const { flags } = sessionPicker;
    sessionPicker = { show: false, flags: [], toolName: '' };
    try {
      const command = await BuildF8xCommand(flags);
      // Switch to SSH tab and send command
      window.dispatchEvent(new CustomEvent('f8x-execute', { detail: { sessionId, command } }));
      window.dispatchEvent(new CustomEvent('switchTab', { detail: 'sshManager' }));
    } catch (e) {
      console.error('BuildF8xCommand failed:', e);
    }
  }

  function installSelected() {
    if (selectedModules.size === 0) return;
    const flags = [];
    for (const id of selectedModules) {
      const mod = catalog.find(m => m.id === id);
      if (mod) flags.push(mod.flag);
    }
    const names = [];
    for (const id of selectedModules) {
      const mod = catalog.find(m => m.id === id);
      if (mod) names.push(mod.nameZh || mod.name);
    }
    openSessionPicker(flags, names.join(', '));
  }

  function showInstallConfirm(e, mod) {
    e.stopPropagation();
    installConfirm = { show: true, mod };
  }

  function toggleIncludes(e, modId) {
    e.stopPropagation();
    const next = new Set(expandedIncludes);
    if (next.has(modId)) next.delete(modId); else next.add(modId);
    expandedIncludes = next;
  }

  function confirmInstallSingle() {
    const mod = installConfirm.mod;
    installConfirm = { show: false, mod: null };
    if (!mod) return;
    openSessionPicker([mod.flag], mod.nameZh || mod.name);
  }

  let catalogSource = $state('');
  let refreshing = $state(false);

  async function refreshCatalog() {
    refreshing = true;
    catalogError = '';
    try {
      const result = await RefreshF8xCatalog();
      if (result.success) {
        catalogSource = `${result.source} · v${result.version} · ${result.count} tools`;
      } else {
        catalogSource = '';
        catalogError = result.error || (t.f8xCatalogError || '无法加载工具目录，请检查网络连接');
      }
      const [cat, cats, pre] = await Promise.all([
        GetF8xCatalog(), GetF8xCategories(), GetF8xPresets()
      ]);
      catalog = cat || [];
      categories = cats || [];
      presets = pre || [];
      if (catalog.length > 0) catalogError = '';
    } catch(e) {
      catalogError = e.toString();
    }
    refreshing = false;
  }

  function categoryIcon(catId) {
    const icons = {
      'basic': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" /><circle cx="12" cy="12" r="3" /></svg>',
      'development': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>',
      'pentest-recon': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>',
      'pentest-exploit': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.28-3.07a.75.75 0 010-1.3l5.28-3.07a.75.75 0 011.08.67v6.1a.75.75 0 01-1.08.66z" /><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>',
      'pentest-post': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 3v1.5M3 21v-6m0 0l2.77-.693a9 9 0 016.208.682l.108.054a9 9 0 006.086.71l3.114-.732a48.524 48.524 0 01-.005-10.499l-3.11.732a9 9 0 01-6.085-.711l-.108-.054a9 9 0 00-6.208-.682L3 4.5M3 15V4.5" /></svg>',
      'blue-team': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>',
      'red-infra': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" /></svg>',
      'vuln-env': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" /></svg>',
      'misc': '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.28-3.07a.75.75 0 010-1.3l5.28-3.07a.75.75 0 011.08.67v6.1a.75.75 0 01-1.08.66zM20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>'
    };
    return icons[catId] || '<svg class="inline w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>';
  }
</script>

<div class="space-y-4">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <p class="text-[11px] text-gray-500">{t.f8xStoreDesc || '基于 f8x 的一站式工具安装平台，支持 150+ 渗透/开发/运维工具'}</p>
    </div>
    <div class="flex items-center gap-2">
      {#if catalogSource}
        <span class="text-[10px] px-2 py-0.5 rounded-full bg-blue-50 text-blue-600">{catalogSource}</span>
      {/if}
      <button onclick={refreshCatalog} disabled={refreshing} class="text-[11px] text-gray-400 hover:text-red-500 transition-colors disabled:opacity-40" title={t.refreshOnlineCatalog}>
        {refreshing ? '⟳...' : '⟳'}
      </button>
      <button onclick={() => BrowserOpenURL('https://github.com/ffffffff0x/f8x')} class="text-[11px] text-gray-400 hover:text-red-500 transition-colors">
        GitHub ↗
      </button>
    </div>
  </div>

  <!-- Catalog Error Banner -->
  {#if catalogError}
    <div class="bg-red-50 border border-red-200 rounded-xl px-5 py-3 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>
        <span class="text-[12px] text-red-700">{catalogError}</span>
      </div>
      <button onclick={refreshCatalog} disabled={refreshing} class="text-[11px] px-3 py-1 rounded-lg bg-red-100 hover:bg-red-200 text-red-700 disabled:opacity-40 transition-colors">
        {refreshing ? '⟳...' : `⟳ ${t.retry}`}
      </button>
    </div>
  {/if}

  <!-- Toolbar -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 flex-1">
        <span class="text-[12px] font-medium text-gray-700 whitespace-nowrap">{t.f8xTargetVPS || '目标主机'}</span>
        <span class="text-[11px] text-gray-500 bg-gray-50 px-3 py-1.5 rounded-lg border border-gray-200">
          {#if getActiveSshSessions().length > 0}
            <span class="text-green-600">● {getActiveSshSessions().length}</span> {t.f8xActiveSessions || '个活跃 SSH 连接'}
          {:else}
            <span class="text-gray-400">● 0</span> {t.f8xNoSessions || '无活跃连接'}
          {/if}
        </span>
        {#if getActiveSshSessions().length === 0}
          <button onclick={() => { if (onTabChange) onTabChange('sshManager'); }} class="text-[11px] px-3 py-1.5 rounded-lg bg-red-50 hover:bg-red-100 text-red-600 transition-colors flex items-center gap-1 cursor-pointer">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
            {t.f8xGoToSSH || '去连接'}
          </button>
        {/if}
      </div>

      <div class="flex items-center gap-2">
        {#if selectedModules.size > 0}
          <span class="text-[11px] text-gray-500">{t.f8xSelected || '已选'} {selectedModules.size}</span>
          <button onclick={() => selectedModules = new Set()} class="text-[11px] text-gray-400 hover:text-gray-600 cursor-pointer">
            {t.f8xClearSelection || '清除'}
          </button>
        {/if}
        <button onclick={installSelected} disabled={selectedModules.size === 0} class="text-[12px] px-4 py-1.5 rounded-lg bg-red-600 hover:bg-red-700 text-white disabled:opacity-40 transition-colors font-medium cursor-pointer">
          {t.f8xBatchInstall || '批量安装'}
        </button>
      </div>
    </div>
  </div>

  <!-- Presets -->
  <div class="flex items-center gap-2 flex-wrap">
    <span class="text-[11px] text-gray-500 mr-1">{t.f8xPresets || '快捷预设'}:</span>
    {#each presets as preset}
      <button onclick={() => selectPreset(preset)} class="text-[11px] px-3 py-1 rounded-full border border-gray-200 hover:border-red-300 hover:bg-red-50 text-gray-600 hover:text-red-600 transition-colors">
        {preset.nameZh || preset.name} <span class="text-gray-400">({preset.flags?.length || 0})</span>
      </button>
    {/each}
  </div>

  <!-- Search + Category Tabs -->
  <div class="flex items-center gap-3">
    <div class="relative flex-1 max-w-xs">
      <input type="text" bind:value={searchQuery} placeholder={t.f8xSearch || '搜索工具...'} class="w-full text-[12px] border border-gray-200 rounded-lg pl-8 pr-7 py-1.5 bg-white focus:outline-none focus:ring-1 focus:ring-red-300" />
      <svg class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
      {#if searchQuery}
        <button onclick={() => searchQuery = ''} class="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-gray-200 hover:bg-gray-300 flex items-center justify-center transition-colors">
          <svg class="w-2.5 h-2.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      {/if}
    </div>
    <div class="flex items-center gap-1 flex-wrap flex-1">
      <button onclick={() => activeCategory = 'all'} class="text-[11px] px-2.5 py-1 rounded-lg transition-colors {activeCategory === 'all' ? 'bg-red-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}">
        {t.f8xAll || '全部'} ({catalog.length})
      </button>
      {#each categories as cat}
        <button onclick={() => activeCategory = cat.id} class="text-[11px] px-2.5 py-1 rounded-lg transition-colors {activeCategory === cat.id ? 'bg-red-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}">
          {@html categoryIcon(cat.id)} {cat.nameZh || cat.name} ({cat.count})
        </button>
      {/each}
    </div>
  </div>

  <!-- Tool Grid -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <div class="w-5 h-5 border-2 border-red-200 border-t-red-600 rounded-full animate-spin"></div>
    </div>
  {:else}
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
      {#each filteredModules() as mod}
        {@const isSelected = selectedModules.has(mod.id)}
        {@const isBatch = (mod.tags || []).includes('batch')}
        {@const hasIncludes = isBatch && mod.includes && mod.includes.length > 0}
        {@const isExpanded = expandedIncludes.has(mod.id)}
        <div class="group bg-white rounded-xl border transition-all cursor-pointer {isSelected ? 'border-red-400 ring-1 ring-red-200 bg-red-50/30' : 'border-gray-100 hover:border-gray-200 hover:shadow-sm'}" onclick={() => toggleModule(mod.id)}>
          <div class="p-3">
            <div class="flex items-start justify-between mb-1.5">
              <div class="flex items-center gap-1.5">
                <h4 class="text-[12px] font-semibold text-gray-900 leading-tight">{mod.name}</h4>
              </div>
              <div class="flex items-center gap-1">
                {#if isBatch}
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-amber-50 text-amber-600 font-medium">SUITE</span>
                {/if}
                <input type="checkbox" checked={isSelected} onclick={(e) => { e.stopPropagation(); toggleModule(mod.id); }} class="w-3.5 h-3.5 rounded border-gray-300 text-red-600 focus:ring-red-500 cursor-pointer" />
              </div>
            </div>
            <p class="text-[10px] text-gray-500 leading-relaxed line-clamp-2 mb-2">{mod.descriptionZh || mod.description}</p>
            <div class="flex items-center justify-between">
              <span class="text-[9px] text-gray-400 font-mono">{mod.flag}</span>
              <div class="flex items-center gap-1">
                {#if hasIncludes}
                  <button onclick={(e) => toggleIncludes(e, mod.id)} class="text-[10px] px-2 py-0.5 rounded bg-gray-100 hover:bg-blue-100 text-gray-500 hover:text-blue-600 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer inline-flex items-center gap-0.5" title={t.f8xViewTools || '查看包含的工具'}>
                    <svg class="w-3 h-3 transition-transform {isExpanded ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" /></svg>
                    {mod.includes.length}
                  </button>
                {/if}
                <button onclick={(e) => showInstallConfirm(e, mod)} class="text-[10px] px-2 py-0.5 rounded bg-gray-100 hover:bg-red-100 text-gray-500 hover:text-red-600 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer">
                  {t.f8xInstall || '安装'}
                </button>
              </div>
            </div>
          </div>
          {#if hasIncludes && isExpanded}
            <div class="border-t border-gray-100 px-3 py-2 bg-gray-50/50 max-h-48 overflow-y-auto" onclick={(e) => e.stopPropagation()}>
              <div class="flex flex-wrap gap-1">
                {#each mod.includes as tool}
                  {#if tool.url}
                    <a href={tool.url} target="_blank" rel="noopener noreferrer" onclick={(e) => { e.stopPropagation(); BrowserOpenURL(tool.url); e.preventDefault(); }} class="text-[9px] px-1.5 py-0.5 rounded bg-white border border-gray-200 text-gray-700 hover:border-blue-300 hover:text-blue-600 hover:bg-blue-50 transition-colors cursor-pointer" title={tool.url}>
                      {tool.name}
                    </a>
                  {:else}
                    <span class="text-[9px] px-1.5 py-0.5 rounded bg-white border border-gray-200 text-gray-500">
                      {tool.name}
                    </span>
                  {/if}
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    {#if filteredModules().length === 0}
      <div class="text-center py-8 text-gray-400 text-[12px]">
        {t.f8xNoResults || '没有匹配的工具'}
      </div>
    {/if}
  {/if}
</div>

<!-- Install Confirm Dialog -->
<Modal show={installConfirm.show} onclose={() => installConfirm = { show: false, mod: null }}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.f8xConfirmInstall || '确认安装'}</h3>
            <p class="text-[13px] text-gray-500">{t.f8xConfirmInstallDesc || '将通过 SSH 终端执行安装'}</p>
          </div>
        </div>
        <div class="bg-gray-50 rounded-lg px-4 py-3">
          <p class="text-[13px] font-medium text-gray-900">{installConfirm.mod?.name}</p>
          <p class="text-[11px] text-gray-500 mt-0.5">{installConfirm.mod?.descriptionZh || installConfirm.mod?.description}</p>
          <p class="text-[11px] text-gray-400 font-mono mt-1">f8x {installConfirm.mod?.flag}</p>
        </div>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => installConfirm = { show: false, mod: null }}
        >{t.cancel || '取消'}</button>
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors cursor-pointer"
          onclick={confirmInstallSingle}
        >{t.f8xInstall || '安装'}</button>
      </div>
    </div>
</Modal>

<!-- SSH Session Picker Modal -->
<Modal show={sessionPicker.show} onclose={() => sessionPicker = { show: false, flags: [], toolName: '' }}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-md w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-gray-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.f8xSelectSession || '选择 SSH 终端'}</h3>
            <p class="text-[13px] text-gray-500">{sessionPicker.toolName || sessionPicker.flags.join(' ')}</p>
          </div>
        </div>

        {#if sessionPicker.noSessions || getActiveSshSessions().length === 0}
          <div class="text-center py-6">
            <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p class="text-[13px] text-gray-500 mb-3">{t.f8xNoActiveSessions || '没有活跃的 SSH 连接'}</p>
            <p class="text-[11px] text-gray-400 mb-4">{t.f8xGoToSSHHint || '请先在 SSH 终端页面建立连接'}</p>
            <button 
              class="h-8 px-4 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors cursor-pointer"
              onclick={() => { sessionPicker = { show: false, flags: [], toolName: '' }; if (onTabChange) onTabChange('sshManager'); }}
            >
              <svg class="w-3.5 h-3.5 inline-block mr-1 -mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
              {t.f8xGoToSSH || '去连接'}
            </button>
          </div>
        {:else}
          <div class="space-y-2 max-h-60 overflow-y-auto">
            {#each getActiveSshSessions() as session}
              <button
                class="w-full flex items-center gap-3 px-4 py-3 rounded-lg border border-gray-200 hover:border-red-300 hover:bg-red-50/50 transition-colors text-left cursor-pointer"
                onclick={() => executeOnSession(session.id)}
              >
                <div class="w-8 h-8 rounded-full bg-green-50 flex items-center justify-center flex-shrink-0">
                  <span class="w-2 h-2 rounded-full bg-green-500"></span>
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-[12px] font-medium text-gray-900 truncate">
                    {session.caseName || session.host || 'SSH'}
                    {#if session.isExternal}
                      <span class="text-[10px] text-purple-500 ml-1">(外部)</span>
                    {/if}
                  </p>
                  <p class="text-[10px] text-gray-400 truncate">
                    {session.user}@{session.host}
                  </p>
                </div>
                <svg class="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            {/each}
          </div>
        {/if}
      </div>
      <div class="px-6 py-3 bg-gray-50 flex justify-end">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => sessionPicker = { show: false, flags: [], toolName: '' }}
        >{t.cancel || '取消'}</button>
      </div>
    </div>
</Modal>
