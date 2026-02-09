<script>

  import { onMount, onDestroy } from 'svelte';
  import { i18n as i18nData } from './lib/i18n.js';
  import { EventsOn, EventsOff, WindowMinimise, WindowMaximise, WindowUnmaximise, WindowIsMaximised, Quit, Environment } from '../wailsjs/runtime/runtime.js';
  import { ListCases, ListTemplates, GetConfig, GetMCPStatus, StartMCPServer, StopMCPServer, GetResourceSummary, GetBalances, GetTerraformMirrorConfig, GetNotificationEnabled, GetCurrentProject, ListProjects, SwitchProject, CreateProject } from '../wailsjs/go/main/App.js';
  import Console from './components/Console/Console.svelte';
  import CloudResources from './components/Resources/CloudResources.svelte';
  import Compose from './components/Compose/Compose.svelte';
  import AIIntegration from './components/AI/AIIntegration.svelte';
  import SpecialModules from './components/SpecialModules/SpecialModules.svelte';
  import Registry from './components/Registry/Registry.svelte';
  import Credentials from './components/Credentials/Credentials.svelte';
  import LocalTemplates from './components/LocalTemplates/LocalTemplates.svelte';
  import Dashboard from './components/Dashboard/Dashboard.svelte';
  import Settings from './components/Settings/Settings.svelte';
  import Sidebar from './components/Sidebar/Sidebar.svelte';

  let cases = $state([]);
  let templates = $state([]);
  let logs = $state([]);
  let config = $state({ redcPath: '', projectPath: '', logPath: '', httpProxy: '', httpsProxy: '', noProxy: '', debugEnabled: false });
  let activeTab = $state('dashboard');
  let isLoading = $state(false);
  let error = $state('');
  let terraformMirror = $state({ enabled: false, configPath: '', managed: false, fromEnv: false, providers: [] });
  let notificationEnabled = $state(false);
  let debugEnabled = $state(false);
  let isMaximised = $state(false);
  let isWindows = $state(false);

  // MCP state
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });
  let mcpForm = $state({ mode: 'sse', address: 'localhost:8080' });
  let mcpLoading = $state(false);

  // Project state
  let projects = $state([]);
  let currentProject = $state({ name: '', path: '' });
  let projectLoading = $state(false);

  // i18n state
  let lang = $state(localStorage.getItem('lang') || 'zh');
  const i18n = { ...i18nData };
  let t = $derived(i18n[lang]);

  
  // Component references
  let dashboardComponent = $state();
  let cloudResourcesComponent = $state();

  function toggleLang() {
    lang = lang === 'zh' ? 'en' : 'zh';
    localStorage.setItem('lang', lang);
  }

  // Project management functions
  async function loadProjects() {
    try {
      projectLoading = true;
      const projectList = await ListProjects();
      projects = projectList || [];
      // Set current project from the list or default
      if (projects.length > 0) {
        const current = projects.find(p => p.name === redc.Project) || projects[0];
        currentProject = current;
      }
    } catch (e) {
      console.error('加载项目列表失败:', e);
      projects = [];
    } finally {
      projectLoading = false;
    }
  }

  async function handleSwitchProject(projectName) {
    if (projectName === currentProject.name) return;
    
    try {
      projectLoading = true;
      await SwitchProject(projectName);
      currentProject = projects.find(p => p.name === projectName) || { name: projectName, path: '' };
      // Refresh data after project switch
      await refreshData();
    } catch (e) {
      console.error('切换项目失败:', e);
      error = `切换项目失败: ${e.message}`;
    } finally {
      projectLoading = false;
    }
  }

  /**
   * @param {CustomEvent} event
   */
  function handleSwitchTab(event) {
    activeTab = event.detail;
  }

  // Window control functions
  async function minimiseWindow() {
    WindowMinimise();
  }

  async function toggleMaximise() {
    const maximised = await WindowIsMaximised();
    if (maximised) {
      WindowUnmaximise();
      isMaximised = false;
    } else {
      WindowMaximise();
      isMaximised = true;
    }
  }

  function closeWindow() {
    Quit();
  }

  onMount(async () => {
    // 检测平台
    const env = await Environment();
    isWindows = env.platform === 'windows';
    
    EventsOn('log', (message) => {
      logs = [...logs, { time: new Date().toLocaleTimeString(), message }];
      if (dashboardComponent && dashboardComponent.updateCreateStatusFromLog) {
        dashboardComponent.updateCreateStatusFromLog(message);
      }
    });
    EventsOn('refresh', async () => {
      await refreshData();
    });
    
    // Listen for tab switch events from child components
    window.addEventListener('switchTab', handleSwitchTab);
    
    // Check initial maximised state (only for Windows)
    if (isWindows) {
      isMaximised = await WindowIsMaximised();
    }
    
    await refreshData();
    await loadProjects();
  });

  onDestroy(() => {
    EventsOff('log');
    EventsOff('refresh');
    
    // Remove tab switch event listener
    window.removeEventListener('switchTab', handleSwitchTab);
  });

  async function refreshData() {
    isLoading = true;
    error = '';
    try {
      [cases, templates, config, terraformMirror, notificationEnabled] = await Promise.all([
        ListCases(),
        ListTemplates(),
        GetConfig(),
        GetTerraformMirrorConfig(),
        GetNotificationEnabled()
      ]);
      debugEnabled = !!config.debugEnabled;
      // Refresh dashboard component if it exists
      if (dashboardComponent && dashboardComponent.refresh) {
        await dashboardComponent.refresh();
      }
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    } finally {
      isLoading = false;
    }
  }

  // MCP functions
  async function loadMCPStatus() {
    try {
      mcpStatus = await GetMCPStatus();
    } catch (e) {
      console.error('Failed to load MCP status:', e);
    }
  }

  async function handleStartMCP() {
    mcpLoading = true;
    try {
      mcpForm.mode = 'sse';
      await StartMCPServer(mcpForm.mode, mcpForm.address);
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

  async function handleStopMCP() {
    mcpLoading = true;
    try {
      await StopMCPServer();
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

  // Resources functions
  async function loadResourceSummary() {
    if (cloudResourcesComponent && cloudResourcesComponent.loadResourceSummary) {
      await cloudResourcesComponent.loadResourceSummary();
    }
  }


</script>

<div class="h-screen flex bg-[#fafbfc]">
  <!-- Sidebar -->
  <Sidebar 
    {t} 
    {lang}
    {activeTab} 
    onTabChange={(tab) => activeTab = tab}
    onToggleLang={toggleLang}
    onLoadMCPStatus={loadMCPStatus}
    onLoadResourceSummary={loadResourceSummary}
  />

  <!-- Main -->
  <div class="flex-1 flex flex-col min-w-0">
    <!-- Header -->
    <header class="h-14 bg-white border-b border-gray-100 flex items-center justify-between px-6" style="--wails-draggable:drag">
      <h1 class="text-[15px] font-medium text-gray-900">
        {#if activeTab === 'dashboard'}{t.sceneManage}{:else if activeTab === 'console'}{t.console}{:else if activeTab === 'resources'}{t.resources}{:else if activeTab === 'compose'}{t.compose}{:else if activeTab === 'registry'}{t.templateRepo}{:else if activeTab === 'localTemplates'}{t.localTmplManage}{:else if activeTab === 'ai'}{t.aiIntegration}{:else if activeTab === 'credentials'}{t.credentials}{:else if activeTab === 'specialModules'}{t.specialModules}{:else}{t.settings}{/if}
      </h1>
      <div class="flex items-center gap-2" style="--wails-draggable:no-drag">
        <button 
          class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-50 text-gray-400 hover:text-gray-600 transition-colors"
          onclick={() => { refreshData(); if (activeTab === 'ai') loadMCPStatus(); if (activeTab === 'resources') loadResourceSummary(); }}
          title="刷新"
          aria-label="刷新"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
          </svg>
        </button>
        
        <!-- Window Controls (Windows only) -->
        {#if isWindows}
        <div class="flex items-center ml-2 -mr-2">
          <button 
            class="w-12 h-14 flex items-center justify-center hover:bg-gray-100 text-gray-600 transition-colors"
            onclick={minimiseWindow}
            title="最小化"
            aria-label="最小化"
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 12 12">
              <path stroke="currentColor" stroke-width="1" d="M0 6h12"/>
            </svg>
          </button>
          <button 
            class="w-12 h-14 flex items-center justify-center hover:bg-gray-100 text-gray-600 transition-colors"
            onclick={toggleMaximise}
            title={isMaximised ? "还原" : "最大化"}
            aria-label={isMaximised ? "还原" : "最大化"}
          >
            {#if isMaximised}
              <svg class="w-3 h-3" fill="none" viewBox="0 0 12 12">
                <rect x="2" y="2" width="8" height="8" stroke="currentColor" stroke-width="1" fill="none"/>
                <path stroke="currentColor" stroke-width="1" d="M2 2V0h8v8h-2"/>
              </svg>
            {:else}
              <svg class="w-3 h-3" fill="none" viewBox="0 0 12 12">
                <rect x="1" y="1" width="10" height="10" stroke="currentColor" stroke-width="1" fill="none"/>
              </svg>
            {/if}
          </button>
          <button 
            class="w-12 h-14 flex items-center justify-center hover:bg-red-500 hover:text-white text-gray-600 transition-colors"
            onclick={closeWindow}
            title="关闭"
            aria-label="关闭"
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 12 12">
              <path stroke="currentColor" stroke-width="1" d="M1 1l10 10M11 1L1 11"/>
            </svg>
          </button>
        </div>
        {/if}
      </div>
    </header>

    <!-- Content -->
    <main class="flex-1 overflow-auto p-6">
      {#if error}
        <div class="mb-5 flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
          <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
          <span class="text-[13px] text-red-700 flex-1">{error}</span>
          <button class="text-red-400 hover:text-red-600" onclick={() => error = ''} aria-label="关闭错误提示" title="关闭">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      {/if}

      {#if isLoading}
        <div class="flex items-center justify-center h-64">
          <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
        </div>
      {:else if activeTab === 'dashboard'}
        <Dashboard bind:this={dashboardComponent} {t} onTabChange={(tab) => activeTab = tab} />

      {:else if activeTab === 'console'}
        <Console bind:logs {t} />

      {:else if activeTab === 'resources'}
        <CloudResources bind:this={cloudResourcesComponent} {t} />

      {:else if activeTab === 'compose'}
        <Compose {t} />

      {:else if activeTab === 'settings'}
        <Settings {t} bind:config bind:terraformMirror bind:debugEnabled bind:notificationEnabled />

      {:else if activeTab === 'registry'}
        <Registry {t} />

      {:else if activeTab === 'ai'}
        <AIIntegration {t} />

      {:else if activeTab === 'specialModules'}
        <SpecialModules {t} />

      {:else if activeTab === 'credentials'}
        <Credentials {t} />

      {:else if activeTab === 'localTemplates'}
        <LocalTemplates {t} />
      {/if}
    </main>
  </div>
</div>

<style>

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
  }
  :global(select) {
    appearance: none;
    background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
    background-position: right 0.5rem center;
    background-repeat: no-repeat;
    background-size: 1.5em 1.5em;
    padding-right: 2.5rem;
  }

</style>