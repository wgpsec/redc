<script>

  import { onMount } from 'svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { Environment } from '../../../wailsjs/runtime/runtime.js';
  import { ListProjects, GetCurrentProject, SwitchProject, CreateProject } from '../../../wailsjs/go/main/App.js';
  import { toast } from '../../lib/toast.js';

let { 
    t, 
    activeTab, 
    lang,
    version,
    onTabChange, 
    onToggleLang, 
    onLoadMCPStatus, 
    onLoadResourceSummary,
    onCheckUpdate
  } = $props();
  
  // Project switching state - managed internally
  let projects = $state([]);
  let currentProject = $state('');
  let projectLoading = $state(false);
  
  // Detect platform and fullscreen mode
  let isMac = $state(false);
  let isFullscreen = $state(false);

  // Project switching state (projects and currentProject are already declared in props)
  let showProjectDropdown = $state(false);
  let showNewProjectModal = $state(false);
  let newProjectName = $state('');
  let isLoadingProjects = $state(false);
  
  onMount(() => {
    // Detect platform and load projects
    (async () => {
      try {
        const env = await Environment();
        isMac = env.platform === 'darwin';
      } catch (e) {
        // Fallback: detect from user agent
        isMac = navigator.platform.toLowerCase().includes('mac');
      }

      // Load projects
      await loadProjects();
    })();
    
    const checkFullscreen = () => {
      isFullscreen = window.innerHeight === window.screen.height && window.innerWidth === window.screen.width;
    };
    
    checkFullscreen();
    window.addEventListener('resize', checkFullscreen);
    
    return () => {
      window.removeEventListener('resize', checkFullscreen);
    };
  });
  
  // Compute left padding: only add padding on macOS when not fullscreen
  const leftPadding = $derived(isMac && !isFullscreen ? 'pl-24' : '');

  // Load projects list
  async function loadProjects() {
    isLoadingProjects = true;
    try {
      const [projectsList, current] = await Promise.all([
        ListProjects(),
        GetCurrentProject()
      ]);
      projects = projectsList || [];
      currentProject = current || '';
    } catch (err) {
      console.error('Failed to load projects:', err);
    } finally {
      isLoadingProjects = false;
    }
  }

  // Switch to a different project
  async function handleSwitchProject(projectName) {
    if (projectName === currentProject) {
      showProjectDropdown = false;
      return;
    }
    try {
      await SwitchProject(projectName);
      currentProject = projectName;
      showProjectDropdown = false;
      // Trigger a page refresh to reload data
      window.location.reload();
    } catch (err) {
      console.error('Failed to switch project:', err);
      toast.error((t.switchProjectFailed || '切换项目失败') + ': ' + err.message);
    }
  }

  // Create a new project
  async function handleCreateProject() {
    if (!newProjectName.trim()) {
      toast.warning(t.pleaseEnterProjectName || '请输入项目名称');
      return;
    }
    try {
      await CreateProject(newProjectName.trim());
      showNewProjectModal = false;
      newProjectName = '';
      await loadProjects();
      // Switch to the new project
      await handleSwitchProject(newProjectName.trim());
    } catch (err) {
      console.error('Failed to create project:', err);
      toast.error((t.createProjectFailed || '创建项目失败') + ': ' + err.message);
    }
  }
  
  // Use a getter function to ensure we always reference the current prop values
  const navSections = $derived([
    { type: 'item', id: 'dashboard', icon: 'dashboard', labelKey: 'dashboard' },
    { type: 'group', key: 'deploy', labelKey: 'navGroupDeploy', items: [
      { id: 'cases', icon: 'cases', labelKey: 'cases' },
      { id: 'customDeployment', icon: 'customDeployment', labelKey: 'customDeployment' },
      { id: 'compose', icon: 'compose', labelKey: 'compose' },
      { id: 'userdataScripts', icon: 'userdataScripts', labelKey: 'userdataScripts' },
    ]},
    { type: 'group', key: 'ops', labelKey: 'navGroupOps', items: [
      { id: 'console', icon: 'console', labelKey: 'console' },
      { id: 'resources', icon: 'resources', labelKey: 'resources', onClick: () => onLoadResourceSummary() },
      { id: 'sshManager', icon: 'sshManager', labelKey: 'sshManager' },
      { id: 'taskCenter', icon: 'taskCenter', labelKey: 'taskCenter' },
    ]},
    { type: 'group', key: 'template', labelKey: 'navGroupTemplate', items: [
      { id: 'registry', icon: 'registry', labelKey: 'registry' },
      { id: 'localTemplates', icon: 'localTemplates', labelKey: 'localTemplates' },
    ]},
    { type: 'group', key: 'ai', labelKey: 'navGroupAI', items: [
      { id: 'ai', icon: 'ai', labelKey: 'ai', onClick: () => onLoadMCPStatus() },
      { id: 'aiChat', icon: 'aiChat', labelKey: 'aiChat' },
      { id: 'agentMemory', icon: 'agentMemory', labelKey: 'agentMemory' },
    ]},
    { type: 'group', key: 'system', labelKey: 'navGroupSystem', items: [
      { id: 'credentials', icon: 'credentials', labelKey: 'credentials' },
      { id: 'pluginManager', icon: 'pluginManager', labelKey: 'pluginManager' },
      { id: 'httpServer', icon: 'httpServer', labelKey: 'httpServer' },
      { id: 'settings', icon: 'settings', labelKey: 'settings' },
    ]},
  ]);

  // Collapsible group state — persisted to localStorage
  let collapsedGroups = $state(loadCollapsedGroups());

  function loadCollapsedGroups() {
    try {
      const saved = localStorage.getItem('redc-sidebar-collapsed');
      return saved ? JSON.parse(saved) : {};
    } catch { return {}; }
  }

  function toggleGroup(key) {
    collapsedGroups = { ...collapsedGroups, [key]: !collapsedGroups[key] };
    localStorage.setItem('redc-sidebar-collapsed', JSON.stringify(collapsedGroups));
  }

  function isGroupActive(items) {
    return items.some(item => item.id === activeTab);
  }

  // Auto-expand group when its child becomes active
  $effect(() => {
    for (const section of navSections) {
      if (section.type === 'group' && collapsedGroups[section.key]) {
        if (section.items.some(item => item.id === activeTab)) {
          collapsedGroups = { ...collapsedGroups, [section.key]: false };
          localStorage.setItem('redc-sidebar-collapsed', JSON.stringify(collapsedGroups));
          break;
        }
      }
    }
  });

  const icons = {
    dashboard: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z',
    cases: 'M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4',
    customDeployment: 'M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z',
    console: 'M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z',
    resources: 'M3 7.5l9 4.5 9-4.5M3 12l9 4.5 9-4.5M3 16.5l9 4.5 9-4.5',
    compose: 'M3.75 6A2.25 2.25 0 016 3.75h12A2.25 2.25 0 0120.25 6v12A2.25 2.25 0 0118 20.25H6A2.25 2.25 0 013.75 18V6z M8 8h8M8 12h8M8 16h5',
    credentials: 'M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z',
    registry: 'M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z',
    localTemplates: 'M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z',
    userdataScripts: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z',
    ai: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456z',
    aiChat: 'M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155',
    sshManager: 'M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z',
    taskCenter: 'M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z',
    pluginManager: 'M13.5 16.875h3.375m0 0h3.375m-3.375 0V13.5m0 3.375v3.375M6 10.5h2.25a2.25 2.25 0 002.25-2.25V6a2.25 2.25 0 00-2.25-2.25H6A2.25 2.25 0 003.75 6v2.25A2.25 2.25 0 006 10.5zm0 9.75h2.25A2.25 2.25 0 0010.5 18v-2.25a2.25 2.25 0 00-2.25-2.25H6a2.25 2.25 0 00-2.25 2.25V18A2.25 2.25 0 006 20.25zm9.75-9.75H18a2.25 2.25 0 002.25-2.25V6A2.25 2.25 0 0018 3.75h-2.25A2.25 2.25 0 0013.5 6v2.25a2.25 2.25 0 002.25 2.25z',
    httpServer: 'M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5a17.92 17.92 0 0 1-8.716-2.247m0 0A8.966 8.966 0 0 1 3 12c0-1.264.26-2.466.732-3.558',
    agentMemory: 'M12 18v-5.25m0 0a6.01 6.01 0 001.5-.189m-1.5.189a6.01 6.01 0 01-1.5-.189m3.75 7.478a12.06 12.06 0 01-4.5 0m3.75 2.383a14.406 14.406 0 01-3 0M14.25 18v-.192c0-.983.658-1.823 1.508-2.316a7.5 7.5 0 10-7.517 0c.85.493 1.509 1.333 1.509 2.316V18',
    settings: 'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z M15 12a3 3 0 11-6 0 3 3 0 016 0z'
  };

  function handleNavClick(item) {
    onTabChange(item.id);
    if (item.onClick) {
      item.onClick();
    }
  }

  function openGitHub() {
    BrowserOpenURL('https://github.com/wgpsec/redc');
  }

