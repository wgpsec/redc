<script>
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { onMount } from 'svelte';
  
  let { t, version, updateStatus, onCheckUpdate } = $props();
  
  let changelog = $state([]);
  let loading = $state(true);
  let expandedVersions = $state(new Set());
  
  onMount(async () => {
    try {
      const res = await fetch('/changelog.json');
      const data = await res.json();
      changelog = data.changelog || [];
      if (changelog.length > 0) {
        expandedVersions = new Set([changelog[0].version]);
      }
    } catch (e) {
      console.error('Failed to load changelog:', e);
    } finally {
      loading = false;
    }
  });

  function toggleVersion(version) {
    const next = new Set(expandedVersions);
    if (next.has(version)) { next.delete(version); } else { next.add(version); }
    expandedVersions = next;
  }

  function openLink(url) { BrowserOpenURL(url); }

  function getChangeType(text) {
    if (text.startsWith('新增：') || text.startsWith('新增:')) return { type: 'new', label: '新增', color: 'bg-emerald-50 text-emerald-700' };
    if (text.startsWith('修复：') || text.startsWith('修复:')) return { type: 'fix', label: '修复', color: 'bg-red-50 text-red-700' };
    if (text.startsWith('优化：') || text.startsWith('优化:')) return { type: 'improve', label: '优化', color: 'bg-blue-50 text-blue-700' };
    return { type: 'other', label: '变更', color: 'bg-gray-100 text-gray-600' };
  }

  function getChangeContent(text) {
    return text.replace(/^(新增|修复|优化)[：:]/, '');
  }

  const features = [
    { icon: 'cloud', title: 'multiCloud', desc: 'multiCloudDesc', titleFallback: '多云支持', descFallback: '支持阿里云、腾讯云、华为云、火山云等主流云平台' },
    { icon: 'bolt', title: 'quickDeploy', desc: 'quickDeployDesc', titleFallback: '快速部署', descFallback: '一键部署各种红队场景，节省时间和精力' },
    { icon: 'template', title: 'templateManage', desc: 'templateManageDesc', titleFallback: '模板管理', descFallback: '丰富的模板库，支持自定义和分享模板' },
    { icon: 'clock', title: 'scheduledTasks', desc: 'scheduledTasksFeatureDesc', titleFallback: '定时任务', descFallback: '支持定时启动和停止场景，自动化管理' },
    { icon: 'cost', title: 'costEstimate', desc: 'costEstimateFeatureDesc', titleFallback: '成本估算', descFallback: '实时估算云资源成本，控制预算' },
    { icon: 'ai', title: 'aiIntegration', desc: 'aiIntegrationFeatureDesc', titleFallback: 'AI 集成', descFallback: '支持 MCP 协议，与 AI 助手无缝集成' },
  ];

  const highlights = [
    { icon: 'shield', text: 'aboutDesc1', fallback: '专为红队设计的云基础设施管理平台，简化和自动化云环境中的部署和管理工作' },
    { icon: 'server', text: 'aboutDesc2', fallback: '快速部署 C2 服务器、钓鱼平台、漏洞环境等，支持多云平台' },
    { icon: 'code', text: 'aboutDesc3', fallback: '基于 Terraform 构建，无需深入了解语法即可轻松管理云基础设施' },
  ];

  const links = [
    { url: 'https://github.com/wgpsec/redc', icon: 'github', title: 'GitHub', sub: 'github.com/wgpsec/redc' },
    { url: 'https://redc.wgpsec.org', icon: 'doc', titleKey: 'documentation', titleFallback: '文档', sub: 'redc.wgpsec.org' },
    { url: 'https://www.wgpsec.org', icon: 'web', titleKey: 'team', titlePrefix: 'WgpSec ', titleFallback: '团队', sub: 'www.wgpsec.org' },
  ];
</script>

