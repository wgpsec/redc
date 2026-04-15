<script>
  import { onMount } from 'svelte';
  import { ListPlugins, InstallPlugin, UninstallPlugin, EnablePlugin, DisablePlugin, UpdatePlugin, GetPluginConfig, SavePluginConfig, FetchPluginRegistry, GetPluginsDir } from '../../../wailsjs/go/main/App.js';
  import { compareVersions } from '../../utils/version.js';
  import Modal from '../UI/Modal.svelte';

  let { t } = $props();

  let plugins = $state([]);
  let registryPlugins = $state([]);
  let loading = $state(false);
  let registryLoading = $state(false);
  let error = $state('');
  let installSource = $state('');
  let installing = $state(false);
  let actionLoading = $state('');
  let activeView = $state('installed');
  let searchQuery = $state('');

  let configModal = $state({ show: false, plugin: null, config: '', schema: null, saving: false });
  let confirmModal = $state({ show: false, action: '', pluginName: '', message: '' });
  let pluginsDir = $state('');

  const enabledPlugins = $derived(plugins.filter(p => p.enabled));
  const disabledPlugins = $derived(plugins.filter(p => !p.enabled));
  const installedMap = $derived(new Map(plugins.map(p => [p.name, p.version])));

  const filteredPlugins = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return plugins;
    return plugins.filter(p => p.name?.toLowerCase().includes(q) || p.description?.toLowerCase().includes(q) || p.tags?.some(t => t.toLowerCase().includes(q)));
  });

  const filteredRegistry = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return registryPlugins;
    return registryPlugins.filter(p => p.name?.toLowerCase().includes(q) || p.description?.toLowerCase().includes(q) || p.tags?.some(t => t.toLowerCase().includes(q)));
  });

  async function loadPlugins() {
    loading = true;
    error = '';
    try {
      plugins = (await ListPlugins() || []).sort((a, b) => (a.name || '').localeCompare(b.name || ''));
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      loading = false;
    }
  }

  async function handleInstall() {
    if (!installSource.trim()) return;
    installing = true;
    error = '';
    try {
      await InstallPlugin(installSource.trim());
      installSource = '';
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      installing = false;
    }
  }

  async function handleToggle(p) {
    actionLoading = p.name;
    try {
      if (p.enabled) { await DisablePlugin(p.name); } else { await EnablePlugin(p.name); }
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  async function handleUpdate(p) {
    actionLoading = p.name + '-update';
    try {
      await UpdatePlugin(p.name);
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  function showUninstallConfirm(p) {
    confirmModal = { show: true, action: 'uninstall', pluginName: p.name, message: t.pluginConfirmUninstall?.replace('{name}', p.name) || `Uninstall ${p.name}?` };
  }

  async function handleConfirmAction() {
    if (confirmModal.action === 'uninstall') {
      actionLoading = confirmModal.pluginName + '-uninstall';
      try {
        await UninstallPlugin(confirmModal.pluginName);
        await loadPlugins();
      } catch (e) {
        error = e?.message || String(e);
      } finally {
        actionLoading = '';
      }
    }
    confirmModal = { show: false, action: '', pluginName: '', message: '' };
  }

  async function showConfig(p) {
    try {
      const configStr = await GetPluginConfig(p.name);
      const schema = p.config_schema || {};
      let parsed = {};
      try { parsed = JSON.parse(configStr || '{}') || {}; } catch { parsed = {}; }
      const formValues = {};
      for (const [key, field] of Object.entries(schema)) {
        if (parsed[key] !== undefined) { formValues[key] = parsed[key]; }
        else if (field.default !== undefined && field.default !== '') { formValues[key] = field.type === 'boolean' ? (field.default === 'true') : field.default; }
        else { formValues[key] = field.type === 'boolean' ? false : ''; }
      }
      for (const [key, val] of Object.entries(parsed)) { if (!(key in formValues)) formValues[key] = val; }
      configModal = { show: true, plugin: p, config: configStr || '{}', schema, saving: false, formValues, useForm: Object.keys(schema).length > 0 };
    } catch (e) {
      error = e?.message || String(e);
    }
  }

  function updateConfigFormValue(key, value) {
    configModal.formValues = { ...configModal.formValues, [key]: value };
    configModal.config = JSON.stringify(configModal.formValues, null, 2);
  }

  async function saveConfig() {
    configModal.saving = true;
    try {
      await SavePluginConfig(configModal.plugin.name, configModal.config);
      configModal = { show: false, plugin: null, config: '', schema: null, saving: false };
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
      configModal.saving = false;
    }
  }

  onMount(() => { loadPlugins(); GetPluginsDir().then(d => pluginsDir = d).catch(() => {}); });

  async function loadRegistry() {
    registryLoading = true;
    try {
      registryPlugins = await FetchPluginRegistry() || [];
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      registryLoading = false;
    }
  }

  async function handleInstallFromRegistry(url) {
    actionLoading = url;
    error = '';
    try {
      await InstallPlugin(url);
      await loadPlugins();
      await loadRegistry();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  async function handleUpdateFromRegistry(name) {
    actionLoading = name + '-market-update';
    error = '';
    try {
      await UpdatePlugin(name);
      await loadPlugins();
      await loadRegistry();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  function switchView(view) {
    activeView = view;
    searchQuery = '';
    if (view === 'market' && registryPlugins.length === 0) { loadRegistry(); }
  }
</script>

<div class="space-y-4">
  <!-- Error banner -->
  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-2.5 rounded-xl flex items-center justify-between">
      <span class="text-[12px]">{error}</span>
      <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''}>
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  <!-- Toolbar: tabs + stats + search + install + refresh -->
  <div class="flex items-center gap-3 flex-wrap">
    <!-- Pill tabs -->
    <div class="flex gap-1 bg-gray-100 rounded-lg p-1">
      <button
        onclick={() => switchView('installed')}
        class="px-3 py-1 text-[12px] rounded-md transition-colors cursor-pointer {activeView === 'installed' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
      >{t.pluginInstalled || '已安装'} {plugins.length > 0 ? plugins.length : ''}</button>
      <button
        onclick={() => switchView('market')}
        class="px-3 py-1 text-[12px] rounded-md transition-colors cursor-pointer {activeView === 'market' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
      >{t.pluginMarket || '插件市场'}</button>
    </div>

    <!-- Inline stats (installed view) -->
    {#if activeView === 'installed' && plugins.length > 0}
      <div class="flex items-center gap-3 text-[11px] text-gray-400">
        <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> {enabledPlugins.length} {t.pluginEnabled || '启用'}</span>
        <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-gray-300"></span> {disabledPlugins.length} {t.pluginDisabled || '禁用'}</span>
      </div>
    {/if}

    <div class="flex-1"></div>

    <!-- Search -->
    <div class="relative">
      <svg class="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-1/2 -translate-y-1/2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder={t.pluginSearch || '搜索插件...'}
        class="pl-8 pr-3 py-1.5 text-[12px] w-48 border border-gray-200 rounded-lg bg-white focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
      />
    </div>

    <!-- Refresh -->
    <button
      onclick={() => activeView === 'installed' ? loadPlugins() : loadRegistry()}
      disabled={loading || registryLoading}
      class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
      title={t.refresh || '刷新'}
    >
      <svg class="w-4 h-4 {(loading || registryLoading) ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>
    </button>
  </div>

  {#if activeView === 'installed'}

  <!-- Install form -->
  <div class="flex gap-2">
    <input
      type="text"
      bind:value={installSource}
      placeholder={t.pluginInstallPlaceholder || 'Git 仓库地址或本地路径'}
      class="flex-1 px-3 py-2 text-[12px] border border-gray-200 rounded-lg bg-white focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
      onkeydown={(e) => e.key === 'Enter' && handleInstall()}
    />
    <button
      onclick={handleInstall}
      disabled={installing || !installSource.trim()}
      class="px-4 py-2 bg-gray-900 text-white text-[12px] rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors flex items-center gap-1.5"
    >
      {#if installing}
        <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        {t.pluginInstalling || '安装中...'}
      {:else}
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
        {t.pluginInstall || '安装'}
      {/if}
    </button>
  </div>
  {#if pluginsDir}
    <div class="flex items-center gap-1.5 text-[10px] text-gray-400 -mt-2">
      <svg class="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" /></svg>
      <span class="font-mono select-all">{pluginsDir}</span>
    </div>
  {/if}

  <!-- Plugin list -->
  <div class="bg-white rounded-xl border border-gray-100">
    {#if loading && plugins.length === 0}
      <div class="flex items-center justify-center gap-2 py-12">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
      </div>
    {:else if plugins.length === 0}
      <div class="py-12 text-center">
        <svg class="w-10 h-10 text-gray-200 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M13.5 16.875h3.375m0 0h3.375m-3.375 0V13.5m0 3.375v3.375M6 10.5h2.25a2.25 2.25 0 002.25-2.25V6a2.25 2.25 0 00-2.25-2.25H6A2.25 2.25 0 003.75 6v2.25A2.25 2.25 0 006 10.5zm0 9.75h2.25A2.25 2.25 0 0010.5 18v-2.25a2.25 2.25 0 00-2.25-2.25H6a2.25 2.25 0 00-2.25 2.25V18A2.25 2.25 0 006 20.25zm9.75-9.75H18a2.25 2.25 0 002.25-2.25V6A2.25 2.25 0 0018 3.75h-2.25A2.25 2.25 0 0013.5 6v2.25a2.25 2.25 0 002.25 2.25z" /></svg>
        <div class="text-[12px] text-gray-500">{t.pluginEmpty || '暂无已安装插件'}</div>
        <div class="text-[11px] text-gray-400 mt-1">{t.pluginEmptyHint || '通过上方输入框安装第一个插件'}</div>
      </div>
    {:else if filteredPlugins.length === 0}
      <div class="py-12 text-center">
        <svg class="w-8 h-8 text-gray-200 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
        <div class="text-[12px] text-gray-400">{t.noSearchResults || '无匹配结果'}</div>
        <button class="text-[11px] text-gray-500 hover:text-gray-700 mt-1 cursor-pointer" onclick={() => searchQuery = ''}>{t.clearSearch || '清除搜索'}</button>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each filteredPlugins as p}
          <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="w-1.5 h-1.5 rounded-full flex-shrink-0 {p.enabled ? 'bg-emerald-500' : 'bg-gray-300'}"></span>
                  <span class="text-[12px] font-medium text-gray-900 truncate">{p.name}</span>
                  <span class="text-[11px] text-gray-400">v{p.version}</span>
                  {#if p.category}
                    <span class="text-[10px] px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{p.category}</span>
                  {/if}
                </div>
                <div class="text-[11px] text-gray-500 mt-0.5 truncate pl-3.5">{p.description}</div>
                {#if p.tags?.length}
                  <div class="flex gap-1 mt-1 flex-wrap pl-3.5">
                    {#each p.tags as tag}
                      <span class="text-[10px] px-1.5 py-0.5 bg-gray-50 text-gray-500 rounded">#{tag}</span>
                    {/each}
                  </div>
                {/if}
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <button
                  onclick={() => handleToggle(p)}
                  disabled={actionLoading === p.name}
                  class="relative w-9 h-5 rounded-full transition-colors cursor-pointer {p.enabled ? 'bg-emerald-500' : 'bg-gray-300'}"
                  title={p.enabled ? (t.pluginClickDisable || '点击禁用') : (t.pluginClickEnable || '点击启用')}
                >
                  <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform {p.enabled ? 'translate-x-4' : 'translate-x-0'}"></span>
                </button>

                {#if p.config_schema && Object.keys(p.config_schema).length > 0}
                  <button
                    onclick={() => showConfig(p)}
                    class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
                    title={t.pluginConfig || '配置'}
                  >
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                  </button>
                {/if}

                <button
                  onclick={() => handleUpdate(p)}
                  disabled={actionLoading === p.name + '-update'}
                  class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
                  title={t.pluginUpdate || '更新'}
                >
                  <svg class="w-3.5 h-3.5 {actionLoading === p.name + '-update' ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>
                </button>

                <button
                  onclick={() => showUninstallConfirm(p)}
                  disabled={actionLoading === p.name + '-uninstall'}
                  class="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg cursor-pointer transition-colors"
                  title={t.pluginUninstall || '卸载'}
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
  {/if}

  {#if activeView === 'market'}
  <!-- Plugin Market -->
  <div class="bg-white rounded-xl border border-gray-100">
    {#if registryLoading && registryPlugins.length === 0}
      <div class="flex items-center justify-center gap-2 py-12">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
      </div>
    {:else if registryPlugins.length === 0}
      <div class="py-12 text-center">
        <svg class="w-10 h-10 text-gray-200 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" /></svg>
        <div class="text-[12px] text-gray-500">{t.pluginMarketEmpty || '暂无可用插件'}</div>
        <div class="text-[11px] text-gray-400 mt-1">{t.pluginMarketSource || '插件来源: redc.wgpsec.org'}</div>
      </div>
    {:else if filteredRegistry.length === 0}
      <div class="py-12 text-center">
        <svg class="w-8 h-8 text-gray-200 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
        <div class="text-[12px] text-gray-400">{t.noSearchResults || '无匹配结果'}</div>
        <button class="text-[11px] text-gray-500 hover:text-gray-700 mt-1 cursor-pointer" onclick={() => searchQuery = ''}>{t.clearSearch || '清除搜索'}</button>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each filteredRegistry as rp}
          <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-[12px] font-medium text-gray-900 truncate">{rp.name}</span>
                  <span class="text-[11px] text-gray-400">v{rp.version}</span>
                  {#if rp.category}
                    <span class="text-[10px] px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{rp.category}</span>
                  {/if}
                </div>
                <div class="text-[11px] text-gray-500 mt-0.5">{rp.description}</div>
                {#if rp.author}
                  <div class="text-[11px] text-gray-400 mt-0.5">by {rp.author}</div>
                {/if}
                {#if rp.tags?.length}
                  <div class="flex gap-1 mt-1 flex-wrap">
                    {#each rp.tags as tag}
                      <span class="text-[10px] px-1.5 py-0.5 bg-gray-50 text-gray-500 rounded">#{tag}</span>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="shrink-0">
                {#if installedMap.has(rp.name)}
                  {#if compareVersions(rp.version, installedMap.get(rp.name)) > 0}
                    <div class="flex items-center gap-2">
                      <span class="text-[11px] text-gray-400">v{installedMap.get(rp.name)} → v{rp.version}</span>
                      <button
                        onclick={() => handleUpdateFromRegistry(rp.name)}
                        disabled={actionLoading === rp.name + '-market-update'}
                        class="px-3 py-1.5 text-[12px] bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors flex items-center gap-1"
                      >
                        {#if actionLoading === rp.name + '-market-update'}
                          <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                        {/if}
                        {t.pluginUpdate || '更新'}
                      </button>
                    </div>
                  {:else}
                    <span class="px-2.5 py-1 text-[11px] text-emerald-600 bg-emerald-50 rounded-lg">{t.pluginAlreadyInstalled || '已安装'}</span>
                  {/if}
                {:else}
                  <button
                    onclick={() => handleInstallFromRegistry(rp.url)}
                    disabled={actionLoading === rp.url}
                    class="px-3 py-1.5 text-[12px] bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors flex items-center gap-1"
                  >
                    {#if actionLoading === rp.url}
                      <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                    {/if}
                    {t.pluginInstall || '安装'}
                  </button>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Manual install -->
  <div class="bg-white rounded-xl border border-gray-100 p-4">
    <h3 class="text-[12px] font-medium text-gray-700 mb-2">{t.pluginInstallManual || '手动安装'}</h3>
    <div class="flex gap-2">
      <input
        type="text"
        bind:value={installSource}
        placeholder={t.pluginInstallPlaceholder || 'Git 仓库地址或本地路径'}
        class="flex-1 px-3 py-2 text-[12px] border border-gray-200 rounded-lg bg-white focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
        onkeydown={(e) => e.key === 'Enter' && handleInstall()}
      />
      <button
        onclick={handleInstall}
        disabled={installing || !installSource.trim()}
        class="px-4 py-2 bg-gray-900 text-white text-[12px] rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors flex items-center gap-1.5"
      >
        {#if installing}
          <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        {/if}
        {t.pluginInstall || '安装'}
      </button>
    </div>
  </div>
  {/if}
</div>

<!-- Config Modal -->
<Modal show={configModal.show} onclose={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false, formValues: {}, useForm: false }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-medium text-gray-900">{t.pluginConfig || '插件配置'} — {configModal.plugin?.name}</h3>
        <button class="p-1 text-gray-400 hover:text-gray-600 cursor-pointer rounded-lg hover:bg-gray-100 transition-colors" onclick={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false, formValues: {}, useForm: false }}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      <div class="p-4">
        {#if configModal.useForm && configModal.schema}
          <div class="space-y-3">
            {#each Object.entries(configModal.schema) as [key, field]}
              <div>
                <label class="block text-[12px] font-medium text-gray-700 mb-1">
                  {key}
                  {#if field.required}<span class="text-red-500 ml-0.5">*</span>{/if}
                  {#if field.type && field.type !== 'string'}
                    <span class="text-gray-300 ml-1 text-[10px]">{field.type}</span>
                  {/if}
                </label>
                {#if field.description}
                  <div class="text-[11px] text-gray-400 mb-1">{field.description}</div>
                {/if}
                {#if field.type === 'boolean'}
                  <button
                    type="button"
                    class="h-8 flex items-center gap-2 px-3 rounded-lg bg-gray-50 cursor-pointer"
                    onclick={() => updateConfigFormValue(key, !(configModal.formValues[key] === true || configModal.formValues[key] === 'true'))}
                  >
                    <div class="relative w-8 h-[18px] rounded-full transition-colors {(configModal.formValues[key] === true || configModal.formValues[key] === 'true') ? 'bg-gray-900' : 'bg-gray-300'}">
                      <div class="absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white shadow transition-transform {(configModal.formValues[key] === true || configModal.formValues[key] === 'true') ? 'translate-x-[16px]' : 'translate-x-[2px]'}"></div>
                    </div>
                    <span class="text-[12px] text-gray-600">{(configModal.formValues[key] === true || configModal.formValues[key] === 'true') ? 'true' : 'false'}</span>
                  </button>
                {:else if field.type === 'number'}
                  <input
                    type="number"
                    class="w-full h-9 px-3 text-[13px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-gray-900"
                    placeholder={field.default || ''}
                    value={configModal.formValues[key] ?? ''}
                    oninput={(e) => updateConfigFormValue(key, e.currentTarget.value ? Number(e.currentTarget.value) : '')}
                  />
                {:else}
                  <input
                    type="text"
                    class="w-full h-9 px-3 text-[13px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-gray-900"
                    placeholder={field.default || ''}
                    value={configModal.formValues[key] ?? ''}
                    oninput={(e) => updateConfigFormValue(key, e.currentTarget.value)}
                  />
                {/if}
              </div>
            {/each}
          </div>
        {:else}
          <textarea
            bind:value={configModal.config}
            rows="8"
            class="w-full px-3 py-2 text-[12px] font-mono border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900 resize-none"
            placeholder="JSON config..."
          ></textarea>
        {/if}
      </div>
      <div class="px-4 py-3 border-t border-gray-100 flex justify-end gap-2">
        <button
          onclick={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false, formValues: {}, useForm: false }}
          class="px-3 py-1.5 text-[12px] text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer"
        >{t.cancel || '取消'}</button>
        <button
          onclick={saveConfig}
          disabled={configModal.saving}
          class="px-3 py-1.5 text-[12px] bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
        >{configModal.saving ? (t.saving || '保存中...') : (t.save || '保存')}</button>
      </div>
    </div>
</Modal>

<!-- Confirm Modal -->
<Modal show={confirmModal.show} onclose={() => confirmModal = { show: false, action: '', pluginName: '', message: '' }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="p-5 text-center">
        <div class="w-10 h-10 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-3">
          <svg class="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" /></svg>
        </div>
        <div class="text-[13px] text-gray-700">{confirmModal.message}</div>
      </div>
      <div class="px-4 py-3 border-t border-gray-100 flex justify-end gap-2">
        <button
          onclick={() => confirmModal = { show: false, action: '', pluginName: '', message: '' }}
          class="px-3 py-1.5 text-[12px] text-gray-600 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer"
        >{t.cancel || '取消'}</button>
        <button
          onclick={handleConfirmAction}
          class="px-3 py-1.5 text-[12px] bg-red-600 text-white rounded-lg hover:bg-red-700 cursor-pointer"
        >{t.pluginConfirmBtn || '确认卸载'}</button>
      </div>
    </div>
</Modal>
