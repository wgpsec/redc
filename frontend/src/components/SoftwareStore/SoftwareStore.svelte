<script>
  import { onMount, onDestroy } from 'svelte';
  import { ListCases, GetF8xCatalog, GetF8xCategories, GetF8xPresets, GetF8xStatus, EnsureF8x, RunF8xInstall, GetF8xInstallHistory, GetF8xRunningTasks, RefreshF8xCatalog } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

  let { t, onTabChange } = $props();

  let catalog = $state([]);
  let categories = $state([]);
  let presets = $state([]);
  let cases = $state([]);
  let loading = $state(true);
  let catalogError = $state('');

  let selectedCaseID = $state('');
  let activeCategory = $state('all');
  let searchQuery = $state('');
  let selectedModules = $state(new Set());
  let f8xStatus = $state(null);
  let checkingStatus = $state(false);

  let installing = $state(false);
  let installLog = $state([]);
  let showLog = $state(false);
  let currentTaskID = $state('');
  let installHistory = $state([]);

  // Install confirm dialog
  let installConfirm = $state({ show: false, mod: null });

  // Log panel resize
  let logHeight = $state(256);
  let resizing = $state(false);
  let resizeStartY = 0;
  let resizeStartH = 0;

  // Auto-scroll log
  let logContainer = $state(null);

  onMount(async () => {
    try {
      const [cat, cats, pre, caseList] = await Promise.all([
        GetF8xCatalog(),
        GetF8xCategories(),
        GetF8xPresets(),
        ListCases()
      ]);
      catalog = cat || [];
      categories = cats || [];
      presets = pre || [];
      cases = (caseList || []).filter(c => c.state === 'running');
      if (cases.length > 0) selectedCaseID = cases[0].id;
      if (catalog.length === 0) {
        catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
      }
    } catch (e) {
      console.error('Failed to load f8x catalog:', e);
      catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
    }
    loading = false;
  });

  // Listen for f8x events
  let cleanupOutput, cleanupDone;
  onMount(() => {
    cleanupOutput = EventsOn('f8x:output', (data) => {
      if (data.taskID === currentTaskID) {
        installLog = [...installLog, { type: data.type, text: data.text }];
        scrollLogToBottom();
      }
    });
    cleanupDone = EventsOn('f8x:done', (data) => {
      if (data.taskID === currentTaskID) {
        installing = false;
        installLog = [...installLog, { type: data.status === 'success' ? 'success' : 'error', text: data.status === 'success' ? '\n✅ Installation completed successfully!' : `\n❌ Installation failed: ${data.error || 'Unknown error'}` }];
        scrollLogToBottom();
        loadHistory();
      }
    });

    // Resize handlers
    const onMouseMove = (e) => {
      if (!resizing) return;
      const delta = resizeStartY - e.clientY;
      logHeight = Math.max(128, Math.min(600, resizeStartH + delta));
    };
    const onMouseUp = () => { resizing = false; };
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
    return () => {
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };
  });
  onDestroy(() => {
    if (cleanupOutput) EventsOff('f8x:output');
    if (cleanupDone) EventsOff('f8x:done');
  });

  function scrollLogToBottom() {
    requestAnimationFrame(() => {
      if (logContainer) logContainer.scrollTop = logContainer.scrollHeight;
    });
  }

  function startResize(e) {
    resizing = true;
    resizeStartY = e.clientY;
    resizeStartH = logHeight;
    e.preventDefault();
  }

  async function checkF8xStatus() {
    if (!selectedCaseID) return;
    checkingStatus = true;
    try {
      f8xStatus = await GetF8xStatus(selectedCaseID);
    } catch (e) {
      f8xStatus = { error: e.toString() };
    }
    checkingStatus = false;
  }

  async function loadHistory() {
    if (!selectedCaseID) return;
    try {
      installHistory = await GetF8xInstallHistory(selectedCaseID) || [];
    } catch(e) { /* ignore */ }
  }

  $effect(() => {
    if (selectedCaseID) {
      checkF8xStatus();
      loadHistory();
    }
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
        (m.tags || []).some(tag => tag.includes(q))
      );
    }
    return list;
  });

  // Check if a module was installed (from history)
  function isInstalled(flag) {
    return installHistory.some(r => r.status === 'success' && r.flags && r.flags.includes(flag));
  }

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

  async function installSelected() {
    if (selectedModules.size === 0 || !selectedCaseID || installing) return;
    const flags = [];
    for (const id of selectedModules) {
      const mod = catalog.find(m => m.id === id);
      if (mod) flags.push(mod.flag);
    }
    installing = true;
    installLog = [];
    showLog = true;
    try {
      currentTaskID = await RunF8xInstall(selectedCaseID, flags);
    } catch (e) {
      installing = false;
      installLog = [{ type: 'error', text: 'Failed to start: ' + e.toString() }];
    }
  }

  function showInstallConfirm(e, mod) {
    e.stopPropagation();
    if (!selectedCaseID || installing) return;
    installConfirm = { show: true, mod };
  }

  async function confirmInstallSingle() {
    const mod = installConfirm.mod;
    installConfirm = { show: false, mod: null };
    if (!mod || !selectedCaseID || installing) return;
    installing = true;
    installLog = [];
    showLog = true;
    try {
      currentTaskID = await RunF8xInstall(selectedCaseID, [mod.flag]);
    } catch (e) {
      installing = false;
      installLog = [{ type: 'error', text: 'Failed to start: ' + e.toString() }];
    }
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
      'basic': '⚙️', 'development': '💻', 'pentest-recon': '🔍',
      'pentest-exploit': '💥', 'pentest-post': '🎯', 'blue-team': '🛡️',
      'red-infra': '🏗️', 'vuln-env': '🎪', 'misc': '🧰'
    };
    return icons[catId] || '📦';
  }

  function formatDuration(start, end) {
    if (!start || !end) return '';
    const ms = new Date(end) - new Date(start);
    if (ms < 1000) return '<1s';
    const s = Math.floor(ms / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    const rs = s % 60;
    return rs > 0 ? `${m}m${rs}s` : `${m}m`;
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
      <button onclick={refreshCatalog} disabled={refreshing} class="text-[11px] text-gray-400 hover:text-red-500 transition-colors disabled:opacity-40" title="刷新在线目录">
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
        {refreshing ? '⟳...' : '⟳ 重试'}
      </button>
    </div>
  {/if}

  <!-- Target VPS Selector -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 flex-1">
        <span class="text-[12px] font-medium text-gray-700 whitespace-nowrap">{t.f8xTargetVPS || '目标主机'}</span>
        <select bind:value={selectedCaseID} class="text-[12px] border border-gray-200 rounded-lg px-3 py-1.5 bg-gray-50 focus:outline-none focus:ring-1 focus:ring-red-300 flex-1 max-w-xs">
          {#if cases.length === 0}
            <option value="">{t.f8xNoCases || '无可用主机（请先部署场景）'}</option>
          {:else}
            {#each cases as c}
              <option value={c.id}>{c.name || c.id} ({c.type})</option>
            {/each}
          {/if}
        </select>
        {#if cases.length === 0}
          <button onclick={() => onTabChange && onTabChange('cases')} class="text-[11px] px-3 py-1.5 rounded-lg bg-red-50 hover:bg-red-100 text-red-600 transition-colors flex items-center gap-1">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
            {t.f8xGoToDeploy || '去部署'}
          </button>
        {:else}
          <button onclick={checkF8xStatus} disabled={!selectedCaseID || checkingStatus} class="text-[11px] px-3 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 disabled:opacity-40 transition-colors flex items-center gap-1.5">
            {#if checkingStatus}
              <div class="w-3 h-3 border-[1.5px] border-gray-300 border-t-gray-600 rounded-full animate-spin"></div>
              {t.f8xChecking || '检测中...'}
            {:else}
              {t.f8xCheckStatus || '检测状态'}
            {/if}
          </button>
          {#if f8xStatus && !checkingStatus}
            <span class="text-[11px] px-2.5 py-1 rounded-lg {f8xStatus.error ? 'bg-red-50 text-red-600 border border-red-200' : f8xStatus.deployed ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-amber-50 text-amber-600 border border-amber-200'}">
              {f8xStatus.error ? '⚠ 连接失败' : f8xStatus.deployed ? `✓ f8x ${f8xStatus.version || '已部署'}` : '✗ f8x 未部署'}
            </span>
          {/if}
        {/if}
      </div>

      <div class="flex items-center gap-2">
        {#if selectedModules.size > 0}
          <span class="text-[11px] text-gray-500">{t.f8xSelected || '已选'} {selectedModules.size}</span>
          <button onclick={() => selectedModules = new Set()} class="text-[11px] text-gray-400 hover:text-gray-600">
            {t.f8xClearSelection || '清除'}
          </button>
        {/if}
        <button onclick={installSelected} disabled={selectedModules.size === 0 || !selectedCaseID || installing} class="text-[12px] px-4 py-1.5 rounded-lg bg-red-600 hover:bg-red-700 text-white disabled:opacity-40 transition-colors font-medium">
          {installing ? (t.f8xInstalling || '安装中...') : (t.f8xBatchInstall || '批量安装')}
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
          {categoryIcon(cat.id)} {cat.nameZh || cat.name} ({cat.count})
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
        {@const installed = isInstalled(mod.flag)}
        <div class="group bg-white rounded-xl border transition-all cursor-pointer {isSelected ? 'border-red-400 ring-1 ring-red-200 bg-red-50/30' : 'border-gray-100 hover:border-gray-200 hover:shadow-sm'}" onclick={() => toggleModule(mod.id)}>
          <div class="p-3">
            <div class="flex items-start justify-between mb-1.5">
              <div class="flex items-center gap-1.5">
                <h4 class="text-[12px] font-semibold text-gray-900 leading-tight">{mod.name}</h4>
                {#if installed}
                  <span class="text-[8px] px-1 py-0.5 rounded bg-green-50 text-green-600 font-medium leading-none">已装</span>
                {/if}
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
              <button onclick={(e) => showInstallConfirm(e, mod)} disabled={!selectedCaseID || installing} class="text-[10px] px-2 py-0.5 rounded bg-gray-100 hover:bg-red-100 text-gray-500 hover:text-red-600 disabled:opacity-30 transition-colors opacity-0 group-hover:opacity-100">
                {t.f8xInstall || '安装'}
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>

    {#if filteredModules().length === 0}
      <div class="text-center py-8 text-gray-400 text-[12px]">
        {t.f8xNoResults || '没有匹配的工具'}
      </div>
    {/if}
  {/if}

  <!-- Install Log Drawer -->
  {#if showLog}
    <div class="bg-gray-900 rounded-xl border border-gray-700 overflow-hidden">
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="h-1.5 bg-gray-700 hover:bg-gray-600 cursor-ns-resize flex items-center justify-center transition-colors" onmousedown={startResize}>
        <div class="w-8 h-0.5 bg-gray-500 rounded-full"></div>
      </div>
      <div class="px-4 py-2 border-b border-gray-700 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-[12px] text-gray-300 font-medium">{t.f8xInstallLog || '安装日志'}</span>
          {#if installing}
            <div class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <button onclick={() => installLog = []} class="text-[10px] text-gray-500 hover:text-gray-300">{t.f8xClearLog || '清空'}</button>
          <button onclick={() => showLog = false} class="text-[10px] text-gray-500 hover:text-gray-300">✕</button>
        </div>
      </div>
      <div bind:this={logContainer} class="p-4 overflow-y-auto font-mono text-[11px] leading-relaxed whitespace-pre-wrap break-all" style="max-height: {logHeight}px">
        {#each installLog as entry}
          <span class="{entry.type === 'error' ? 'text-red-400' : entry.type === 'success' ? 'text-green-400' : entry.type === 'info' ? 'text-blue-400' : 'text-gray-300'}">{entry.text}</span>
        {/each}
        {#if installLog.length === 0}
          <span class="text-gray-600">{t.f8xWaitingOutput || '等待输出...'}</span>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Install History -->
  {#if installHistory.length > 0}
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.f8xHistory || '安装历史'}</h3>
      </div>
      <div class="divide-y divide-gray-50">
        {#each installHistory.slice().reverse().slice(0, 10) as record}
          <div class="px-5 py-2 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <span class="w-2 h-2 rounded-full {record.status === 'success' ? 'bg-green-400' : record.status === 'failed' ? 'bg-red-400' : 'bg-yellow-400'}"></span>
              <span class="text-[11px] font-mono text-gray-700">{record.flags}</span>
            </div>
            <div class="flex items-center gap-3">
              {#if record.startedAt && record.finishedAt}
                <span class="text-[10px] text-gray-400 font-mono">{formatDuration(record.startedAt, record.finishedAt)}</span>
              {/if}
              <span class="text-[10px] text-gray-400">{record.startedAt ? new Date(record.startedAt).toLocaleString() : ''}</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full {record.status === 'success' ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-600'}">
                {record.status}
              </span>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<!-- Install Confirm Dialog -->
{#if installConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => installConfirm = { show: false, mod: null }}>
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
            <p class="text-[13px] text-gray-500">{t.f8xConfirmInstallDesc || '将在目标主机上执行安装'}</p>
          </div>
        </div>
        <div class="bg-gray-50 rounded-lg px-4 py-3">
          <p class="text-[13px] font-medium text-gray-900">{installConfirm.mod?.name}</p>
          <p class="text-[11px] text-gray-500 mt-0.5">{installConfirm.mod?.descriptionZh || installConfirm.mod?.description}</p>
          <p class="text-[11px] text-gray-400 font-mono mt-1">f8x {installConfirm.mod?.flag}</p>
        </div>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={() => installConfirm = { show: false, mod: null }}
        >{t.cancel || '取消'}</button>
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          onclick={confirmInstallSingle}
        >{t.f8xInstall || '安装'}</button>
      </div>
    </div>
  </div>
{/if}