</script>

<aside class="w-44 bg-white border-r border-gray-100 flex flex-col overflow-hidden overscroll-none">
  <!-- Logo -->
  <div class="h-14 flex items-center px-4 border-b border-gray-100 {leftPadding}" style="--wails-draggable:drag">
    <div class="flex items-center gap-0.5">
      <span class="text-[14px] font-semibold text-gray-900">Red</span>
      <div class="w-6 h-6 rounded-md bg-rose-600 flex items-center justify-center">
        <span class="text-white text-[13px] font-bold">C</span>
      </div>
    </div>
  </div>
  
  <!-- Navigation -->
  <nav class="flex-1 p-2 overflow-y-auto">
    <div class="space-y-0.5">
      {#each navSections as section}
        {#if section.type === 'item'}
          <button 
            class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all whitespace-nowrap
              {activeTab === section.id ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50 cursor-pointer'}"
            onclick={() => handleNavClick(section)}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d={icons[section.icon]} />
            </svg>
            {t[section.labelKey]}
          </button>
        {:else}
          <!-- Group header -->
          <button
            class="w-full flex items-center justify-between px-2.5 pt-3 pb-1 cursor-pointer group"
            onclick={() => toggleGroup(section.key)}
          >
            <span class="text-[10px] uppercase tracking-wider font-semibold transition-colors
              {isGroupActive(section.items) ? 'text-gray-600' : 'text-gray-400 group-hover:text-gray-500'}">
              {t[section.labelKey]}
            </span>
            <svg class="w-3 h-3 text-gray-300 group-hover:text-gray-400 transition-all {collapsedGroups[section.key] ? '-rotate-90' : ''}"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
            </svg>
          </button>
          <!-- Group items -->
          {#if !collapsedGroups[section.key]}
            {#each section.items as item}
              <button 
                class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all whitespace-nowrap
                  {activeTab === item.id ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50 cursor-pointer'}"
                onclick={() => handleNavClick(item)}
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d={icons[item.icon]} />
                </svg>
                {t[item.labelKey]}
              </button>
            {/each}
          {/if}
        {/if}
      {/each}
    </div>
  </nav>

  <!-- Project Switcher -->
  <div class="p-2 border-t border-gray-100">
    <div class="relative">
      <button
        class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
          bg-gray-50 hover:bg-gray-100 text-gray-700 border border-gray-100 cursor-pointer"
        onclick={() => showProjectDropdown = !showProjectDropdown}
        title={t.switchProject || '切换项目'}
      >
        <svg class="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
        </svg>
        <span class="flex-1 text-left truncate">{currentProject || (t.selectProject || '选择项目...')}</span>
        <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
        </svg>
      </button>

      {#if showProjectDropdown}
        <div class="absolute bottom-full left-0 right-0 mb-1 bg-white border border-gray-100 rounded-lg border border-gray-200 max-h-48 overflow-y-auto z-50">
          {#if isLoadingProjects}
            <div class="px-3 py-2 text-[11px] text-gray-500">{t.loadingProjects || '加载中...'}</div>
          {:else if projects.length === 0}
            <div class="px-3 py-2 text-[11px] text-gray-500">{t.noProjects || '暂无项目'}</div>
          {:else}
            {#each projects as project}
              <button
                class="w-full flex items-center gap-2 px-3 py-2 text-[12px] transition-colors hover:bg-gray-50 cursor-pointer {project.name === currentProject ? 'bg-gray-50 text-rose-600 font-medium' : 'text-gray-700'}"
                onclick={() => handleSwitchProject(project.name)}
              >
                {#if project.name === currentProject}
                  <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                {:else}
                  <span class="w-3"></span>
                {/if}
                <span class="truncate">{project.name}</span>
              </button>
            {/each}
          {/if}
          <div class="border-t border-gray-100 my-1"></div>
          <button
            class="w-full flex items-center gap-2 px-3 py-2 text-[12px] text-rose-600 hover:bg-rose-50 transition-colors cursor-pointer"
            onclick={() => { showProjectDropdown = false; showNewProjectModal = true; }}
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            {t.newProject || '新建项目'}
          </button>
        </div>
      {/if}
    </div>
  </div>

  <!-- New Project Modal -->
  {#if showNewProjectModal}
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-visible">
      <div class="bg-white rounded-lg p-4 w-64 border border-gray-200">
        <h3 class="text-[13px] font-medium text-gray-900 mb-3">{t.newProject || '新建项目'}</h3>
        <input
          type="text"
          bind:value={newProjectName}
          placeholder={t.enterProjectName || '输入项目名称...'}
          class="w-full px-3 py-2 text-[12px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-rose-500 focus:border-transparent"
          onkeydown={(e) => e.key === 'Enter' && handleCreateProject()}
        />
        <div class="flex justify-end gap-2 mt-3">
          <button
            class="px-3 py-1.5 text-[11px] text-gray-600 hover:bg-gray-100 rounded transition-colors cursor-pointer"
            onclick={() => { showNewProjectModal = false; newProjectName = ''; }}
          >
            {t.cancel || '取消'}
          </button>
          <button
            class="px-3 py-1.5 text-[11px] bg-rose-600 text-white hover:bg-rose-700 rounded transition-colors disabled:opacity-50 cursor-pointer"
            onclick={handleCreateProject}
            disabled={!newProjectName.trim()}
          >
            {t.create || '创建'}
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Footer -->
  <div class="p-2 border-t border-gray-100">
    <div class="flex items-center justify-between px-2 py-2">
      <button
        class="text-[10px] text-gray-400 hover:text-gray-600 hover:bg-gray-50 px-2 py-1 rounded transition-colors whitespace-nowrap cursor-pointer"
        onclick={() => { if (onCheckUpdate) onCheckUpdate(); onTabChange('about'); }}
        title={t.aboutRedC || '关于 RedC，点击检查更新'}
      >
        {version || 'v3.0.7'} by WgpSec
      </button>
      <div class="flex items-center gap-1">
        <button
          class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors text-[10px] font-medium cursor-pointer"
          onclick={onToggleLang}
          title={t.switchLanguage || '切换语言'}
        >{lang === 'zh' ? (t.langEn || 'EN') : (t.langZh || '中')}</button>
        <button
          class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
          onclick={openGitHub}
          title="GitHub"
        >
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
        </button>
      </div>
    </div>
  </div>
</aside>
