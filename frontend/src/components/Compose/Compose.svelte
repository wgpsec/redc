<script>

  import { ComposePreview, ComposeUp, ComposeDown, SelectComposeFile } from '../../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../../wailsjs/runtime/runtime.js';
  import { loadComposeTemplates } from '../../lib/composeTemplates.js';
  import { composeState, initComposeEvents, dismissComposeStatus, onComposeStateChange } from '../../lib/composeState.js';
  import { onMount, onDestroy, tick } from 'svelte';
  import ELK from 'elkjs/lib/elk.bundled.js';
  import ZoomControls from '../UI/ZoomControls.svelte';
  import Modal from '../UI/Modal.svelte';
  import PageGuide from '../UI/PageGuide.svelte';
  import HelpTooltip from '../UI/HelpTooltip.svelte';

  let { t, onTabChange } = $props();

  let composeTemplates = $state([]);
  let selectedTemplate = $state(null);
  let templatesLoading = $state(true);

  // Reactive mirror of persistent compose state
  let composeStatus = $state(composeState.status);
  let composeAction = $state(composeState.action);
  let composeStatusError = $state(composeState.error);
  let composeLogs = $state(composeState.logs);
  let logContainerEl = $state(null);

  // Destroy confirmation dialog
  let destroyConfirm = $state({ show: false, loading: false });

  let unsubscribe = null;

  onMount(async () => {
    composeTemplates = await loadComposeTemplates();
    templatesLoading = false;

    // Initialize persistent event listeners (only once globally)
    initComposeEvents(EventsOn);

    // Sync from persistent state on mount
    composeStatus = composeState.status;
    composeAction = composeState.action;
    composeStatusError = composeState.error;
    composeLogs = composeState.logs;

    // Subscribe to updates
    unsubscribe = onComposeStateChange((s) => {
      composeStatus = s.status;
      composeAction = s.action;
      composeStatusError = s.error;
      composeLogs = [...composeState.logs];
      if (s.status === 'done') {
        setTimeout(() => previewCompose(), 500);
      }
      tick().then(() => {
        if (logContainerEl) {
          logContainerEl.scrollTop = logContainerEl.scrollHeight;
        }
      });
    });
  });

  onDestroy(() => {
    if (unsubscribe) unsubscribe();
  });

  // State
  let configSource = $state('template'); // 'template' | 'file'
  let composeFilePath = $state('');
  let composeProfiles = $state('');
  let composeSummary = $state(null);
  let composeLoading = $state(false);
  let composeError = $state('');
  let hasManuallyPreviewed = $state(false);
  let lastPreviewedPath = $state('');
  let showAdvanced = $state(false);

  function getStatusBadge(status) {
    switch (status) {
      case 'running': return { color: 'bg-emerald-500', text: t.composeStatusRunning || '运行中', ring: 'ring-emerald-200' };
      case 'stopped': return { color: 'bg-red-500', text: t.composeStatusStopped || '已停止', ring: 'ring-red-200' };
      case 'created': return { color: 'bg-blue-500', text: t.composeStatusCreated || '已创建', ring: 'ring-blue-200' };
      default: return { color: 'bg-gray-300', text: t.composeStatusNotDeployed || '未部署', ring: 'ring-gray-200' };
    }
  }

  // Topology state
  let composeTopoModal = $state(false);
  let elkNodes = $state([]);
  let elkEdges = $state([]);
  let svgViewBox = $state('0 0 800 400');
  let topoZoom = $state(1);

  function getProviderColor(provider) {
    const p = (provider || '').toLowerCase();
    if (p.includes('aliyun') || p.includes('alicloud')) return { border: '#f97316', text: '#ea580c', label: 'Aliyun' };
    if (p.includes('aws') || p.includes('amazon')) return { border: '#f59e0b', text: '#d97706', label: 'AWS' };
    if (p.includes('tencent') || p.includes('tencentcloud')) return { border: '#3b82f6', text: '#2563eb', label: 'Tencent' };
    if (p.includes('volcengine') || p.includes('volcano')) return { border: '#10b981', text: '#059669', label: 'Volcano' };
    if (p.includes('huawei') || p.includes('hcloud')) return { border: '#ef4444', text: '#dc2626', label: 'Huawei' };
    if (p.includes('azure')) return { border: '#0ea5e9', text: '#0284c7', label: 'Azure' };
    if (p.includes('gcp') || p.includes('google')) return { border: '#8b5cf6', text: '#7c3aed', label: 'GCP' };
    return { border: '#6b7280', text: '#4b5563', label: 'Cloud' };
  }

  async function openComposeTopology() {
    composeTopoModal = true;
    elkNodes = [];
    elkEdges = [];
    topoZoom = 1;
    await layoutComposeTopology();
  }

  async function layoutComposeTopology() {
    if (!composeSummary?.services?.length) return;
    try {
      const elk = new ELK();
      const services = composeSummary.services;
      const NODE_W = 200;
      const NODE_H = 58;

      // Build name set for edge filtering (using rawName and name)
      const nameSet = new Set();
      services.forEach(s => { nameSet.add(s.name); nameSet.add(s.rawName); });

      // Build edges from dependsOn
      const rawEdges = [];
      services.forEach(s => {
        if (s.dependsOn && s.dependsOn.length > 0) {
          s.dependsOn.forEach(dep => {
            if (nameSet.has(dep)) {
              rawEdges.push({ from: s.name, to: dep });
            }
          });
        }
      });

      const nodeCount = services.length;
      const nodeSpacing = nodeCount > 8 ? '25' : '35';
      const layerSpacing = nodeCount > 8 ? '40' : '55';

      const graph = {
        id: 'root',
        layoutOptions: {
          'elk.algorithm': 'layered',
          'elk.direction': 'DOWN',
          'elk.spacing.nodeNode': nodeSpacing,
          'elk.layered.spacing.nodeNodeBetweenLayers': layerSpacing,
          'elk.padding': '[top=15,left=15,bottom=15,right=15]',
          'elk.layered.nodePlacement.strategy': 'BRANDES_KOEPF',
        },
        children: services.map(s => ({ id: s.name, width: NODE_W, height: NODE_H })),
        edges: rawEdges.map((e, i) => ({ id: `e${i}`, sources: [e.from], targets: [e.to] })),
      };

      const layout = /** @type {any} */ (await elk.layout(graph));

      const svcMap = {};
      services.forEach(s => { svcMap[s.name] = s; });

      const newNodes = (layout.children || []).map(n => {
        const svc = svcMap[n.id] || {};
        const color = getProviderColor(svc.provider);
        return {
          id: n.id,
          x: n.x, y: n.y, w: n.width, h: n.height,
          svc,
          color,
          label: svc.name || n.id,
          sublabel: svc.template || '',
          replicas: svc.replicas || 1,
        };
      });

      const newEdges = (layout.edges || []).map(e => {
        const sections = e.sections || [];
        if (sections.length > 0) {
          const s = sections[0];
          return { id: e.id, startPoint: s.startPoint, endPoint: s.endPoint, bendPoints: s.bendPoints || [] };
        }
        const src = newNodes.find(n => n.id === (e.sources?.[0]));
        const tgt = newNodes.find(n => n.id === (e.targets?.[0]));
        if (src && tgt) {
          return { id: e.id, startPoint: { x: src.x + src.w / 2, y: src.y + src.h }, endPoint: { x: tgt.x + tgt.w / 2, y: tgt.y }, bendPoints: [] };
        }
        return null;
      }).filter(Boolean);

      const padding = 40;
      const maxX = Math.max(...newNodes.map(n => n.x + n.w), 400) + padding * 2;
      const maxY = Math.max(...newNodes.map(n => n.y + n.h), 200) + padding * 2;

      elkNodes = newNodes;
      elkEdges = newEdges;
      svgViewBox = `0 0 ${maxX} ${maxY}`;
    } catch (e) {
      console.error('ELK layout failed:', e);
    }
  }

  function edgePath(edge) {
    const { startPoint, endPoint, bendPoints } = edge;
    if (bendPoints.length === 0) {
      const midY = (startPoint.y + endPoint.y) / 2;
      return `M ${startPoint.x} ${startPoint.y} C ${startPoint.x} ${midY}, ${endPoint.x} ${midY}, ${endPoint.x} ${endPoint.y}`;
    }
    let d = `M ${startPoint.x} ${startPoint.y}`;
    const pts = [startPoint, ...bendPoints, endPoint];
    for (let i = 1; i < pts.length; i++) {
      const prev = pts[i - 1]; const curr = pts[i];
      const midY = (prev.y + curr.y) / 2;
      d += ` C ${prev.x} ${midY}, ${curr.x} ${midY}, ${curr.x} ${curr.y}`;
    }
    return d;
  }

  // Functions
  function handleSelectTemplate() {
    if (!selectedTemplate) {
      composeFilePath = '';
      return;
    }
    const template = composeTemplates.find(t => t.name === selectedTemplate);
    if (template && template.path) {
      composeFilePath = template.path + '/redc-compose.yaml';
      hasManuallyPreviewed = true;
      previewCompose();
    }
  }

  async function handleBrowseFile() {
    try {
      const selectedPath = await SelectComposeFile();
      if (selectedPath) {
        composeFilePath = selectedPath;
        hasManuallyPreviewed = true;
        previewCompose();
      }
    } catch (e) {
      console.error('Failed to select file:', e);
    }
  }

  function parseComposeProfiles(value) {
    if (!value) return [];
    return value
      .split(',')
      .map(v => v.trim())
      .filter(Boolean);
  }

  // Auto-preview when file path or profiles change (only after first manual preview)
  let timer = null;
  
  async function previewCompose() {
    if (!composeFilePath) {
      composeError = '';
      composeSummary = null;
      return;
    }
    
    hasManuallyPreviewed = true;
    composeLoading = true;
    composeError = '';
    try {
      composeSummary = await ComposePreview(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
      composeSummary = null;
    } finally {
      composeLoading = false;
    }
  }

  $effect(() => {
    if (hasManuallyPreviewed && composeFilePath && composeFilePath !== lastPreviewedPath) {
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        lastPreviewedPath = composeFilePath;
        previewCompose();
      }, 500);
    }
  });

  export async function handleComposeUp() {
    if (!composeFilePath) {
      composeError = t.composeFile + ' ' + t.paramRequired;
      return;
    }
    
    composeError = '';
    composeLogs = [];
    try {
      await ComposeUp(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
    }
  }

  function showDestroyConfirm() {
    if (!composeFilePath) {
      composeError = t.composeFile + ' ' + t.paramRequired;
      return;
    }
    destroyConfirm = { show: true, loading: false };
  }

  function cancelDestroy() {
    destroyConfirm = { show: false, loading: false };
  }

  export async function handleComposeDown() {
    if (!composeFilePath) {
      composeError = t.composeFile + ' ' + t.paramRequired;
      return;
    }
    
    destroyConfirm = { ...destroyConfirm, loading: true };
    composeError = '';
    composeLogs = [];
    try {
      await ComposeDown(composeFilePath, parseComposeProfiles(composeProfiles));
      destroyConfirm = { show: false, loading: false };
    } catch (e) {
      composeError = e.message || String(e);
      destroyConfirm = { show: false, loading: false };
    }
  }

  function dismissStatus() {
    dismissComposeStatus();
  }

</script>

<div class="space-y-4">
  <PageGuide text={t.pgCompose} dismissKey="compose" />
  <!-- No templates hint -->
  {#if !templatesLoading && composeTemplates.length === 0}
    <div class="bg-blue-50 border border-blue-100 rounded-xl p-5">
      <div class="flex items-start gap-3">
        <svg class="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
        </svg>
        <div class="flex-1">
          <p class="text-[13px] text-blue-700">{t.noComposeTemplatesHint}</p>
          <button
            class="mt-3 h-8 px-4 bg-blue-500 text-white text-[12px] font-medium rounded-lg hover:bg-blue-600 transition-colors cursor-pointer"
            onclick={() => onTabChange && onTabChange('registry')}
          >
            {t.noComposeTemplatesHintButton}
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Two-column layout -->
  <div class="flex flex-col lg:flex-row gap-4">
    <!-- LEFT COLUMN: Configuration -->
    <div class="w-full lg:w-[340px] lg:flex-shrink-0 space-y-4">
      <div class="bg-white rounded-xl border border-gray-100 p-5">
        <div class="text-[13px] font-semibold text-gray-900 mb-4">{t.composeFile || '配置来源'}</div>

        <!-- Radio: From Template -->
        <label class="flex items-center gap-2.5 cursor-pointer group mb-3">
          <input type="radio" name="configSource" value="template" bind:group={configSource}
            class="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500 cursor-pointer" />
          <span class="text-[13px] text-gray-700 group-hover:text-gray-900 transition-colors">{t.composeSourceTemplate || '从模板选择'}</span>
        </label>
        {#if configSource === 'template'}
          <div class="ml-6 mb-4">
            <select
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow"
              bind:value={selectedTemplate}
              onchange={handleSelectTemplate}
            >
              <option value={null}>{templatesLoading ? (t.loading || '加载中...') : (t.selectTemplate || '请选择模板')}</option>
              {#each composeTemplates as tmpl}
                <option value={tmpl.name}>{tmpl.nameZh || tmpl.name}</option>
              {/each}
            </select>
            {#if selectedTemplate}
              {@const currentTemplate = composeTemplates.find(t => t.name === selectedTemplate)}
              {#if currentTemplate?.description}
                <p class="mt-2 text-[11px] text-gray-400 leading-relaxed">{currentTemplate.description}</p>
              {/if}
            {/if}
          </div>
        {/if}

        <!-- Radio: Custom File -->
        <label class="flex items-center gap-2.5 cursor-pointer group">
          <input type="radio" name="configSource" value="file" bind:group={configSource}
            class="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500 cursor-pointer" />
          <span class="text-[13px] text-gray-700 group-hover:text-gray-900 transition-colors">{t.composeSourceFile || '自定义文件路径'}</span>
        </label>
        {#if configSource === 'file'}
          <div class="ml-6 mt-2">
            <div class="flex gap-1.5">
              <input
                type="text"
                placeholder="redc-compose.yaml"
                class="flex-1 h-9 px-3 text-[12px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow font-mono"
                bind:value={composeFilePath}
              />
              <button
                class="h-9 px-3 bg-gray-100 text-gray-600 text-[11px] font-medium rounded-lg hover:bg-gray-200 transition-colors flex-shrink-0"
                onclick={handleBrowseFile}
              >
                {t.browseFile}
              </button>
            </div>
          </div>
        {/if}
      </div>

      <!-- Advanced Options (collapsible) -->
      <div class="bg-white rounded-xl border border-gray-100">
        <button
          class="w-full flex items-center justify-between px-5 py-3.5 text-[12px] font-medium text-gray-500 hover:text-gray-700 transition-colors"
          onclick={() => showAdvanced = !showAdvanced}
        >
          <span>{t.composeAdvanced || '高级选项'}</span>
          <svg class="w-4 h-4 transition-transform {showAdvanced ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
          </svg>
        </button>
        {#if showAdvanced}
          <div class="px-5 pb-4 border-t border-gray-50">
            <label class="block text-[11px] font-medium text-gray-500 mb-1.5 mt-3">{t.composeProfiles} <HelpTooltip text={t.helpComposeProfiles} /></label>
            <input
              type="text"
              placeholder="prod, dev"
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow"
              bind:value={composeProfiles}
            />
          </div>
        {/if}
      </div>

      {#if composeError}
        <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border border-red-100 rounded-xl text-[12px] text-red-700">
          <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
          <span class="flex-1">{composeError}</span>
          <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => composeError = ''}>
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
      {/if}
    </div>

    <!-- RIGHT COLUMN: Services + Actions + Logs -->
    <div class="flex-1 min-w-0 space-y-4">
      {#if composeLoading}
        <!-- Loading -->
        <div class="bg-white rounded-xl border border-gray-100 p-5">
          <div class="flex items-center justify-center h-32">
            <div class="flex flex-col items-center gap-3">
              <div class="w-7 h-7 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
              <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
            </div>
          </div>
        </div>
      {:else if composeSummary?.services?.length > 0}
        <!-- Service list -->
        <div class="bg-white rounded-xl border border-gray-100 p-5">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <span class="text-[14px] font-semibold text-gray-900">{t.composePreview}</span>
              <span class="text-[11px] text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full">{composeSummary.total} {t.composeSvcCount || '个服务'}</span>
            </div>
            <button
              class="flex items-center gap-1.5 h-7 px-3 text-[11px] font-medium text-gray-600 bg-gray-50 hover:bg-gray-100 border border-gray-200 rounded-lg transition-colors cursor-pointer"
              onclick={openComposeTopology}
            >
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
              </svg>
              {t.composeTopology || '拓扑视图'}
            </button>
            <HelpTooltip text={t.helpComposeTopo} />
          </div>
          <div class="border border-gray-100 rounded-lg overflow-hidden">
            <table class="w-full text-[12px]">
              <thead>
                <tr class="bg-gray-50 border-b border-gray-100">
                  <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.serviceName}</th>
                  <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.serviceTemplate}</th>
                  <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.serviceProvider}</th>
                  <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.serviceStatus || '状态'}</th>
                  <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.serviceDepends}</th>
                  <th class="text-right px-4 py-2.5 font-medium text-gray-500">{t.serviceReplicas}</th>
                </tr>
              </thead>
              <tbody>
                {#each composeSummary.services as svc}
                  {@const badge = getStatusBadge(svc.status)}
                  <tr class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors">
                    <td class="px-4 py-3 text-gray-800 font-medium">{svc.name}</td>
                    <td class="px-4 py-3 text-gray-600 font-mono text-[11px]">{svc.template}</td>
                    <td class="px-4 py-3 text-gray-600">{svc.provider || '-'}</td>
                    <td class="px-4 py-3">
                      <span class="inline-flex items-center gap-1.5">
                        <span class="w-2 h-2 rounded-full {badge.color} ring-2 {badge.ring}"></span>
                        <span class="text-[11px] text-gray-600">{badge.text}</span>
                      </span>
                    </td>
                    <td class="px-4 py-3 text-gray-600">{(svc.dependsOn?.length > 0) ? svc.dependsOn.join(', ') : '-'}</td>
                    <td class="px-4 py-3 text-right text-gray-600">{svc.replicas || 1}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

        <!-- Action buttons -->
        <div class="flex items-center gap-3">
          <button
            class="h-10 px-6 bg-emerald-500 text-white text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 inline-flex items-center gap-2 shadow-sm cursor-pointer"
            onclick={handleComposeUp}
            disabled={composeStatus === 'running'}
          >
            {#if composeStatus === 'running' && composeAction === 'up'}
              <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              {t.composeDeploying || '正在部署...'}
            {:else}
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" /></svg>
              {t.composeUp}
            {/if}
          </button>
          <button
            class="h-10 px-5 text-red-600 bg-white border border-red-200 text-[13px] font-medium rounded-lg hover:bg-red-50 transition-colors disabled:opacity-50 inline-flex items-center gap-2 cursor-pointer"
            onclick={showDestroyConfirm}
            disabled={composeStatus === 'running'}
          >
            {#if composeStatus === 'running' && composeAction === 'down'}
              <div class="w-4 h-4 border-2 border-red-200 border-t-red-600 rounded-full animate-spin"></div>
              {t.composeDestroying || '正在销毁...'}
            {:else}
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
              {t.composeDown}
            {/if}
          </button>
        </div>
      {:else}
        <!-- Empty state guidance -->
        <div class="bg-white rounded-xl border border-gray-100 p-5">
          <div class="flex flex-col items-center justify-center py-16 text-center">
            <div class="w-16 h-16 rounded-2xl bg-gray-50 flex items-center justify-center mb-4">
              <svg class="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75L2.25 12l4.179 2.25m0-4.5l5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L12 12.75 6.429 9.75m11.142 0l4.179 2.25-4.179 2.25m0 0L12 17.25l-5.571-3m11.142 0l4.179 2.25L12 21.75l-9.75-5.25 4.179-2.25" />
              </svg>
            </div>
            <p class="text-[13px] text-gray-400 max-w-xs">{t.composeGuideHint || '选择 Compose 模板或指定文件路径开始编排'}</p>
          </div>
        </div>
      {/if}

      <!-- Compose operation status & log panel -->
      {#if composeStatus !== 'idle'}
        <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
          <!-- Status header -->
          <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between {composeStatus === 'running' ? 'bg-blue-50' : composeStatus === 'done' ? 'bg-emerald-50' : 'bg-red-50'}">
            <div class="flex items-center gap-2.5">
              {#if composeStatus === 'running'}
                <div class="w-4 h-4 border-2 border-blue-200 border-t-blue-600 rounded-full animate-spin"></div>
                <span class="text-[13px] font-medium text-blue-700">
                  {composeAction === 'up' ? (t.composeDeploying || '正在部署编排...') : (t.composeDestroying || '正在销毁编排...')}
                </span>
              {:else if composeStatus === 'done'}
                <svg class="w-4 h-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
                <span class="text-[13px] font-medium text-emerald-700">
                  {composeAction === 'up' ? (t.composeUpDone || '编排部署完成') : (t.composeDownDone || '编排销毁完成')}
                </span>
              {:else if composeStatus === 'error'}
                <svg class="w-4 h-4 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                <span class="text-[13px] font-medium text-red-700">
                  {composeAction === 'up' ? (t.composeUpFailed || '编排部署失败') : (t.composeDownFailed || '编排销毁失败')}
                </span>
              {/if}
            </div>
            {#if composeStatus !== 'running'}
              <button
                class="text-[11px] text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
                onclick={dismissStatus}
              >{t.dismiss || '关闭'}</button>
            {/if}
          </div>

          <!-- Error message -->
          {#if composeStatus === 'error' && composeStatusError}
            <div class="px-5 py-3 bg-red-50 border-b border-red-100 text-[12px] text-red-600">{composeStatusError}</div>
          {/if}

          <!-- Log panel -->
          {#if composeLogs.length > 0}
            <div
              class="bg-gray-900 px-4 py-3 max-h-56 overflow-y-auto font-mono text-[11px] leading-5"
              bind:this={logContainerEl}
            >
              {#each composeLogs as log}
                <div class="text-gray-300">
                  <span class="text-gray-500 select-none">{log.time}</span>
                  <span class="ml-2">{log.message}</span>
                </div>
              {/each}
              {#if composeStatus === 'running'}
                <div class="text-gray-500 animate-pulse">▍</div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<Modal show={composeTopoModal} onclose={() => composeTopoModal = false}>
    <div class="bg-white rounded-2xl shadow-2xl w-full max-w-[1100px] mx-4 max-h-[90vh] flex flex-col overflow-hidden" onclick={e => e.stopPropagation()}>
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <div class="flex items-center gap-3">
          <svg class="w-5 h-5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
          </svg>
          <span class="text-[15px] font-semibold text-gray-900">{t.composeTopology || '编排服务拓扑'}</span>
          <span class="text-[13px] text-gray-400">— {composeFilePath || 'redc-compose.yaml'}</span>
        </div>
        <button class="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => composeTopoModal = false}>
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      <div class="flex-1 overflow-y-auto px-6 py-4 space-y-4">
        <!-- Stats bar -->
        <div class="flex items-center gap-4 text-[12px] text-gray-500">
          <span class="font-medium text-gray-700">{composeSummary?.total || 0} {t.composeSvcCount || '个服务'}</span>
          {#if elkEdges.length > 0}
            <span>· {elkEdges.length} {t.composeDepsCount || '条依赖'}</span>
          {/if}
          {#if composeSummary?.services}
            {@const providers = [...new Set(composeSummary.services.map(s => (s.provider||'').split(/[,\s]/)[0]).filter(Boolean))]}
            {#if providers.length > 0}
              <span>·</span>
              {#each providers as p}
                {@const color = getProviderColor(p)}
                <span class="inline-flex items-center gap-1">
                  <span class="w-2 h-2 rounded-full inline-block" style="background:{color.border}"></span>
                  <span>{p}</span>
                </span>
              {/each}
            {/if}
          {/if}
        </div>

        <!-- Provider legend -->
        {#if composeSummary?.services}
          {@const providerGroups = composeSummary.services.reduce((acc, s) => {
            const p = (s.provider||'').split(/[,\s]/)[0] || 'unknown';
            if (!acc[p]) acc[p] = 0;
            acc[p]++;
            return acc;
          }, {})}
          <div class="flex flex-wrap gap-2">
            {#each Object.entries(providerGroups) as [p, cnt]}
              {@const color = getProviderColor(p)}
              <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium border" style="border-color:{color.border}; color:{color.text}; background:{color.border}18">
                <span class="w-2 h-2 rounded-full inline-block" style="background:{color.border}"></span>
                {p} × {cnt}
              </span>
            {/each}
          </div>
        {/if}

        <!-- Topology SVG -->
        {#if elkNodes.length > 0}
          <div class="relative">
            <ZoomControls bind:zoom={topoZoom} {t} />
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="bg-gray-50 rounded-xl border border-gray-200 overflow-auto" style="max-height: 65vh;"
              onwheel={(e) => { e.preventDefault(); topoZoom = Math.min(3, Math.max(0.3, topoZoom + (e.deltaY > 0 ? -0.08 : 0.08))); }}>
              <svg viewBox={svgViewBox} preserveAspectRatio="xMidYMid meet" class="w-full" style="min-height: 300px; transform: scale({topoZoom}); transform-origin: center top;">
                <defs>
                  <marker id="compose-arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
                    <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
                  </marker>
                </defs>

                {#each elkEdges as edge}
                  <path d={edgePath(edge)} fill="none" stroke="#cbd5e1" stroke-width="1.5" marker-end="url(#compose-arrowhead)" class="transition-colors hover:stroke-red-400" />
                {/each}

                {#each elkNodes as node}
                  <g transform="translate({node.x}, {node.y})">
                    <rect width={node.w} height={node.h} rx="8" ry="8"
                      fill="white" stroke={node.color.border} stroke-width="2" class="drop-shadow-sm" />
                    <circle cx="14" cy="22" r="4" fill={node.color.border} />
                    <text x="26" y="18" font-size="11" font-weight="600" fill="#374151" font-family="system-ui, sans-serif">
                      {node.label}
                    </text>
                    <text x="26" y="32" font-size="9" fill="#9ca3af" font-family="system-ui, sans-serif">
                      {node.sublabel}
                    </text>
                    <text x="26" y="46" font-size="8" fill={node.color.text} font-family="system-ui, sans-serif" font-style="italic">
                      {node.svc?.provider || ''}
                    </text>
                    {#if node.replicas > 1}
                      <rect x={node.w - 30} y="6" width="22" height="14" rx="7" fill={node.color.border} />
                      <text x={node.w - 19} y="17" font-size="9" font-weight="700" fill="white" text-anchor="middle" font-family="system-ui, sans-serif">
                        ×{node.replicas}
                      </text>
                    {/if}
                  </g>
                {/each}
              </svg>
            </div>
          </div>
        {:else if composeSummary?.services?.length > 0}
          <!-- Fallback: service list -->
          <div class="space-y-2">
            {#each composeSummary.services as svc}
              {@const color = getProviderColor(svc.provider)}
              <div class="flex items-center gap-3 px-3 py-2 rounded-lg border" style="border-color:{color.border}40; background:{color.border}08">
                <span class="w-2.5 h-2.5 rounded-full flex-shrink-0" style="background:{color.border}"></span>
                <span class="text-[12px] font-medium text-gray-700">{svc.name}</span>
                <span class="text-[11px] text-gray-400">{svc.template}</span>
                {#if svc.dependsOn?.length > 0}
                  <span class="text-[10px] text-gray-400">← {svc.dependsOn.join(', ')}</span>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-gray-100 flex justify-end">
        <button class="h-9 px-5 bg-gray-100 text-gray-700 text-[12px] font-medium rounded-lg hover:bg-gray-200 transition-colors cursor-pointer" onclick={() => composeTopoModal = false}>
          {t.close || '关闭'}
        </button>
      </div>
    </div>
</Modal>

<Modal show={destroyConfirm.show} onclose={cancelDestroy}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 overflow-hidden" onclick={e => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
          </div>
          <div>
            <div class="text-[14px] font-semibold text-gray-900">{t.composeDestroyConfirmTitle || '确认销毁编排'}</div>
            <div class="text-[12px] text-gray-500 mt-0.5">{t.composeDestroyConfirmDesc || '此操作将销毁所有编排服务和关联资源，不可撤销'}</div>
          </div>
        </div>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelDestroy}
          disabled={destroyConfirm.loading}
        >{t.cancel}</button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 inline-flex items-center gap-1.5"
          onclick={handleComposeDown}
          disabled={destroyConfirm.loading}
        >
          {#if destroyConfirm.loading}
            <div class="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
          {/if}
          {t.composeDestroyConfirm || '确认销毁'}
        </button>
      </div>
    </div>
</Modal>