<div class="space-y-4">
  <!-- Header: Brand + Version + Developers + License -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="w-12 h-12 rounded-xl flex items-center justify-center" style="background-color: #ed0040;">
          <span class="text-white text-xl font-bold">R</span>
        </div>
        <div>
          <div class="flex items-center gap-2.5">
            <h1 class="text-lg font-bold text-gray-900">RedC</h1>
            <span class="px-2 py-0.5 bg-gray-100 rounded text-[11px] font-medium text-gray-600">{version || 'v3.0.7'}</span>
            <span class="px-2 py-0.5 bg-gray-50 rounded text-[11px] text-gray-400">MIT</span>
            {#if updateStatus && updateStatus.checking}
              <svg class="w-3.5 h-3.5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            {:else if updateStatus && updateStatus.result}
              {#if updateStatus.result.hasUpdate}
                <button
                  class="px-2 py-0.5 text-[11px] bg-red-50 text-red-600 rounded hover:bg-red-100 transition-colors cursor-pointer"
                  onclick={() => BrowserOpenURL(updateStatus.result.downloadURL)}
                >{updateStatus.result.latestVersion} {t.updateAvailable || '可更新'}</button>
              {:else}
                <span class="text-[11px] text-emerald-600">✓ {t.alreadyLatest || '已是最新'}</span>
              {/if}
            {:else}
              <button
                class="text-[11px] text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                onclick={() => onCheckUpdate && onCheckUpdate()}
              >{t.checkUpdate || '检查更新'}</button>
            {/if}
          </div>
          <p class="text-[12px] text-gray-500 mt-0.5">Red Team Cloud Infrastructure Management Platform</p>
        </div>
      </div>
      <!-- Developers -->
      <div class="flex items-center gap-2">
        <button
          class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => openLink('https://github.com/No-Github')}
        >
          <svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" /></svg>
          <span class="text-[12px] font-medium text-gray-600">r0fus0d</span>
        </button>
        <button
          class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => openLink('https://github.com/keac')}
        >
          <svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" /></svg>
          <span class="text-[12px] font-medium text-gray-600">keac</span>
        </button>
      </div>
    </div>
  </div>

  <!-- Highlights (replaces text wall) -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <h2 class="text-[13px] font-semibold text-gray-900 mb-3">{t.aboutIntro || '项目简介'}</h2>
    <div class="space-y-2.5">
      {#each highlights as h}
        <div class="flex items-start gap-3">
          <div class="w-7 h-7 rounded-lg bg-gray-50 flex items-center justify-center flex-shrink-0 mt-0.5">
            {#if h.icon === 'shield'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
            {:else if h.icon === 'server'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" /></svg>
            {:else}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5" /></svg>
            {/if}
          </div>
          <p class="text-[12px] text-gray-600 leading-relaxed">{t[h.text] || h.fallback}</p>
        </div>
      {/each}
    </div>
  </div>

  <!-- Features (unified gray icons) -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <h2 class="text-[13px] font-semibold text-gray-900 mb-3">{t.coreFeatures || '核心特性'}</h2>
    <div class="grid grid-cols-3 gap-3">
      {#each features as f}
        <div class="flex items-start gap-2.5 p-3 rounded-lg bg-gray-50/50">
          <div class="w-7 h-7 rounded-lg bg-white border border-gray-100 flex items-center justify-center flex-shrink-0">
            {#if f.icon === 'cloud'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" /></svg>
            {:else if f.icon === 'bolt'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" /></svg>
            {:else if f.icon === 'template'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" /></svg>
            {:else if f.icon === 'clock'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            {:else if f.icon === 'cost'}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818l.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
            {:else}
              <svg class="w-3.5 h-3.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" /></svg>
            {/if}
          </div>
          <div class="min-w-0">
            <h3 class="text-[12px] font-medium text-gray-900 mb-0.5">{t[f.title] || f.titleFallback}</h3>
            <p class="text-[11px] text-gray-500 leading-relaxed">{t[f.desc] || f.descFallback}</p>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <!-- Links (horizontal 3-column) -->
  <div class="grid grid-cols-3 gap-4">
    {#each links as link}
      <button
        class="bg-white rounded-xl border border-gray-100 p-4 flex items-center gap-3 hover:border-gray-200 hover:shadow-sm transition-all cursor-pointer group text-left"
        onclick={() => openLink(link.url)}
      >
        <div class="w-9 h-9 rounded-lg bg-gray-900 flex items-center justify-center flex-shrink-0">
          {#if link.icon === 'github'}
            <svg class="w-4.5 h-4.5 text-white" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" /></svg>
          {:else if link.icon === 'doc'}
            <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
          {:else}
            <svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" /></svg>
          {/if}
        </div>
        <div class="min-w-0">
          <div class="text-[12px] font-medium text-gray-900">{link.titlePrefix || ''}{link.titleKey ? (t[link.titleKey] || link.titleFallback) : link.title}</div>
          <div class="text-[11px] text-gray-400 truncate">{link.sub}</div>
        </div>
        <svg class="w-4 h-4 text-gray-300 group-hover:text-gray-500 transition-colors ml-auto flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25" /></svg>
      </button>
    {/each}
  </div>

  <!-- Changelog -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <h2 class="text-[13px] font-semibold text-gray-900 mb-3">{t.changelog || '更新日志'}</h2>
    
    {#if loading}
      <div class="flex items-center gap-2 py-8 justify-center">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
      </div>
    {:else if changelog.length === 0}
      <div class="text-center py-8">
        <svg class="w-8 h-8 text-gray-300 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" /></svg>
        <p class="text-[12px] text-gray-400">{t.noChangelog || '暂无更新日志'}</p>
      </div>
    {:else}
      <div class="space-y-1.5">
        {#each changelog as item}
          <div class="border border-gray-100 rounded-lg overflow-hidden">
            <button
              class="w-full flex items-center justify-between px-4 py-2.5 hover:bg-gray-50/50 transition-colors cursor-pointer"
              onclick={() => toggleVersion(item.version)}
            >
              <div class="flex items-center gap-2">
                <svg
                  class="w-3.5 h-3.5 text-gray-400 transition-transform duration-200 {expandedVersions.has(item.version) ? 'rotate-90' : ''}"
                  fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
                ><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
                <span class="px-2 py-0.5 bg-gray-900 text-white text-[11px] font-medium rounded">{item.version}</span>
                <span class="text-[11px] text-gray-400">{item.date}</span>
              </div>
              <span class="text-[11px] text-gray-400">{item.changes.length} {t.changesCount || '项更新'}</span>
            </button>
            {#if expandedVersions.has(item.version)}
              <div class="px-4 pb-3 border-t border-gray-50">
                <ul class="space-y-1.5 mt-2.5">
                  {#each item.changes as change}
                    {@const ct = getChangeType(change)}
                    <li class="flex items-start gap-2 text-[12px] text-gray-600">
                      <span class="px-1.5 py-0 rounded text-[10px] font-medium flex-shrink-0 mt-0.5 {ct.color}">{ct.label}</span>
                      <span class="leading-relaxed">{getChangeContent(change)}</span>
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
