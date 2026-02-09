<script>

  import { onMount } from 'svelte';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { Environment } from '../../../wailsjs/runtime/runtime.js';
  import { ListProjects, GetCurrentProject, SwitchProject, CreateProject } from '../../../wailsjs/go/main/App.js';

let { 
    t, 
    activeTab, 
    lang, 
    onTabChange, 
    onToggleLang, 
    onLoadMCPStatus, 
    onLoadResourceSummary
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
      alert('切换项目失败: ' + err.message);
    }
  }

  // Create a new project
  async function handleCreateProject() {
    if (!newProjectName.trim()) {
      alert('请输入项目名称');
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
      alert('创建项目失败: ' + err.message);
    }
  }
  
  // Use a getter function to ensure we always reference the current prop values
  const navItems = $derived([
    { id: 'dashboard', icon: 'dashboard', labelKey: 'dashboard' },
    { id: 'cases', icon: 'cases', labelKey: 'cases' },
    { id: 'console', icon: 'console', labelKey: 'console' },
    { id: 'resources', icon: 'resources', labelKey: 'resources', onClick: () => onLoadResourceSummary() },
    { id: 'compose', icon: 'compose', labelKey: 'compose' },
    { id: 'credentials', icon: 'credentials', labelKey: 'credentials' },
    { id: 'registry', icon: 'registry', labelKey: 'registry' },
    { id: 'localTemplates', icon: 'localTemplates', labelKey: 'localTemplates' },
    { id: 'specialModules', icon: 'specialModules', labelKey: 'specialModules' },
    { id: 'ai', icon: 'ai', labelKey: 'ai', onClick: () => onLoadMCPStatus() },
    { id: 'settings', icon: 'settings', labelKey: 'settings' }
  ]);

  const icons = {
    dashboard: 'M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z',
    cases: 'M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4',
    console: 'M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z',
    resources: 'M3 7.5l9 4.5 9-4.5M3 12l9 4.5 9-4.5M3 16.5l9 4.5 9-4.5',
    compose: 'M3.75 6A2.25 2.25 0 016 3.75h12A2.25 2.25 0 0120.25 6v12A2.25 2.25 0 0118 20.25H6A2.25 2.25 0 013.75 18V6z M8 8h8M8 12h8M8 16h5',
    credentials: 'M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z',
    registry: 'M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z',
    localTemplates: 'M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z',
    specialModules: 'M11.42 15.17L17.25 21A2.25 2.25 0 0020 18.75V8.25A2.25 2.25 0 0017.75 6H11.42M6.75 6h.008v.008H6.75V6zm2.25 0h.008v.008H9V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.0088v.008h-.008V6zm2.25 0h.008v.008h-.008V6zM6.75 8.25h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 10.5h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.0088v.008h-.008v-.008zM6.75 12.75h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 15h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 17.25h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 19.5h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008z',
    ai: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z',
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
      <div class="w-6 h-6 rounded-md bg-gradient-to-br from-rose-500 to-red-600 flex items-center justify-center">
        <span class="text-white text-[13px] font-bold">C</span>
      </div>
    </div>
  </div>
  
  <!-- Navigation -->
  <nav class="flex-1 p-2">
    <div class="space-y-0.5">
      {#each navItems as item}
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === item.id ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          onclick={() => handleNavClick(item)}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d={icons[item.icon]} />
          </svg>
          {t[item.labelKey]}
        </button>
      {/each}
    </div>
  </nav>

  <!-- Project Switcher -->
  <div class="p-2 border-t border-gray-100">
    <div class="relative">
      <button
        class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
          bg-gray-50 hover:bg-gray-100 text-gray-700 border border-gray-200"
        onclick={() => showProjectDropdown = !showProjectDropdown}
        title={lang === 'zh' ? '切换项目' : 'Switch Project'}
      >
        <svg class="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
        </svg>
        <span class="flex-1 text-left truncate">{currentProject || (lang === 'zh' ? '选择项目...' : 'Select Project...')}</span>
        <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
        </svg>
      </button>

      {#if showProjectDropdown}
        <div class="absolute bottom-full left-0 right-0 mb-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto z-50">
          {#if isLoadingProjects}
            <div class="px-3 py-2 text-[11px] text-gray-500">{lang === 'zh' ? '加载中...' : 'Loading...'}</div>
          {:else if projects.length === 0}
            <div class="px-3 py-2 text-[11px] text-gray-500">{lang === 'zh' ? '暂无项目' : 'No projects'}</div>
          {:else}
            {#each projects as project}
              <button
                class="w-full flex items-center gap-2 px-3 py-2 text-[12px] transition-colors hover:bg-gray-50 {project.name === currentProject ? 'bg-gray-50 text-rose-600 font-medium' : 'text-gray-700'}"
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
            class="w-full flex items-center gap-2 px-3 py-2 text-[12px] text-rose-600 hover:bg-rose-50 transition-colors"
            onclick={() => { showProjectDropdown = false; showNewProjectModal = true; }}
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            {lang === 'zh' ? '新建项目' : 'New Project'}
          </button>
        </div>
      {/if}
    </div>
  </div>

  <!-- New Project Modal -->
  {#if showNewProjectModal}
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg p-4 w-64 shadow-xl">
        <h3 class="text-[13px] font-medium text-gray-900 mb-3">{lang === 'zh' ? '新建项目' : 'New Project'}</h3>
        <input
          type="text"
          bind:value={newProjectName}
          placeholder={lang === 'zh' ? '输入项目名称...' : 'Enter project name...'}
          class="w-full px-3 py-2 text-[12px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-rose-500 focus:border-transparent"
          onkeydown={(e) => e.key === 'Enter' && handleCreateProject()}
        />
        <div class="flex justify-end gap-2 mt-3">
          <button
            class="px-3 py-1.5 text-[11px] text-gray-600 hover:bg-gray-100 rounded transition-colors"
            onclick={() => { showNewProjectModal = false; newProjectName = ''; }}
          >
            {lang === 'zh' ? '取消' : 'Cancel'}
          </button>
          <button
            class="px-3 py-1.5 text-[11px] bg-rose-600 text-white hover:bg-rose-700 rounded transition-colors disabled:opacity-50"
            onclick={handleCreateProject}
            disabled={!newProjectName.trim()}
          >
            {lang === 'zh' ? '创建' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Footer -->
  <div class="p-2 border-t border-gray-100">
    <div class="flex items-center justify-between px-2 py-2">
      <span class="text-[10px] text-gray-400">v2.3.0 by WgpSec</span>
      <div class="flex items-center gap-1">
        <button
          class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors text-[10px] font-medium"
          onclick={onToggleLang}
          title={lang === 'zh' ? 'Switch to English' : '切换到中文'}
        >{lang === 'zh' ? 'EN' : '中'}</button>
        <button
          class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
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
