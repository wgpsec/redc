<script>

  import { onMount, onDestroy } from 'svelte';
  import { ListCases, ListTemplates, StartCase, StopCase, RemoveCase, CreateCase, CreateAndRunCase, GetCaseOutputs, GetTemplateVariables, GetCostEstimate, GetCasePlanPreview, SetCaseTags, GetAllTagNames, CloneCase, ListPlugins } from '../../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../../wailsjs/runtime/runtime.js';
  import { toast } from '../../lib/toast.js';
  import SSHModal from './SSHModal.svelte';
  import ScheduleDialog from './ScheduleDialog.svelte';
  // import ScheduledTasksManager from './ScheduledTasksManager.svelte'; // Moved to TaskCenter
  import ELK from 'elkjs/lib/elk.bundled.js';
  import ZoomControls from '../UI/ZoomControls.svelte';
  import Modal from '../UI/Modal.svelte';

let { t, onTabChange = () => {} } = $props();
  let cases = $state([]);
  let templates = $state([]);
  let selectedTemplate = $state('');
  let newCaseName = $state('');
  let expandedCase = $state(null);
  let caseOutputs = $state({});
  let deleteConfirm = $state({ show: false, caseId: null, caseName: '' });
  let stopConfirm = $state({ show: false, caseId: null, caseName: '' });
  let pluginCheckModal = $state({
    show: false,
    mode: 'single',
    caseName: '',
    caseId: '',
    caseIds: [],
    missingPlugins: [],
    missingByCase: [],
    okCount: 0
  });
  let templateVariables = $state([]);
  let variableValues = $state({});
  let error = $state('');

  // Spot termination/recovery toast
  let spotTerminatedToast = $state({ show: false, caseName: '', downIPs: [], allDown: false, timer: null });
  let spotRecoveryToast = $state({ show: false, caseName: '', status: '', timer: null }); // status: 'recovering' | 'recovered' | 'failed'
  
  // SSH Modal state
  let sshModal = $state({ show: false, caseId: null, caseName: '' });
  
  // Schedule Dialog state
  let scheduleDialog = $state({ show: false, caseId: null, caseName: '', action: '' });
  
  // Scheduled Tasks Manager refresh reference
  // let scheduledTasksManagerRefresh = { current: null }; // Moved to TaskCenter
  
  // Cost estimation state
  let showCostEstimate = $state(false);
  let costEstimate = $state(null);
  let costEstimateLoading = $state(false);
  let costEstimateError = $state('');
  let costEstimateDebounceTimer = null;
  
  // Template list cost estimation state
  let templateCosts = $state({}); // Map of template name to cost estimate
  let templateCostsLoading = $state(new Set()); // Set of template names currently loading
  let allTemplateCostsLoading = $state(false); // Loading state for all templates
  
  // Batch operation state
  let selectedCases = $state(new Set());
  let batchOperating = $state(false);
  let batchDeleteConfirm = $state({ show: false, count: 0 });
  let batchStopConfirm = $state({ show: false, count: 0 });
  
  // Create status state
  let createStatus = $state('idle');
  let createStatusMessage = $state('');
  let createStatusDetail = $state('');
  let createStatusTimer = null;

  // Tag state
  let allTagNames = $state([]);
  let selectedTag = $state('');
  let tagEditCase = $state(null);
  let tagInput = $state('');

  // Context menu state
  let contextMenu = $state({ show: false, x: 0, y: 0, caseId: null, caseName: '', caseState: '', caseType: '' });

  // Search & filter state
  let searchQuery = $state('');
  let statusFilter = $state('all'); // 'all' | 'running' | 'stopped' | 'error'
  let currentPage = $state(1);
  const pageSize = 20;

  // Template search dropdown state
  let templateDropdownOpen = $state(false);
  let templateSearch = $state('');
  let filteredTemplates = $derived((templates || []).filter(tmpl => 
    !templateSearch.trim() || tmpl.name.toLowerCase().includes(templateSearch.trim().toLowerCase())
  ));

  let filteredCases = $derived.by(() => {
    let result = cases;
    // Status filter
    if (statusFilter !== 'all') {
      result = result.filter(c => c.state === statusFilter);
    }
    // Tag filter
    if (selectedTag) {
      result = result.filter(c => c.tags && c.tags.includes(selectedTag));
    }
    // Search query
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      result = result.filter(c =>
        (c.name && c.name.toLowerCase().includes(q)) ||
        (c.id && c.id.toLowerCase().includes(q)) ||
        (c.type && c.type.toLowerCase().includes(q)) ||
        (c.tags && c.tags.some(tag => tag.toLowerCase().includes(q)))
      );
    }
    return result;
  });

  let totalPages = $derived(Math.max(1, Math.ceil(filteredCases.length / pageSize)));
  let paginatedCases = $derived(filteredCases.slice((currentPage - 1) * pageSize, currentPage * pageSize));

  // Reset page when filters change
  let _prevFilterKey = $state('');
  $effect(() => {
    const key = `${searchQuery}|${statusFilter}|${selectedTag}`;
    if (_prevFilterKey && key !== _prevFilterKey) {
      currentPage = 1;
    }
    _prevFilterKey = key;
  });

  // Status counts for tabs
  let statusCounts = $derived({
    all: cases.length,
    running: cases.filter(c => c.state === 'running').length,
    stopped: cases.filter(c => c.state === 'stopped').length,
    error: cases.filter(c => c.state === 'error').length,
  });

  // Clone dialog state
  let cloneDialog = $state({ show: false, caseId: null, caseName: '', sourceName: '' });
  let cloneLoading = $state(false);

  // Running time ticker
  let nowTick = $state(Date.now());
  let tickTimer = null;

  function formatElapsed(fromStr) {
    if (!fromStr) return '';
    const from = new Date(fromStr).getTime();
    if (isNaN(from)) return '';
    const diff = Math.max(0, Math.floor((nowTick - from) / 1000));
    const d = Math.floor(diff / 86400);
    const h = Math.floor((diff % 86400) / 3600);
    const m = Math.floor((diff % 3600) / 60);
    if (d > 0) return `${d}d ${h}h ${m}m`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
  }

  function formatCompactTime(isoStr) {
    if (!isoStr) return '';
    const d = new Date(isoStr);
    if (isNaN(d.getTime())) return isoStr;
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const mi = String(d.getMinutes()).padStart(2, '0');
    return `${mm}-${dd} ${hh}:${mi}`;
  }
  
  // Computed: check if we have persistent error
  let hasPersistentError = $derived(!!getPersistentError());
  
  
  
  // Persistent error that survives refreshes - use window object for global access
  // @ts-ignore
  window.__persistentError = window.__persistentError || null;
  let persistentError = $state(null);
  
  // Sync persistentError from window on mount and periodically
  $effect(() => {
    // @ts-ignore
    persistentError = window.__persistentError;
  });
  
  // Function to get persistent error
  function getPersistentError() {
    return persistentError;
  }
  
  // Function to set persistent error
  function setPersistentError(err) {
    window.__persistentError = err;
    persistentError = err;
  }
  
  // Function to dismiss persistent error
  function dismissPersistentError() {
    console.log('[Cases] Dismissing error');
    setPersistentError(null);
    createStatus = 'idle';
    createStatusMessage = '';
    createStatusDetail = '';
    showErrorDetail = false;
    console.log('[Cases] After dismiss, getPersistentError():', getPersistentError());
  }
  
  // Terraform init hint
  let terraformInitHint = $state({ show: false, message: '', detail: '' });
  let terraformInitHintDismissed = false;
  let terraformInitHintLastDetail = '';

  
  
  // Error display state
  let showErrorDetail = $state(false);
  
  // Plan preview state
  let planPreviewModal = $state({ show: false, caseName: '', caseId: '', loading: false, error: '', data: null });
  let elkNodes = $state([]);
  let elkEdges = $state([]);
  let svgViewBox = $state('0 0 800 600');
  let topoZoom = $state(1);
  
  let copiedKey = $state(null);
  let copiedAllKey = $state(null);
  
  let createBusy = $derived(createStatus === 'creating' || createStatus === 'initializing');

  
  let allSelected = $derived(paginatedCases.length > 0 && paginatedCases.every(c => selectedCases.has(c.id)));

  let someSelected = $derived(selectedCases.size > 0 && !allSelected);

  let hasSelection = $derived(selectedCases.size > 0);

  
  let stateConfig = $derived({
    'running': { label: t.running, color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopped': { label: t.stopped, color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'error': { label: t.error, color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' },
    'created': { label: t.created, color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500' },
    'pending': { label: t.pending, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: t.starting, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'stopping': { label: t.stopping, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'removing': { label: t.removing, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'terminated': { label: t.terminated || '已回收', color: 'text-red-700', bg: 'bg-red-50', dot: 'bg-red-600 animate-pulse' }
  });

  
  onMount(async () => {
    await refresh();
    tickTimer = setInterval(() => { nowTick = Date.now(); }, 60000);
    
    // Close context menu on Escape
    const handleKeydown = (e) => { if (e.key === 'Escape' && contextMenu.show) closeContextMenu(); };
    window.addEventListener('keydown', handleKeydown);
    
    EventsOn('spot-terminated', (data) => {
      console.log('[SpotMonitor] Instance terminated:', data);
      if (spotTerminatedToast.timer) clearTimeout(spotTerminatedToast.timer);
      spotTerminatedToast = {
        show: true,
        caseName: data.caseName || '',
        downIPs: data.downIPs || [],
        allDown: data.allDown || false,
        timer: setTimeout(() => { spotTerminatedToast = { show: false, caseName: '', downIPs: [], allDown: false, timer: null }; }, 15000)
      };
      refresh();
    });

    EventsOn('spot-recovering', (data) => {
      if (spotRecoveryToast.timer) clearTimeout(spotRecoveryToast.timer);
      spotRecoveryToast = { show: true, caseName: data.caseName || '', status: 'recovering', timer: null };
    });

    EventsOn('spot-recovered', (data) => {
      if (spotRecoveryToast.timer) clearTimeout(spotRecoveryToast.timer);
      spotRecoveryToast = {
        show: true, caseName: data.caseName || '', status: 'recovered',
        timer: setTimeout(() => { spotRecoveryToast = { show: false, caseName: '', status: '', timer: null }; }, 10000)
      };
      // Hide terminated toast
      if (spotTerminatedToast.timer) clearTimeout(spotTerminatedToast.timer);
      spotTerminatedToast = { show: false, caseName: '', downIPs: [], allDown: false, timer: null };
      refresh();
    });

    EventsOn('spot-recover-failed', (data) => {
      if (spotRecoveryToast.timer) clearTimeout(spotRecoveryToast.timer);
      spotRecoveryToast = {
        show: true, caseName: data.caseName || '', status: 'failed',
        timer: setTimeout(() => { spotRecoveryToast = { show: false, caseName: '', status: '', timer: null }; }, 15000)
      };
    });
  });
  
  onDestroy(() => {
    if (tickTimer) { clearInterval(tickTimer); tickTimer = null; }
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    if (spotTerminatedToast.timer) {
      clearTimeout(spotTerminatedToast.timer);
    }
    if (spotRecoveryToast.timer) {
      clearTimeout(spotRecoveryToast.timer);
    }
    if (costEstimateDebounceTimer) {
      clearTimeout(costEstimateDebounceTimer);
      costEstimateDebounceTimer = null;
    }
  });
  
  function stripAnsi(value) {
    if (!value) return '';
    return value.replace(/\x1B\[[0-9;]*m/g, '');
  }
  
  function setCreateStatus(status, message, detail = '') {
    // Save error to persistent storage when error occurs
    if (status === 'error') {
      setPersistentError({ message, detail });
    }
    
    // If already showing error, don't overwrite it unless it's a new operation
    if (createStatus === 'error' && status !== 'creating' && status !== 'initializing') {
      return;
    }
    createStatus = status;
    createStatusMessage = message || '';
    createStatusDetail = detail || '';
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    // Error status should stay until user dismisses it or starts a new operation
    // Only auto-clear success status after 3 seconds
    if (status === 'success') {
      createStatusTimer = setTimeout(() => {
        createStatus = 'idle';
        createStatusMessage = '';
        createStatusDetail = '';
      }, 3000);
    }
  }
  
  function detectTerraformInitIssue(message) {
    const lower = message.toLowerCase();
    const hit = lower.includes('registry.terraform.io') || lower.includes('failed to query available provider packages') || lower.includes('x509') || lower.includes('tls') || lower.includes('context deadline') || lower.includes('client.timeout') || lower.includes('could not connect');
    if (hit) {
      if (terraformInitHintDismissed && terraformInitHintLastDetail === message) {
        return;
      }
      terraformInitHintDismissed = false;
      terraformInitHintLastDetail = message;
      terraformInitHint = { show: true, message: t.mirrorDetected, detail: message };
    }
  }
  
  function dismissTerraformInitHint() {
    terraformInitHint = { show: false, message: '', detail: '' };
    terraformInitHintDismissed = true;
  }
  
  // 从模板名称提取云服务商
  function getProviderFromTemplate(templateName) {
    if (!templateName) return 'unknown';
    const parts = templateName.split('/');
    if (parts.length >= 1) {
      const provider = parts[0].toLowerCase();
      const providerMap = {
        'aliyun': 'alicloud',
        'tencent': 'tencentcloud',
        'aws': 'aws',
        'huawei': 'huaweicloud',
        'ucloud': 'ucloud',
        'volcengine': 'volcengine',
        'gcp': 'gcp',
        'ctyun': 'ctyun'
      };
      return providerMap[provider] || provider;
    }
    return 'unknown';
  }
  
  // AI 分析错误 - 跳转到 AI 对话页面
  function handleAIAnalysis() {
    const errorMessage = getPersistentError()?.detail || createStatusDetail;
    if (!errorMessage) {
      toast.warning(t.noErrorToAnalyze || '没有错误信息可以分析');
      return;
    }
    
    let templateName = selectedTemplate || newCaseName || '';
    if (!templateName) {
      const match = errorMessage.match(/模板:\s*(\S+)/);
      if (match) templateName = match[1];
    }
    
    localStorage.setItem('ai-chat-pending-error', JSON.stringify({
      error: errorMessage,
      templateName,
      provider: getProviderFromTemplate(templateName),
      source: 'cases'
    }));
    onTabChange('aiChat');
  }
  
  export async function refresh() {
    // Don't refresh if there's an error status - it will clear the error message
    if (createStatus === 'error') {
      return;
    }
    // Also don't refresh if currently creating/initializing - that would also clear status
    if (createStatus === 'creating' || createStatus === 'initializing') {
      return;
    }
    try {
      [cases, templates] = await Promise.all([
        ListCases(),
        ListTemplates()
      ]);
      allTagNames = await GetAllTagNames().catch(() => []);
      
      // Note: Template list cost preview is now manual (user must click button)
      // This prevents automatic loading of all template costs on page load
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    }
  }
  
  export function updateCreateStatusFromLog(message) {
    const cleanMessage = stripAnsi(message);
    if (cleanMessage.includes(t.sceneCreating || '正在创建场景:') || cleanMessage.includes(t.sceneCreatingAndRunning || '正在创建并运行场景:')) {
      setCreateStatus('creating', t.creating, message);
      return;
    }
    if (cleanMessage.includes(t.sceneInitializing || '场景初始化中:')) {
      setCreateStatus('initializing', t.initializing, message);
      return;
    }
    if (cleanMessage.includes(t.sceneCreateSuccess || '场景创建成功')) {
      setCreateStatus('success', t.createSuccess, message);
      setTimeout(() => refresh(), 500);
      return;
    }
    if (cleanMessage.includes(t.sceneCreateFailed || '场景创建失败') || cleanMessage.includes(t.createSceneError || '创建场景时发生错误')) {
      setCreateStatus('error', t.createFailed, message);
      detectTerraformInitIssue(cleanMessage);
      return;
    }
  }
  
  async function loadTemplateVariables(templateName) {
    if (!templateName) {
      templateVariables = [];
      variableValues = {};
      return;
    }
    try {
      const vars = await GetTemplateVariables(templateName);
      templateVariables = vars || [];
      variableValues = {};
      for (const v of templateVariables) {
        if (v.type === 'bool') {
          variableValues[v.name] = v.defaultValue === 'true' ? 'true' : 'false';
        } else {
          variableValues[v.name] = v.defaultValue || '';
        }
      }
    } catch (e) {
      console.error('Failed to load template variables:', e);
      templateVariables = [];
      variableValues = {};
    }
  }
  
  async function handleCreate() {
    if (!selectedTemplate) {
      error = t.selectTemplateErr;
      return;
    }
    setCreateStatus('creating', t.creating, '');
    try {
      /** @type {Record<string, string>} */
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = String(value);
        }
      }
      await CreateCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
    } catch (e) {
      error = e.message || String(e);
      setCreateStatus('error', t.createFailed, error);
    }
  }
  
  async function handleCreateAndRun() {
    if (!selectedTemplate) {
      error = t.selectTemplateErr;
      return;
    }
    setCreateStatus('creating', t.creating, '');
    try {
      /** @type {Record<string, string>} */
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = String(value);
        }
      }
      await CreateAndRunCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
    } catch (e) {
      error = e.message || String(e);
      setCreateStatus('error', t.createFailed, error);
    }
  }
  
  async function checkMissingPlugins(pluginsStr) {
    if (!pluginsStr || !pluginsStr.trim()) return [];
    try {
      const installed = await ListPlugins();
      const installedNames = new Set(installed.map(p => p.manifest?.name || p.name));
      const required = pluginsStr.split(',').map(s => s.trim()).filter(Boolean);
      return required.filter(name => !installedNames.has(name));
    } catch {
      return [];
    }
  }

  async function handleStart(caseId) {
    const c = cases.find(c => c.id === caseId);
    if (c && c.plugins) {
      const missing = await checkMissingPlugins(c.plugins);
      if (missing.length > 0) {
        pluginCheckModal = {
          show: true, mode: 'single',
          caseName: c.name, caseId,
          caseIds: [], missingPlugins: missing,
          missingByCase: [], okCount: 0
        };
        return;
      }
    }
    doStart(caseId);
  }

  async function doStart(caseId) {
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'starting' } : c);
    try {
      await StartCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }

  function confirmPluginStart() {
    const { mode, caseId, caseIds } = pluginCheckModal;
    pluginCheckModal = { show: false, mode: 'single', caseName: '', caseId: '', caseIds: [], missingPlugins: [], missingByCase: [], okCount: 0 };
    if (mode === 'single') {
      doStart(caseId);
    } else {
      caseIds.forEach(id => doStart(id));
    }
  }

  function cancelPluginCheck() {
    pluginCheckModal = { show: false, mode: 'single', caseName: '', caseId: '', caseIds: [], missingPlugins: [], missingByCase: [], okCount: 0 };
  }

  function goToPluginManager() {
    pluginCheckModal = { show: false, mode: 'single', caseName: '', caseId: '', caseIds: [], missingPlugins: [], missingByCase: [], okCount: 0 };
    onTabChange('pluginManager');
  }

  function showCloneDialog(caseId, caseName) {
    cloneDialog = { show: true, caseId, caseName: caseName + '-clone', sourceName: caseName };
  }

  function cancelClone() {
    cloneDialog = { show: false, caseId: null, caseName: '', sourceName: '' };
  }

  async function confirmClone() {
    const { caseId, caseName } = cloneDialog;
    cloneDialog = { show: false, caseId: null, caseName: '', sourceName: '' };
    cloneLoading = true;
    try {
      await CloneCase(caseId, caseName);
    } catch (e) {
      error = (t.cloneFailed || '克隆失败') + ': ' + (e.message || String(e));
    } finally {
      cloneLoading = false;
    }
  }

  // Plan preview topology
  async function handlePlanPreview(caseId, caseName) {
    planPreviewModal = { show: true, caseName, caseId, loading: true, error: '', data: null };
    elkNodes = [];
    elkEdges = [];
    topoZoom = 1;
    try {
      const data = await GetCasePlanPreview(caseId);
      if (data && data.hasChanges && data.resources && data.resources.length > 0) {
        await layoutTopology(data);
      }
      planPreviewModal = { ...planPreviewModal, data, loading: false };
    } catch (e) {
      planPreviewModal = { ...planPreviewModal, loading: false, error: e.message || String(e) };
    }
  }

  function getActionColor(actions) {
    if (!actions || actions.length === 0) return { border: '#9ca3af', bg: '#f9fafb', text: '#6b7280', label: '?' };
    if (actions.length === 2 && actions[0] === 'delete' && actions[1] === 'create')
      return { border: '#3b82f6', bg: '#eff6ff', text: '#2563eb', label: '↻' };
    switch (actions[0]) {
      case 'create': return { border: '#10b981', bg: '#ecfdf5', text: '#059669', label: '+' };
      case 'update': return { border: '#f59e0b', bg: '#fffbeb', text: '#d97706', label: '~' };
      case 'delete': return { border: '#ef4444', bg: '#fef2f2', text: '#dc2626', label: '-' };
      default: return { border: '#9ca3af', bg: '#f9fafb', text: '#6b7280', label: '·' };
    }
  }

  function getNodeLabel(resource) {
    const parts = resource.type.split('_');
    if (parts.length <= 1) return resource.type;
    const rest = parts.slice(1).map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
    return rest;
  }

  function getNodeDetail(resource) {
    const d = resource?.detail;
    if (!d) return '';
    const parts = [];
    if (d.instance_type) parts.push(d.instance_type);
    if (d.cidr) parts.push(d.cidr);
    if (d.rule) parts.push(d.rule);
    if (d.ingress) parts.push('in: ' + d.ingress);
    if (d.egress && !d.ingress) parts.push('out: ' + d.egress);
    return parts.join(' | ').substring(0, 60);
  }

  async function layoutTopology(data) {
    try {
      const elk = new ELK();

      const NODE_W = 200;
      const hasDetail = data.resources.some(r => r.detail && Object.keys(r.detail).length > 0);
      const NODE_H = hasDetail ? 58 : 48;

      // Build address set for filtering edges
      const addrSet = new Set(data.resources.map(r => r.address));

      // Dynamic spacing based on node count
      const nodeCount = data.resources.length;
      const nodeSpacing = nodeCount > 8 ? '25' : '35';
      const layerSpacing = nodeCount > 8 ? '40' : '50';

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
        children: data.resources.map(r => ({
          id: r.address,
          width: NODE_W,
          height: NODE_H,
        })),
        edges: (data.edges || [])
          .filter(e => addrSet.has(e.from) && addrSet.has(e.to))
          .map((e, i) => ({
            id: `e${i}`,
            sources: [e.from],
            targets: [e.to],
          })),
      };

      const layout = await elk.layout(graph);

      // Build resource map for colors
      const resMap = {};
      data.resources.forEach(r => { resMap[r.address] = r; });

      const newNodes = (layout.children || []).map(n => ({
        id: n.id,
        x: n.x,
        y: n.y,
        w: n.width,
        h: n.height,
        resource: resMap[n.id],
        color: getActionColor(resMap[n.id]?.actions),
        label: getNodeLabel(resMap[n.id] || {}),
        detailText: getNodeDetail(resMap[n.id] || {}),
      }));

      const newEdges = (layout.edges || []).map(e => {
        const sections = e.sections || [];
        if (sections.length > 0) {
          const s = sections[0];
          return {
            id: e.id,
            startPoint: s.startPoint,
            endPoint: s.endPoint,
            bendPoints: s.bendPoints || [],
          };
        }
        // Fallback: center-to-center
        const src = newNodes.find(n => n.id === (e.sources?.[0]));
        const tgt = newNodes.find(n => n.id === (e.targets?.[0]));
        if (src && tgt) {
          return {
            id: e.id,
            startPoint: { x: src.x + src.w / 2, y: src.y + src.h },
            endPoint: { x: tgt.x + tgt.w / 2, y: tgt.y },
            bendPoints: [],
          };
        }
        return null;
      }).filter(Boolean);

      // Calculate viewBox
      const padding = 40;
      const maxX = Math.max(...newNodes.map(n => n.x + n.w), 400) + padding * 2;
      const maxY = Math.max(...newNodes.map(n => n.y + n.h), 200) + padding * 2;

      // Assign to state in one batch
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
      // Straight line with slight curve
      const midY = (startPoint.y + endPoint.y) / 2;
      return `M ${startPoint.x} ${startPoint.y} C ${startPoint.x} ${midY}, ${endPoint.x} ${midY}, ${endPoint.x} ${endPoint.y}`;
    }
    // Multiple bend points
    let d = `M ${startPoint.x} ${startPoint.y}`;
    const pts = [startPoint, ...bendPoints, endPoint];
    for (let i = 1; i < pts.length; i++) {
      const prev = pts[i - 1];
      const curr = pts[i];
      const midY = (prev.y + curr.y) / 2;
      d += ` C ${prev.x} ${midY}, ${curr.x} ${midY}, ${curr.x} ${curr.y}`;
    }
    return d;
  }

  function handleStartFromPreview() {
    const caseId = planPreviewModal.caseId;
    planPreviewModal = { show: false, caseName: '', caseId: '', loading: false, error: '', data: null };
    handleStart(caseId);
  }
  
  async function handleStop(caseId) {
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'stopping' } : c);
    try {
      await StopCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }
  
  function showStopConfirm(caseId, caseName) {
    stopConfirm = { show: true, caseId, caseName };
  }
  
  function cancelStop() {
    stopConfirm = { show: false, caseId: null, caseName: '' };
  }
  
  async function confirmStop() {
    const caseId = stopConfirm.caseId;
    stopConfirm = { show: false, caseId: null, caseName: '' };
    await handleStop(caseId);
  }
  
  function showDeleteConfirm(caseId, caseName) {
    deleteConfirm = { show: true, caseId, caseName };
  }
  
  function cancelDelete() {
    deleteConfirm = { show: false, caseId: null, caseName: '' };
  }
  
  async function confirmDelete() {
    const caseId = deleteConfirm.caseId;
    deleteConfirm = { show: false, caseId: null, caseName: '' };
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'removing' } : c);
    try {
      await RemoveCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }
  
  function getShortId(id) {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }
  
  function getStateConfig(state) {
    return stateConfig[state] || stateConfig['pending'];
  }

  function openContextMenu(e, c) {
    e.preventDefault();
    // Clamp position to keep menu within viewport
    const x = Math.min(e.clientX, window.innerWidth - 200);
    const y = Math.min(e.clientY, window.innerHeight - 300);
    contextMenu = { show: true, x, y, caseId: c.id, caseName: c.name, caseState: c.state, caseType: c.type };
  }

  function closeContextMenu() {
    contextMenu = { show: false, x: 0, y: 0, caseId: null, caseName: '', caseState: '', caseType: '' };
  }

  function ctxAction(fn) {
    fn();
    closeContextMenu();
  }
  
  async function toggleCaseExpand(caseId, state) {
    if (expandedCase === caseId) {
      expandedCase = null;
      return;
    }
    expandedCase = caseId;
    if (state === 'running' && !caseOutputs[caseId]) {
      try {
        const outputs = await GetCaseOutputs(caseId);
        if (outputs) {
          caseOutputs[caseId] = outputs;
          caseOutputs = caseOutputs;
        }
      } catch (e) {
        console.error('Failed to load outputs:', e);
      }
    }
  }
  
  async function copyToClipboard(value, key) {
    try {
      await navigator.clipboard.writeText(value);
      copiedKey = key;
      setTimeout(() => { copiedKey = null; }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  async function copyAllOutputs(outputs) {
    try {
      const text = Object.entries(outputs)
        .map(([key, value]) => `${key}=${value}`)
        .join('\n');
      await navigator.clipboard.writeText(text);
      copiedAllKey = expandedCase;
      setTimeout(() => { copiedAllKey = null; }, 2000);
    } catch (e) {
      console.error('Failed to copy all outputs:', e);
    }
  }

  // ============================================================================
  // Cost Estimation Functions
  // ============================================================================

  /**
   * Load base cost estimate for a template using default variable values
   * This is used for the template list preview
   * Failures are handled silently (no error messages shown to user)
   */
  async function loadTemplateCost(templateName) {
    if (!templateName || templateCostsLoading.has(templateName)) {
      return;
    }
    
    // Mark as loading
    templateCostsLoading.add(templateName);
    templateCostsLoading = templateCostsLoading;
    
    try {
      // Get template variables to extract defaults
      const vars = await GetTemplateVariables(templateName);
      
      // Build variables object with default values only
      /** @type {Record<string, string>} */
      const defaultVars = {};
      if (vars && vars.length > 0) {
        for (const v of vars) {
          if (v.defaultValue && v.defaultValue !== '') {
            defaultVars[v.name] = String(v.defaultValue);
          }
        }
      }
      
      // Call GetCostEstimate with default variables
      const estimate = await GetCostEstimate(templateName, defaultVars);
      
      // Store the estimate
      templateCosts[templateName] = estimate;
      templateCosts = templateCosts; // Trigger reactivity
    } catch (e) {
      // Silent failure - don't show error to user
      // Just don't add the cost to templateCosts
      console.debug(`Failed to load cost for template ${templateName}:`, e);
    } finally {
      // Remove from loading set
      templateCostsLoading.delete(templateName);
      templateCostsLoading = templateCostsLoading;
    }
  }

  /**
   * Load base cost estimates for all templates
   * Called manually by user clicking the "Load All Template Costs" button
   */
  async function loadAllTemplateCosts() {
    if (!templates || templates.length === 0) {
      return;
    }
    
    allTemplateCostsLoading = true;
    
    try {
      // Load costs for all templates in parallel
      // Each loadTemplateCost handles its own errors silently
      await Promise.all(templates.map(tmpl => loadTemplateCost(tmpl.name)));
    } finally {
      allTemplateCostsLoading = false;
    }
  }

  async function loadCostEstimate() {
    if (!selectedTemplate) return;
    
    // Set loading state and clear previous errors
    costEstimateLoading = true;
    costEstimateError = '';
    
    try {
      // Prepare variables object with non-empty values
      /** @type {Record<string, string>} */
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = String(value);
        }
      }
      
      // Call GetCostEstimate API
      costEstimate = await GetCostEstimate(selectedTemplate, vars);
      
      // Show modal on success
      showCostEstimate = true;
    } catch (e) {
      // Set user-friendly error message
      costEstimateError = e.message || String(e);
    } finally {
      // Clear loading state
      costEstimateLoading = false;
    }
  }

  /**
   * Debounced cost estimation function
   * Waits 500ms after the last variable change before triggering cost estimation
   * Only triggers if the cost estimate modal is currently shown
   */
  function debouncedCostEstimate() {
    // Clear any existing timer
    if (costEstimateDebounceTimer) {
      clearTimeout(costEstimateDebounceTimer);
    }
    
    // Set new timer for 500ms delay
    costEstimateDebounceTimer = setTimeout(() => {
      // Only trigger if cost estimate modal is currently shown
      if (showCostEstimate) {
        loadCostEstimate();
      }
    }, 500);
  }

  // Watch for variable changes and trigger debounced cost estimation
  // This reactive statement runs whenever variableValues changes
  $effect(() => {
	if (selectedTemplate && Object.keys(variableValues).length > 0) {
    debouncedCostEstimate();
  }
});

  // ============================================================================
  // Batch Operation Functions
  // ============================================================================

  function toggleSelectAll() {
    if (allSelected) {
      const pageIds = new Set(paginatedCases.map(c => c.id));
      selectedCases = new Set([...selectedCases].filter(id => !pageIds.has(id)));
    } else {
      selectedCases = new Set([...selectedCases, ...paginatedCases.map(c => c.id)]);
    }
  }

  function toggleSelectCase(caseId) {
    const newSet = new Set(selectedCases);
    if (newSet.has(caseId)) {
      newSet.delete(caseId);
    } else {
      newSet.add(caseId);
    }
    selectedCases = newSet;
  }

  function showBatchDeleteConfirm() {
    batchDeleteConfirm = { show: true, count: selectedCases.size };
  }

  function cancelBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
  }

  async function confirmBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
    batchOperating = true;
    
    const caseIds = Array.from(selectedCases);
    
    try {
      // Execute deletions in parallel
      await Promise.all(caseIds.map(caseId => RemoveCase(caseId)));
      selectedCases = new Set();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }

  function showBatchStopConfirm() {
    batchStopConfirm = { show: true, count: selectedCases.size };
  }

  function cancelBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
  }

  async function confirmBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
    batchOperating = true;
    
    const caseIds = Array.from(selectedCases);
    
    try {
      // Execute stops in parallel
      await Promise.all(caseIds.map(caseId => StopCase(caseId)));
      selectedCases = new Set();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }

  async function handleBatchStart() {
    batchOperating = true;

    const caseIds = Array.from(selectedCases);
    const selectedCaseList = caseIds.map(id => cases.find(c => c.id === id)).filter(Boolean);

    // Check plugin dependencies for all selected cases
    const casesWithPlugins = selectedCaseList.filter(c => c.plugins && c.plugins.trim());
    let missingByCase = [];
    if (casesWithPlugins.length > 0) {
      try {
        const installed = await ListPlugins();
        const installedNames = new Set(installed.map(p => p.manifest?.name || p.name));
        for (const c of casesWithPlugins) {
          const required = c.plugins.split(',').map(s => s.trim()).filter(Boolean);
          const missing = required.filter(name => !installedNames.has(name));
          if (missing.length > 0) {
            missingByCase.push({ name: c.name, caseId: c.id, missing });
          }
        }
      } catch {
        // Ignore plugin check errors, proceed with start
      }
    }

    if (missingByCase.length > 0) {
      pluginCheckModal = {
        show: true, mode: 'batch',
        caseName: '', caseId: '',
        caseIds,
        missingPlugins: [],
        missingByCase,
        okCount: caseIds.length - missingByCase.length
      };
      batchOperating = false;
      return;
    }

    try {
      await Promise.all(caseIds.map(caseId => doStart(caseId)));
      selectedCases = new Set();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }

  // Tag colors
  const tagColors = [
    'bg-blue-100 text-blue-700', 'bg-purple-100 text-purple-700', 'bg-pink-100 text-pink-700',
    'bg-indigo-100 text-indigo-700', 'bg-teal-100 text-teal-700', 'bg-orange-100 text-orange-700',
    'bg-cyan-100 text-cyan-700', 'bg-rose-100 text-rose-700', 'bg-lime-100 text-lime-700',
    'bg-amber-100 text-amber-700',
  ];
  function getTagColor(tag) {
    let hash = 0;
    for (let i = 0; i < tag.length; i++) hash = ((hash << 5) - hash + tag.charCodeAt(i)) | 0;
    return tagColors[Math.abs(hash) % tagColors.length];
  }

  async function addTagToCase(caseId, tag) {
    if (!tag.trim()) return;
    const c = cases.find(x => x.id === caseId);
    const tags = [...(c?.tags || [])];
    if (tags.includes(tag.trim())) return;
    tags.push(tag.trim());
    await SetCaseTags(caseId, tags);
    await refresh();
    tagInput = '';
  }

  async function removeTagFromCase(caseId, tag) {
    const c = cases.find(x => x.id === caseId);
    const tags = (c?.tags || []).filter(t => t !== tag);
    await SetCaseTags(caseId, tags);
    await refresh();
  }


</script>

<div class="space-y-5">
  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''} aria-label={t.closeError}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Quick Create -->
  {#if !templates || templates.length === 0}
    <div class="bg-blue-50 border border-blue-100 rounded-xl p-5 mb-4">
      <div class="flex items-start gap-3">
        <svg class="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
        </svg>
        <div class="flex-1">
          <p class="text-[13px] text-blue-700">{t.noTemplatesHint}</p>
          <button 
            class="mt-3 h-8 px-4 bg-blue-500 text-white text-[12px] font-medium rounded-lg hover:bg-blue-600 transition-colors cursor-pointer"
            onclick={() => onTabChange && onTabChange('registry')}
          >
            {t.noTemplatesHintButton}
          </button>
        </div>
      </div>
    </div>
  {/if}
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-end gap-4 mb-4">
      <div class="flex-1 relative">
        <label for="templateSelect" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.template}</label>
        <button
          type="button"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-left flex items-center justify-between focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow cursor-pointer {selectedTemplate ? 'text-gray-900' : 'text-gray-400'}"
          onclick={() => { templateDropdownOpen = !templateDropdownOpen; if (!templateDropdownOpen) templateSearch = ''; }}
        >
          <span class="truncate">{selectedTemplate || t.selectTemplate}</span>
          <svg class="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
          </svg>
        </button>
        {#if templateDropdownOpen}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div class="fixed inset-0 z-40" onclick={() => { templateDropdownOpen = false; templateSearch = ''; }}></div>
          <div class="absolute top-full left-0 right-0 mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 max-h-[320px] flex flex-col">
            <div class="p-2 border-b border-gray-100">
              <input
                type="text"
                class="w-full h-8 px-2.5 text-[13px] bg-gray-50 border border-gray-200 rounded-md text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-gray-400"
                placeholder={t.searchTemplate || '搜索模板...'}
                bind:value={templateSearch}
                autofocus
                onkeydown={(e) => {
                  if (e.key === 'Escape') { templateDropdownOpen = false; templateSearch = ''; }
                }}
              />
            </div>
            <div class="overflow-y-auto flex-1">
              {#each filteredTemplates as tmpl}
                <button
                  type="button"
                  class="w-full px-3 py-2 text-[13px] text-left hover:bg-gray-50 flex items-center justify-between cursor-pointer {selectedTemplate === tmpl.name ? 'bg-gray-100 font-medium text-gray-900' : 'text-gray-700'}"
                  onclick={() => { selectedTemplate = tmpl.name; templateDropdownOpen = false; templateSearch = ''; loadTemplateVariables(tmpl.name); }}
                >
                  <span class="truncate">{tmpl.name}</span>
                  {#if templateCosts[tmpl.name]}
                    <span class="text-[11px] text-gray-400 ml-2 flex-shrink-0">{templateCosts[tmpl.name].currency} {templateCosts[tmpl.name].total_monthly_cost.toFixed(2)}/mo</span>
                  {/if}
                </button>
              {:else}
                <div class="px-3 py-4 text-[12px] text-gray-400 text-center">{t.noResults || '无匹配结果'}</div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
      <div class="w-48">
        <label for="caseName" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.name}</label>
        <input 
          id="caseName"
          type="text" 
          placeholder={t.optional}
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={newCaseName} 
        />
      </div>
      <button 
        class="h-10 px-5 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[13px] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        onclick={handleCreate}
        disabled={createBusy}
      >
        {t.create}
      </button>
      <button 
        class="h-10 px-5 bg-emerald-500 text-white text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        onclick={handleCreateAndRun}
        disabled={createBusy}
      >
        {t.createAndRun}
      </button>
      {#if selectedTemplate}
        <button 
          class="h-10 px-5 bg-blue-600 text-white text-[13px] font-medium rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          onclick={loadCostEstimate}
          disabled={costEstimateLoading}
        >
          {#if costEstimateLoading}
            <span class="flex items-center gap-2">
              <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              {t.calculating}
            </span>
          {:else}
            {t.costEstimate}
          {/if}
        </button>
      {/if}
      <button 
        class="h-10 px-5 text-red-500 border border-red-500 bg-white hover:bg-red-50 text-[13px] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        onclick={loadAllTemplateCosts}
        disabled={allTemplateCostsLoading || templates.length === 0}
      >
        {#if allTemplateCostsLoading}
          <span class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            {t.loadingAllTemplateCosts}
          </span>
        {:else}
          {t.batchEstimate}
        {/if}
      </button>
    </div>

    <!-- Cost Estimation Error Display -->
    {#if costEstimateError}
      <div class="flex items-center gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
        <svg class="w-4 h-4 text-amber-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        <div class="flex-1">
          <div class="text-[13px] text-amber-800 font-medium">{t.costEstimateError}</div>
          <div class="text-[12px] text-amber-700 mt-0.5">{costEstimateError}</div>
          <div class="text-[11px] text-amber-600 mt-1">{t.costEstimateErrorHint}</div>
        </div>
        <button class="text-amber-400 hover:text-amber-600 cursor-pointer" onclick={() => costEstimateError = ''} aria-label={t.closeTip}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/if}

    <!-- Show persistent error or current create status -->
    {#if createStatus !== 'idle' || getPersistentError()}
      {@const hasError = createStatus === 'error' || !!getPersistentError()}
      <div class="mt-3 rounded-lg border {hasError ? 'border-red-200 bg-red-50' : createStatus === 'success' ? 'border-emerald-200 bg-emerald-50' : 'border-gray-100 bg-gray-50'} px-3 py-2 text-[12px] relative">
        {#if createStatus === 'creating' || createStatus === 'initializing'}
          <div class="flex items-center gap-2">
            <div class="w-3.5 h-3.5 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
            <span class="text-gray-700">{createStatusMessage}</span>
          </div>
        {:else if createStatus === 'success'}
          <span class="text-emerald-600">{createStatusMessage}</span>
        {:else if hasError}
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div class="flex-1">
              <h4 class="text-[13px] font-semibold text-red-900">{t.sceneCreationFailed || '场景创建失败'}</h4>
              <p class="text-[12px] text-red-700 mt-1">{t.checkConfigRetry || '请检查配置后重试。错误详情：'}</p>
              {#if (getPersistentError()?.detail || createStatusDetail)}
                <pre class="mt-2 p-3 bg-white rounded border border-red-200 text-[11px] text-red-800 overflow-x-auto whitespace-pre-wrap max-h-48">{getPersistentError()?.detail || createStatusDetail}</pre>
              {/if}
              <!-- AI Analysis - 跳转到 AI 对话 -->
              <button 
                class="mt-3 px-3 py-1.5 text-[12px] font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded-md hover:bg-blue-100 transition-colors flex items-center gap-1.5 cursor-pointer"
                onclick={handleAIAnalysis}
              >
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
                {t.aiAnalyzeError || 'AI 分析错误原因'}
              </button>
            </div>
            <!-- 关闭按钮 -->
            <button 
              class="text-red-400 hover:text-red-600 ml-2 flex-shrink-0 cursor-pointer"
              onclick={dismissPersistentError}
              title={t.close}
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        {/if}
        {#if createStatus !== 'error' && createStatusDetail}
          <span class="text-gray-400 truncate">{createStatusDetail}</span>
        {/if}
      </div>
    {:else if costEstimateError}
      <!-- Spacer to maintain layout when cost estimate error is shown but no create status -->
      <div class="mt-3"></div>
    {/if}

    {#if terraformInitHint.show}
      <div class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[12px] text-amber-700 relative">
        <button
          class="absolute right-2 top-2 text-amber-400 hover:text-amber-600 cursor-pointer"
          onclick={dismissTerraformInitHint}
          aria-label={t.close}
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
        <div class="flex items-start gap-2">
          <svg class="w-4 h-4 mt-0.5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3m0 4h.01M10.29 3.86l-7.4 12.8A2 2 0 004.61 19h14.78a2 2 0 001.72-2.34l-7.4-12.8a2 2 0 00-3.42 0z" />
          </svg>
          <div class="flex-1">
            <div class="font-medium">{t.mirrorDetected}</div>
            <div class="text-amber-600 mt-1">{t.mirrorDetectedDesc}</div>
            {#if terraformInitHint.detail}
              <div class="text-amber-500 mt-1 truncate">{terraformInitHint.detail}</div>
            {/if}
            <div class="mt-2 text-amber-700">
              <div class="font-medium">{t.mirrorFixTitle}</div>
              <ul class="mt-1 list-disc list-inside text-amber-600 space-y-0.5">
                <li>{t.mirrorFixStep1}</li>
                <li>{t.mirrorFixStep2}</li>
                <li>{t.mirrorFixStep3}</li>
              </ul>
            </div>
            <div class="mt-2 flex flex-wrap gap-2">
              <button
                class="h-8 px-3 bg-white text-amber-700 text-[12px] font-medium rounded-md border border-amber-200 hover:bg-amber-100 transition-colors"
                onclick={() => onTabChange('settings')}
              >{t.mirrorGoSettings}</button>
            </div>
          </div>
        </div>
      </div>
    {:else if costEstimateError}
      <!-- Spacer to maintain layout when cost estimate error is shown but no terraform hint -->
      <div class="mt-3"></div>
    {/if}
    
    <!-- Template Variables -->
    {#if templateVariables.length > 0}
      <div class="border-t border-gray-100 pt-4 mt-4">
        <div class="text-[12px] font-medium text-gray-500 mb-3">{t.templateParams}</div>
        <div class="grid grid-cols-2 gap-3">
          {#each templateVariables as variable}
            <div class="flex flex-col">
              <label for="var-{variable.name}" class="text-[11px] text-gray-500 mb-1">
                {variable.name}
                {#if variable.required}
                  <span class="text-red-500">*</span>
                {/if}
                {#if variable.description}
                  <span class="text-gray-500 ml-1">({variable.description})</span>
                {/if}
                {#if variable.type && variable.type !== 'string'}
                  <span class="text-gray-300 ml-1 text-[10px]">{variable.type}</span>
                {/if}
              </label>
              {#if variable.type === 'bool'}
                <button
                  type="button"
                  class="h-9 flex items-center gap-2 px-3 rounded-lg bg-gray-50 cursor-pointer"
                  onclick={() => variableValues[variable.name] = variableValues[variable.name] === 'true' ? 'false' : 'true'}
                >
                  <div class="relative w-8 h-[18px] rounded-full transition-colors {variableValues[variable.name] === 'true' ? 'bg-gray-900' : 'bg-gray-300'}">
                    <div class="absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white shadow transition-transform {variableValues[variable.name] === 'true' ? 'translate-x-[16px]' : 'translate-x-[2px]'}"></div>
                  </div>
                  <span class="text-[12px] text-gray-600">{variableValues[variable.name] === 'true' ? 'true' : 'false'}</span>
                </button>
              {:else if variable.type === 'number'}
                <input
                  id="var-{variable.name}"
                  type="number"
                  class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  placeholder={variable.defaultValue || '0'}
                  bind:value={variableValues[variable.name]}
                />
              {:else if variable.name === 'region' && variable.validation?.allowed_values?.length > 0}
                <select
                  id="var-{variable.name}"
                  class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow appearance-none cursor-pointer"
                  bind:value={variableValues[variable.name]}
                >
                  {#each variable.validation.allowed_values as val}
                    <option value={val}>{val}</option>
                  {/each}
                </select>
              {:else}
                <input
                  id="var-{variable.name}"
                  type={variable.sensitive ? 'password' : 'text'}
                  class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  placeholder={variable.defaultValue || ''}
                  bind:value={variableValues[variable.name]}
                />
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  <!-- Scheduled Tasks Manager moved to TaskCenter page -->

  <!-- Table -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <!-- Search + Status Tabs -->
    <div class="px-5 py-3 border-b border-gray-100">
      <div class="flex items-center gap-3">
        <!-- Search -->
        <div class="relative flex-1 max-w-xs">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
          </svg>
          <input
            type="text"
            placeholder={t.searchCases || '搜索场景...'}
            class="w-full h-8 pl-9 pr-3 text-[12px] bg-gray-50 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-shadow"
            bind:value={searchQuery}
          />
          {#if searchQuery}
            <button class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 cursor-pointer" onclick={() => searchQuery = ''}>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          {/if}
        </div>
        <!-- Status Tabs -->
        <div class="flex items-center gap-1 bg-gray-50 rounded-lg p-0.5">
          {#each [
            { key: 'all', label: t.tagFilterAll || '全部' },
            { key: 'running', label: t.running || '运行中' },
            { key: 'stopped', label: t.stopped || '已停止' },
            { key: 'error', label: t.error || '错误' },
          ] as tab}
            <button
              class="px-2.5 py-1 text-[11px] font-medium rounded-md transition-colors cursor-pointer
                {statusFilter === tab.key ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
              onclick={() => { statusFilter = tab.key; selectedCases = new Set(); }}
            >
              {tab.label}
              <span class="ml-1 text-[10px] {statusFilter === tab.key ? 'text-gray-500' : 'text-gray-400'}">{statusCounts[tab.key]}</span>
            </button>
          {/each}
        </div>
      </div>
    </div>
    <!-- Tag Filter Bar -->
    {#if allTagNames.length > 0}
      <div class="px-5 py-2.5 border-b border-gray-100 flex items-center gap-2 flex-wrap">
        <span class="text-[11px] text-gray-400 mr-1">{t.tags || '标签'}:</span>
        <button
          class="px-2 py-0.5 text-[11px] rounded-full transition-colors {selectedTag === '' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
          onclick={() => { selectedTag = ''; selectedCases = new Set(); }}
        >{t.tagFilterAll || '全部'}</button>
        {#each allTagNames as tag}
          <button
            class="px-2 py-0.5 text-[11px] rounded-full transition-colors {selectedTag === tag ? 'bg-gray-900 text-white' : getTagColor(tag)}"
            onclick={() => { selectedTag = selectedTag === tag ? '' : tag; selectedCases = new Set(); }}
          >{tag}</button>
        {/each}
      </div>
    {/if}
    <!-- Batch Operations Bar -->
    {#if hasSelection}
      <div class="px-5 py-3 bg-blue-50 border-b border-blue-100 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="text-[13px] font-medium text-blue-900">
            {t.selected} {selectedCases.size} {t.items}
          </span>
          <button
            class="text-[12px] text-blue-600 hover:text-blue-800 underline"
            onclick={() => { selectedCases = new Set(); }}
          >
            {t.clearSelection}
          </button>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors disabled:opacity-50"
            onclick={handleBatchStart}
            disabled={batchOperating}
          >
            {t.batchStart}
          </button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors disabled:opacity-50"
            onclick={showBatchStopConfirm}
            disabled={batchOperating}
          >
            {t.batchStop}
          </button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors disabled:opacity-50"
            onclick={showBatchDeleteConfirm}
            disabled={batchOperating}
          >
            {t.batchDelete}
          </button>
        </div>
      </div>
    {/if}
    
    <div class="overflow-x-auto">
    <table class="w-full">
      <thead>
        <tr class="border-b border-gray-100">
          <th class="text-left pl-4 pr-1 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-6">
            <input
              type="checkbox"
              class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
              checked={allSelected}
              indeterminate={someSelected}
              onchange={toggleSelectAll}
            />
          </th>
          <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide whitespace-nowrap">{t.id}</th>
          <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.name}</th>
          <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide whitespace-nowrap">{t.type}</th>
          <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide whitespace-nowrap">{t.state}</th>
          <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide whitespace-nowrap">{t.time}</th>
          <th class="text-right px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide whitespace-nowrap">{t.actions}</th>
        </tr>
      </thead>
      <tbody>
        {#each paginatedCases || [] as c, i}
          <tr 
            class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer"
            onclick={() => toggleCaseExpand(c.id, c.state)}
            oncontextmenu={(e) => openContextMenu(e, c)}
          >
            <td class="pl-4 pr-1 py-3.5" onclick={(e) => e.stopPropagation()}>
              <input
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
                checked={selectedCases.has(c.id)}
                onchange={() => toggleSelectCase(c.id)}
              />
            </td>
            <td class="px-3 py-3.5 whitespace-nowrap">
              <div class="flex items-center gap-2">
                <svg class="w-4 h-4 text-gray-400 transition-transform {expandedCase === c.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
                <code class="text-[12px] text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">{getShortId(c.id)}</code>
              </div>
            </td>
            <td class="px-3 py-3.5">
              <div class="flex items-center gap-1.5 flex-wrap">
                <span class="text-[13px] font-medium text-gray-900 truncate max-w-[200px]" title={c.name}>{c.name}</span>
                {#each c.tags || [] as tag}
                  <span class="px-1.5 py-0 text-[10px] rounded-full whitespace-nowrap {getTagColor(tag)}">{tag}</span>
                {/each}
              </div>
            </td>
            <td class="px-3 py-3.5 whitespace-nowrap">
              <span class="text-[13px] text-gray-600">{c.type}</span>
            </td>
            <td class="px-3 py-3.5 whitespace-nowrap">
              <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {(stateConfig[c.state] || stateConfig['pending']).color}">
                <span class="w-1.5 h-1.5 rounded-full {(stateConfig[c.state] || stateConfig['pending']).dot}"></span>
                {(stateConfig[c.state] || stateConfig['pending']).label}
              </span>
              {#if c.isSpotInstance}
                <span class="ml-1.5 px-1.5 py-0.5 text-[10px] font-medium rounded {c.state === 'terminated' ? 'text-red-700 bg-red-50 border border-red-300 animate-pulse' : 'text-amber-700 bg-amber-50 border border-amber-200'}">
                  {c.state === 'terminated' ? (t.spotTerminated || '已回收') : (t.spotInstance || '抢占式')}
                </span>
              {/if}
            </td>
            <td class="px-3 py-3.5 whitespace-nowrap">
              <span class="text-[12px] text-gray-500" title={c.stateTime}>{formatCompactTime(c.stateTime)}</span>
              {#if c.state === 'running' && c.stateTime}
                <span class="ml-1.5 text-[11px] text-emerald-600 font-medium" title={t.runningTime || '运行时间'}>⏱ {formatElapsed(c.stateTime)}</span>
              {/if}
            </td>
            <td class="px-3 py-3.5 text-right whitespace-nowrap" onclick={(e) => e.stopPropagation()}>
              <div class="inline-flex items-center gap-1">
                {#if c.state !== 'starting' && c.state !== 'stopping' && c.state !== 'removing'}
                  <!-- Tag edit button -->
                  <button
                    class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                    onclick={() => { tagEditCase = tagEditCase === c.id ? null : c.id; tagInput = ''; }}
                    title={t.tags || '标签'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
                    </svg>
                  </button>
                  <!-- Clone button -->
                  <button
                    class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                    onclick={() => showCloneDialog(c.id, c.name)}
                    title={t.cloneCase || '克隆场景'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" />
                    </svg>
                  </button>
                {/if}
                {#if c.state === 'starting' || c.state === 'stopping' || c.state === 'removing'}
                  <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                    {stateConfig[c.state]?.label || t.processing}...
                  </span>
                {:else if c.state !== 'running'}
                  <!-- 预览按钮 -->
                  {#if c.state === 'created' || c.state === 'stopped'}
                    <button 
                      class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                      onclick={() => handlePlanPreview(c.id, c.name)}
                      title={t.planPreviewBtn || '预览'}
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                    </button>
                  {/if}
                  <!-- 定时启动按钮 -->
                  <button 
                    class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                    onclick={() => scheduleDialog = { show: true, caseId: c.id, caseName: c.name, action: 'start' }}
                    title={t.scheduleStart || '定时启动'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </button>
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                    onclick={() => handleStart(c.id)}
                  >{t.start}</button>
                {:else}
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                    onclick={() => sshModal = { show: true, caseId: c.id, caseName: c.name }}
                    title={t.sshOperations || 'SSH 运维'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
                    </svg>
                  </button>
                  <!-- 定时停止按钮 -->
                  <button 
                    class="p-1.5 text-gray-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                    onclick={() => scheduleDialog = { show: true, caseId: c.id, caseName: c.name, action: 'stop' }}
                    title={t.scheduleStop || '定时停止'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </button>
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                    onclick={() => showStopConfirm(c.id, c.name)}
                  >{t.stop}</button>
                {/if}
                {#if c.state !== 'starting' && c.state !== 'stopping' && c.state !== 'removing'}
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                    onclick={() => showDeleteConfirm(c.id, c.name)}
                  >{t.delete}</button>
                {/if}
              </div>
            </td>
          </tr>
          <!-- Tag edit row -->
          {#if tagEditCase === c.id}
            <tr class="bg-blue-50/50">
              <td colspan="7" class="px-5 py-2.5">
                <div class="flex items-center gap-2 flex-wrap pl-6">
                  <span class="text-[11px] text-gray-500">{t.tags || '标签'}:</span>
                  {#each c.tags || [] as tag}
                    <span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[11px] rounded-full {getTagColor(tag)}">
                      {tag}
                      <button class="ml-0.5 hover:opacity-70 cursor-pointer" onclick={() => removeTagFromCase(c.id, tag)}>×</button>
                    </span>
                  {/each}
                  <div class="inline-flex items-center gap-1">
                    <input
                      type="text"
                      class="w-24 text-[11px] px-2 py-0.5 border border-gray-200 rounded-full focus:outline-none focus:ring-1 focus:ring-blue-400"
                      placeholder={t.tagPlaceholder || '输入标签名'}
                      bind:value={tagInput}
                      onkeydown={(e) => { if (e.key === 'Enter') { addTagToCase(c.id, tagInput); } }}
                      list="tagSuggestions"
                    />
                    <datalist id="tagSuggestions">
                      {#each allTagNames.filter(t => !(c.tags || []).includes(t)) as suggestion}
                        <option value={suggestion} />
                      {/each}
                    </datalist>
                    <button
                      class="text-[11px] px-2 py-0.5 bg-blue-500 text-white rounded-full hover:bg-blue-600 disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
                      disabled={!tagInput.trim()}
                      onclick={() => addTagToCase(c.id, tagInput)}
                    >+</button>
                  </div>
                </div>
              </td>
            </tr>
          {/if}
          <!-- Expanded row for outputs -->
          {#if expandedCase === c.id}
            <tr class="bg-slate-50">
              <td colspan="7" class="px-5 py-4">
                <div class="pl-6">
                  {#if c.state === 'running'}
                    {#if caseOutputs[c.id]}
                      <div class="flex items-center justify-between mb-3">
                        <span class="text-[12px] font-medium text-gray-700">{t.outputInfo || '输出信息'}</span>
                        <button
                          class="px-2 py-1 text-[11px] font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-1"
                          onclick={() => copyAllOutputs(caseOutputs[c.id])}
                        >
                          {#if copiedAllKey === c.id}
                            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                            {t.copied || '已复制'}
                          {:else}
                            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                            </svg>
                            {t.copyAll || '复制全部'}
                          {/if}
                        </button>
                      </div>
                      <div class="grid grid-cols-2 gap-3">
                        {#each Object.entries(caseOutputs[c.id]) as [key, value]}
                          <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                            <div class="flex items-center justify-between mb-1">
                              <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                              <button 
                                class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                onclick={(e) => { e.stopPropagation(); copyToClipboard(value, key); }}
                                title={t.copy}
                              >
                                {#if copiedKey === key}
                                  <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                  </svg>
                                  <span class="text-[10px] text-emerald-500">{t.copied}</span>
                                {:else}
                                  <svg class="w-4 h-4 text-gray-400 hover:text-gray-600 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                  </svg>
                                {/if}
                              </button>
                            </div>
                            <div class="text-[13px] font-mono text-gray-900 break-all">{value}</div>
                          </div>
                        {/each}
                      </div>
                    {:else}
                      <div class="text-[13px] text-gray-500">{t.loadingOutputs}</div>
                    {/if}
                  {:else}
                    <div class="text-[13px] text-gray-500">{t.noOutput}</div>
                  {/if}
                </div>
              </td>
            </tr>
          {/if}
        {:else}
          <tr>
            <td colspan="7" class="py-16">
              <div class="flex flex-col items-center text-gray-400">
                <svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                </svg>
                <p class="text-[13px]">{t.noScene}</p>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
    </div>
    {#if totalPages > 1}
      <div class="px-5 py-3 border-t border-gray-100 flex items-center justify-between">
        <span class="text-[11px] text-gray-400">
          {t.showingResults || '显示'} {(currentPage - 1) * pageSize + 1}-{Math.min(currentPage * pageSize, filteredCases.length)} / {filteredCases.length}
        </span>
        <div class="flex items-center gap-1">
          <button
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            onclick={() => currentPage = 1}
            disabled={currentPage === 1}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M18.75 19.5l-7.5-7.5 7.5-7.5m-6 15L5.25 12l7.5-7.5" /></svg>
          </button>
          <button
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            onclick={() => currentPage = Math.max(1, currentPage - 1)}
            disabled={currentPage === 1}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" /></svg>
          </button>
          <span class="px-3 text-[12px] font-medium text-gray-700">{currentPage} / {totalPages}</span>
          <button
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            onclick={() => currentPage = Math.min(totalPages, currentPage + 1)}
            disabled={currentPage === totalPages}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
          </button>
          <button
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
            onclick={() => currentPage = totalPages}
            disabled={currentPage === totalPages}
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 4.5l7.5 7.5-7.5 7.5m6-15l7.5 7.5-7.5 7.5" /></svg>
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>

<!-- Delete Confirmation Modal -->
<Modal show={deleteConfirm.show} onclose={cancelDelete}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
          <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmDelete}</h3>
          <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
        </div>
      </div>
      <p class="text-[13px] text-gray-600">
        {t.confirmDeleteScene} <span class="font-medium text-gray-900">"{deleteConfirm.caseName}"</span>?
      </p>
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button 
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelDelete}
      >{t.cancel}</button>
      <button 
        class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
        onclick={confirmDelete}
      >{t.delete}</button>
    </div>
  </div>
</Modal>

<!-- Batch Delete Confirmation Modal -->
<Modal show={batchDeleteConfirm.show} onclose={cancelBatchDelete}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
          <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchDelete}</h3>
          <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
        </div>
      </div>
      <p class="text-[13px] text-gray-600">
        {t.confirmBatchDeleteMessage} <span class="font-medium text-gray-900">{batchDeleteConfirm.count}</span> {t.scenes}?
      </p>
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button 
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelBatchDelete}
      >{t.cancel}</button>
      <button 
        class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
        onclick={confirmBatchDelete}
      >{t.delete}</button>
    </div>
  </div>
</Modal>

<!-- Batch Stop Confirmation Modal -->
<Modal show={batchStopConfirm.show} onclose={cancelBatchStop}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
          <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchStop}</h3>
          <p class="text-[13px] text-gray-500">{t.stopWarning}</p>
        </div>
      </div>
      <p class="text-[13px] text-gray-600">
        {t.confirmBatchStopMessage} <span class="font-medium text-gray-900">{batchStopConfirm.count}</span> {t.scenes}?
      </p>
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button 
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelBatchStop}
      >{t.cancel}</button>
      <button 
        class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
        onclick={confirmBatchStop}
      >{t.stop}</button>
    </div>
  </div>
</Modal>

<!-- Stop Confirmation Modal -->
<Modal show={stopConfirm.show} onclose={cancelStop}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
          <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmStop}</h3>
          <p class="text-[13px] text-gray-500">{t.stopWarning}</p>
        </div>
      </div>
      <p class="text-[13px] text-gray-600">
        {t.confirmStopScene} <span class="font-medium text-gray-900">"{stopConfirm.caseName}"</span>?
      </p>
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button 
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelStop}
      >{t.cancel}</button>
      <button 
        class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
        onclick={confirmStop}
      >{t.stop}</button>
    </div>
  </div>
</Modal>

<!-- Plugin Dependency Check Modal -->
<Modal show={pluginCheckModal.show} onclose={cancelPluginCheck}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-md w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-3">
        <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
          <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">
            {pluginCheckModal.mode === 'single' ? (t.pluginNotInstalled || '插件未安装') : (t.pluginsMissing || '部分场景缺少插件')}
          </h3>
        </div>
      </div>

      {#if pluginCheckModal.mode === 'single'}
        <p class="text-[13px] text-gray-600 mb-3">
          {t.sceneDepends || '场景'} <span class="font-medium text-gray-900">"{pluginCheckModal.caseName}"</span> {t.dependsPluginsNotInstalled || '依赖以下插件，但尚未安装：'}
        </p>
        <ul class="mb-3 space-y-1">
          {#each pluginCheckModal.missingPlugins as plugin}
            <li class="flex items-center gap-2 text-[12px] text-gray-700">
              <span class="w-1.5 h-1.5 rounded-full bg-red-400 flex-shrink-0"></span>
              {plugin}
            </li>
          {/each}
        </ul>
        <p class="text-[12px] text-gray-500">{t.startWithoutPlugins || '启动后部分功能（如生成配置文件）可能不可用。'}</p>
      {:else}
        <p class="text-[13px] text-gray-600 mb-3">{t.followingScenesPluginsMissing || '以下场景依赖的插件尚未安装：'}</p>
        <div class="mb-3 space-y-2 max-h-48 overflow-y-auto">
          {#each pluginCheckModal.missingByCase as item}
            <div class="px-3 py-2 bg-gray-50 rounded-lg">
              <p class="text-[12px] font-medium text-gray-900">{item.name}</p>
              <p class="text-[11px] text-gray-500 mt-0.5">
                {t.missing || '缺少'}: {item.missing.join(', ')}
              </p>
            </div>
          {/each}
        </div>
        {#if pluginCheckModal.okCount > 0}
          <p class="text-[12px] text-gray-500">{t.otherScenesOk || '其余'} {pluginCheckModal.okCount} {t.scenesPluginsComplete || '个场景插件完整，不受影响。'}</p>
        {/if}
      {/if}
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelPluginCheck}
      >{t.cancel || '取消'}</button>
      <button
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={goToPluginManager}
      >{t.goInstall || '去安装'}</button>
      <button
        class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
        onclick={confirmPluginStart}
      >{pluginCheckModal.mode === 'single' ? (t.startAnyway || '仍然启动') : (t.startAll || '全部启动')}</button>
    </div>
  </div>
</Modal>

<!-- Cost Estimate Modal -->
<Modal show={showCostEstimate && costEstimate} onclose={() => showCostEstimate = false}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-2xl w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <!-- Header -->
      <div class="px-6 py-5 border-b border-gray-100">
        <h3 class="text-[17px] font-semibold text-gray-900">{t.costEstimate}</h3>
        <p class="text-[13px] text-gray-500 mt-1">{costEstimate.disclaimer}</p>
      </div>
      
      <!-- Content -->
      <div class="px-6 py-5">
        <!-- Total Cost Summary -->
        <div class="grid grid-cols-2 gap-4 mb-6">
          <div class="bg-blue-50 rounded-lg p-4">
            <div class="text-[12px] text-blue-600 font-medium">{t.estimatedHourlyCost}</div>
            <div class="text-[24px] font-bold text-blue-900 mt-1">
              {costEstimate.currency} {costEstimate.total_hourly_cost.toFixed(4)}
            </div>
          </div>
          <div class="bg-emerald-50 rounded-lg p-4">
            <div class="text-[12px] text-emerald-600 font-medium">{t.estimatedMonthlyCost}</div>
            <div class="text-[24px] font-bold text-emerald-900 mt-1">
              {costEstimate.currency} {costEstimate.total_monthly_cost.toFixed(2)}
            </div>
          </div>
        </div>
        
        <!-- Cost Breakdown -->
        <div class="text-[13px] font-medium text-gray-700 mb-3">{t.costBreakdown}</div>
        <div class="space-y-2 max-h-64 overflow-y-auto">
          {#each costEstimate.breakdown as item}
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div class="flex-1">
                <div class="text-[13px] font-medium text-gray-900">{item.resource_name}</div>
                <div class="text-[11px] text-gray-500">{item.resource_type} × {item.count}</div>
              </div>
              <div class="text-right">
                {#if item.available}
                  <div class="text-[13px] font-medium text-gray-900">
                    {item.currency} {item.total_monthly.toFixed(2)}/mo
                  </div>
                  <div class="text-[11px] text-gray-500">
                    {item.currency} {item.total_hourly.toFixed(4)}/hr
                  </div>
                {:else}
                  <div class="text-[12px] text-amber-600">{t.pricingUnavailable}</div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
        
        <!-- Warnings -->
        {#if costEstimate.warnings && costEstimate.warnings.length > 0}
          <div class="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
            <div class="text-[12px] font-medium text-amber-800 mb-1">{t.warnings}</div>
            <ul class="text-[11px] text-amber-700 space-y-1">
              {#each costEstimate.warnings as warning}
                <li>• {warning}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
      
      <!-- Footer -->
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={() => showCostEstimate = false}
        >{t.close}</button>
      </div>
    </div>
</Modal>

<!-- SSH Operations Modal -->
{#if sshModal.show}
  <SSHModal 
    {t}
    caseId={sshModal.caseId}
    caseName={sshModal.caseName}
    onClose={() => sshModal = { show: false, caseId: null, caseName: '' }}
  />
{/if}

<!-- Schedule Dialog -->
{#if scheduleDialog.show}
  <ScheduleDialog
    {t}
    caseId={scheduleDialog.caseId}
    caseName={scheduleDialog.caseName}
    action={scheduleDialog.action}
    onClose={() => scheduleDialog = { show: false, caseId: null, caseName: '', action: '' }}
    onSuccess={() => {
      refresh();
    }}
  />
{/if}

<!-- Plan Preview Topology Modal -->
<Modal show={planPreviewModal.show} onclose={() => planPreviewModal = { ...planPreviewModal, show: false }}>
  <div class="bg-white rounded-xl border border-gray-100 shadow-2xl w-[92vw] max-w-[1100px] max-h-[90vh] flex flex-col" onclick={(e) => e.stopPropagation()}>
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.planPreview || '资源拓扑预览'}</h3>
          <span class="text-[13px] text-gray-500">— {planPreviewModal.caseName}</span>
          {#if planPreviewModal.data?.isSpotInstance}
            <span class="px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded">
              {t.spotInstance || '抢占式'}
            </span>
          {/if}
        </div>
        <button class="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => planPreviewModal = { ...planPreviewModal, show: false }}>
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-auto px-6 py-4">
        {#if planPreviewModal.loading}
          <div class="flex items-center justify-center py-16">
            <svg class="animate-spin h-6 w-6 text-blue-600 mr-3" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
            </svg>
            <span class="text-[13px] text-gray-500">{t.planLoading || '正在解析 Plan 文件...'}</span>
          </div>
        {:else if planPreviewModal.error}
          <div class="flex items-center justify-center py-16">
            <div class="text-center">
              <svg class="w-10 h-10 text-gray-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              <p class="text-[13px] text-gray-500">{planPreviewModal.error.includes('not found') ? (t.planNoPlanFile || 'Plan 文件不存在') : planPreviewModal.error}</p>
            </div>
          </div>
        {:else if planPreviewModal.data && !planPreviewModal.data.hasChanges}
          <div class="flex items-center justify-center py-16">
            <p class="text-[13px] text-gray-500">{t.planNoChanges || '没有资源变更'}</p>
          </div>
        {:else if planPreviewModal.data}
          <!-- Stats bar -->
          <div class="flex items-center gap-3 mb-5">
            {#if planPreviewModal.data.toCreate > 0}
              <div class="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-50 border border-emerald-200 rounded-lg">
                <span class="text-emerald-600 font-bold text-[13px]">+</span>
                <span class="text-[12px] font-medium text-emerald-700">{t.planToCreate || '创建'} {planPreviewModal.data.toCreate}</span>
              </div>
            {/if}
            {#if planPreviewModal.data.toUpdate > 0}
              <div class="flex items-center gap-1.5 px-3 py-1.5 bg-amber-50 border border-amber-200 rounded-lg">
                <span class="text-amber-600 font-bold text-[13px]">~</span>
                <span class="text-[12px] font-medium text-amber-700">{t.planToUpdate || '更新'} {planPreviewModal.data.toUpdate}</span>
              </div>
            {/if}
            {#if planPreviewModal.data.toDelete > 0}
              <div class="flex items-center gap-1.5 px-3 py-1.5 bg-red-50 border border-red-200 rounded-lg">
                <span class="text-red-600 font-bold text-[13px]">-</span>
                <span class="text-[12px] font-medium text-red-700">{t.planToDelete || '删除'} {planPreviewModal.data.toDelete}</span>
              </div>
            {/if}
            {#if planPreviewModal.data.toRecreate > 0}
              <div class="flex items-center gap-1.5 px-3 py-1.5 bg-blue-50 border border-blue-200 rounded-lg">
                <span class="text-blue-600 font-bold text-[13px]">↻</span>
                <span class="text-[12px] font-medium text-blue-700">{t.planToRecreate || '重建'} {planPreviewModal.data.toRecreate}</span>
              </div>
            {/if}
            <div class="ml-auto text-[12px] text-gray-400">
              {planPreviewModal.data.resources.length} {t.planResources || '个资源'}
              {#if elkEdges.length > 0}
                · {elkEdges.length} {t.planDependencies || '条依赖'}
              {:else if planPreviewModal.data.edges && planPreviewModal.data.edges.length > 0}
                · {planPreviewModal.data.edges.length} {t.planDependencies || '条依赖'}
              {/if}
            </div>
          </div>

          <!-- Type Summary Table -->
          {#if planPreviewModal.data.typeSummary && planPreviewModal.data.typeSummary.length > 0}
            <div class="mb-4 bg-white rounded-lg border border-gray-200 overflow-hidden">
              <table class="w-full text-[12px]">
                <thead>
                  <tr class="bg-gray-50 border-b border-gray-100">
                    <th class="text-left px-3 py-1.5 font-medium text-gray-500">{t.resourceType || '资源类型'}</th>
                    <th class="text-center px-3 py-1.5 font-medium text-gray-500 w-16">{t.count || '数量'}</th>
                    <th class="text-left px-3 py-1.5 font-medium text-gray-500">{t.keyInfo || '关键信息'}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each planPreviewModal.data.typeSummary as ts}
                    {@const firstRes = planPreviewModal.data.resources.find(r => r.type === ts.type)}
                    {@const detailStr = firstRes?.detail ? Object.values(firstRes.detail).join(' · ') : ''}
                    <tr class="border-b border-gray-50 last:border-0">
                      <td class="px-3 py-1.5">
                        <span class="font-medium text-gray-700">{ts.label}</span>
                        <span class="text-gray-400 ml-1 text-[10px]">{ts.type}</span>
                      </td>
                      <td class="text-center px-3 py-1.5">
                        <span class="inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 rounded-full text-[11px] font-bold {ts.count > 1 ? 'bg-blue-50 text-blue-600' : 'bg-gray-50 text-gray-500'}">
                          {ts.count}
                        </span>
                      </td>
                      <td class="px-3 py-1.5 text-gray-500 text-[11px] truncate max-w-[300px]" title={detailStr}>
                        {detailStr || '-'}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}

          <!-- Topology SVG -->
          {#if elkNodes.length > 0}
            <div class="relative">
              <ZoomControls bind:zoom={topoZoom} {t} />
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="bg-gray-50 rounded-xl border border-gray-200 overflow-auto" style="max-height: 65vh;"
                onwheel={(e) => { e.preventDefault(); topoZoom = Math.min(3, Math.max(0.3, topoZoom + (e.deltaY > 0 ? -0.08 : 0.08))); }}>
                <svg viewBox={svgViewBox} preserveAspectRatio="xMidYMid meet" class="w-full" style="min-height: 350px; transform: scale({topoZoom}); transform-origin: center top;">
                  <defs>
                    <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
                      <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
                    </marker>
                  </defs>

                  {#each elkEdges as edge}
                    <path d={edgePath(edge)} fill="none" stroke="#cbd5e1" stroke-width="1.5" marker-end="url(#arrowhead)" class="transition-colors hover:stroke-blue-400" />
                  {/each}

                  {#each elkNodes as node}
                    <g transform="translate({node.x}, {node.y})">
                      <rect width={node.w} height={node.h} rx="8" ry="8"
                        fill="white" stroke={node.color.border} stroke-width="2" class="drop-shadow-sm" />
                      <circle cx="14" cy="20" r="4" fill={node.color.border} />
                      <text x="26" y="18" font-size="11" font-weight="600" fill="#374151" font-family="system-ui, sans-serif">
                        {node.label}
                      </text>
                      <text x="26" y="32" font-size="9" fill="#9ca3af" font-family="system-ui, sans-serif">
                        {node.resource?.name || ''}
                      </text>
                      {#if node.detailText}
                        <text x="26" y="46" font-size="8" fill="#6b7280" font-family="system-ui, sans-serif" font-style="italic">
                          {node.detailText}
                        </text>
                      {/if}
                      <text x={node.w - 10} y="18" font-size="11" font-weight="700" fill={node.color.text} text-anchor="end" font-family="system-ui, sans-serif">
                        {node.color.label}
                      </text>
                    </g>
                  {/each}
                </svg>
              </div>
            </div>
          {:else}
            <!-- Fallback: simple resource list -->
            <div class="space-y-1.5">
              {#each planPreviewModal.data.resources as r}
                {@const color = getActionColor(r.actions)}
                <div class="flex items-center gap-2 px-3 py-2 bg-white rounded-lg border" style="border-color: {color.border}20;">
                  <span class="w-5 h-5 rounded flex items-center justify-center text-[11px] font-bold" style="background: {color.bg}; color: {color.text};">{color.label}</span>
                  <span class="text-[12px] font-medium text-gray-700">{r.type}</span>
                  <span class="text-[12px] text-gray-400">.{r.name}</span>
                </div>
              {/each}
            </div>
          {/if}
        {/if}
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-100">
        <button
          class="h-9 px-4 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
          onclick={() => planPreviewModal = { ...planPreviewModal, show: false }}
        >{t.close || '关闭'}</button>
        {#if planPreviewModal.data && planPreviewModal.data.hasChanges}
          <button
            class="h-9 px-4 text-[13px] font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 transition-colors cursor-pointer"
            onclick={handleStartFromPreview}
          >{t.planStartScene || '启动场景'}</button>
        {/if}
      </div>
    </div>
</Modal>

<!-- Spot Terminated Toast -->
{#if spotTerminatedToast.show}
  <div class="fixed bottom-6 right-6 z-[9999] max-w-sm">
    <div class="flex items-start gap-3 bg-red-50 border border-red-300 rounded-xl shadow-lg px-4 py-3">
      <span class="text-red-500 mt-0.5">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
        </svg>
      </span>
      <div class="flex-1">
        <p class="text-[13px] font-semibold text-red-800">{t.spotTerminatedTitle || '抢占式实例已回收'}</p>
        <p class="text-[12px] text-red-600 mt-0.5">{spotTerminatedToast.caseName}</p>
        {#if spotTerminatedToast.downIPs.length > 0}
          <p class="text-[11px] text-red-500 mt-0.5 font-mono">{spotTerminatedToast.downIPs.join(', ')}</p>
        {/if}
      </div>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => spotTerminatedToast = { show: false, caseName: '', downIPs: [], allDown: false, timer: null }}>
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  </div>
{/if}

<!-- Spot Recovery Toast -->
{#if spotRecoveryToast.show}
  <div class="fixed bottom-6 right-6 z-[9998] max-w-sm" style={spotTerminatedToast.show ? 'bottom: 7rem' : ''}>
    <div class="flex items-start gap-3 rounded-xl shadow-lg px-4 py-3 {spotRecoveryToast.status === 'recovered' ? 'bg-emerald-50 border border-emerald-300' : spotRecoveryToast.status === 'failed' ? 'bg-amber-50 border border-amber-300' : 'bg-blue-50 border border-blue-300'}">
      <span class="mt-0.5 {spotRecoveryToast.status === 'recovered' ? 'text-emerald-500' : spotRecoveryToast.status === 'failed' ? 'text-amber-500' : 'text-blue-500'}">
        {#if spotRecoveryToast.status === 'recovering'}
          <svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
        {:else if spotRecoveryToast.status === 'recovered'}
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
        {:else}
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" /></svg>
        {/if}
      </span>
      <div class="flex-1">
        <p class="text-[13px] font-semibold {spotRecoveryToast.status === 'recovered' ? 'text-emerald-800' : spotRecoveryToast.status === 'failed' ? 'text-amber-800' : 'text-blue-800'}">
          {spotRecoveryToast.status === 'recovering' ? (t.spotRecovering || '正在自动恢复...') : spotRecoveryToast.status === 'recovered' ? (t.spotRecovered || '已自动恢复') : (t.spotRecoverFailed || '自动恢复失败')}
        </p>
        <p class="text-[12px] mt-0.5 {spotRecoveryToast.status === 'recovered' ? 'text-emerald-600' : spotRecoveryToast.status === 'failed' ? 'text-amber-600' : 'text-blue-600'}">{spotRecoveryToast.caseName}</p>
        {#if spotRecoveryToast.status === 'failed'}
          <p class="text-[11px] text-amber-500 mt-0.5">{t.spotRecoverFailedHint || '可能暂无库存，稍后将重试'}</p>
        {/if}
      </div>
      <button class="opacity-50 hover:opacity-100 cursor-pointer" onclick={() => spotRecoveryToast = { show: false, caseName: '', status: '', timer: null }}>
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  </div>
{/if}

<!-- Clone Dialog -->
<Modal show={cloneDialog.show} onclose={cancelClone}>
  <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <div class="px-6 py-5">
      <div class="flex items-center gap-3 mb-4">
        <div class="w-10 h-10 rounded-full bg-blue-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" />
          </svg>
        </div>
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.cloneCase || '克隆场景'}</h3>
          <p class="text-[13px] text-gray-500">{cloneDialog.sourceName}</p>
        </div>
      </div>
      <label class="block text-[13px] font-medium text-gray-700 mb-1.5">{t.sceneName || '场景名称'}</label>
      <input
        type="text"
        class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
        bind:value={cloneDialog.caseName}
        onkeydown={(e) => { if (e.key === 'Enter' && cloneDialog.caseName.trim()) confirmClone(); }}
        autofocus
      />
    </div>
    <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
      <button 
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={cancelClone}
      >{t.cancel}</button>
      <button 
        class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
        onclick={confirmClone}
        disabled={!cloneDialog.caseName.trim()}
      >{t.clone || '克隆'}</button>
    </div>
  </div>
</Modal>

<!-- Clone Loading Overlay -->
{#if cloneLoading}
  <div class="fixed bottom-4 right-4 z-50 bg-white rounded-lg border border-blue-200 px-4 py-3 flex items-center gap-3 shadow-lg">
    <svg class="w-5 h-5 text-blue-600 animate-spin" fill="none" viewBox="0 0 24 24">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
    </svg>
    <span class="text-[13px] text-blue-700 font-medium">{t.cloneCase || '克隆场景'}...</span>
  </div>
{/if}

<!-- Context Menu -->
{#if contextMenu.show}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-[100]" onclick={closeContextMenu} oncontextmenu={(e) => { e.preventDefault(); closeContextMenu(); }}></div>
  <div
    class="fixed z-[101] bg-white rounded-lg shadow-xl border border-gray-200 py-1 min-w-[180px] text-[13px]"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px;"
  >
    {#if contextMenu.caseState !== 'starting' && contextMenu.caseState !== 'stopping' && contextMenu.caseState !== 'removing'}
      {#if contextMenu.caseState === 'running'}
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => sshModal = { show: true, caseId: contextMenu.caseId, caseName: contextMenu.caseName })}>
          <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
          {t.sshOperations || 'SSH 运维'}
        </button>
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => toggleCaseExpand(contextMenu.caseId, contextMenu.caseState))}>
          <svg class="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 00-1.883 2.542l.857 6a2.25 2.25 0 002.227 1.932H19.05a2.25 2.25 0 002.227-1.932l.857-6a2.25 2.25 0 00-1.883-2.542m-16.5 0V6A2.25 2.25 0 016 3.75h3.879a1.5 1.5 0 011.06.44l2.122 2.12a1.5 1.5 0 001.06.44H18A2.25 2.25 0 0120.25 9v.776" /></svg>
          {t.viewOutput || '查看输出'}
        </button>
        <div class="border-t border-gray-100 my-1"></div>
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-amber-600" onclick={() => ctxAction(() => showStopConfirm(contextMenu.caseId, contextMenu.caseName))}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
          {t.stop}
        </button>
      {:else}
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-emerald-600" onclick={() => ctxAction(() => handleStart(contextMenu.caseId))}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" /></svg>
          {t.start}
        </button>
        {#if contextMenu.caseState === 'created' || contextMenu.caseState === 'stopped'}
          <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => handlePlanPreview(contextMenu.caseId, contextMenu.caseName))}>
            <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
            {t.planPreviewBtn || '预览'}
          </button>
        {/if}
      {/if}
      <div class="border-t border-gray-100 my-1"></div>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => showCloneDialog(contextMenu.caseId, contextMenu.caseName))}>
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" /></svg>
        {t.cloneCase || '克隆场景'}
      </button>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => { tagEditCase = contextMenu.caseId; tagInput = ''; })}>
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" /></svg>
        {t.tags || '标签'}
      </button>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => { const id = contextMenu.caseId; const name = contextMenu.caseName; const type = contextMenu.caseType; closeContextMenu(); localStorage.setItem('ai-chat-pending-error', JSON.stringify({ error: `${t.sceneLabel || '场景'}: ${name} (${type})`, templateName: type, provider: getProviderFromTemplate(type), source: 'context-menu' })); onTabChange('aiChat'); }}>
        <svg class="w-4 h-4 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" /></svg>
        {t.sendToAI || '发送到 AI'}
      </button>
      <div class="border-t border-gray-100 my-1"></div>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-red-600" onclick={() => ctxAction(() => showDeleteConfirm(contextMenu.caseId, contextMenu.caseName))}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
        {t.delete}
      </button>
    {:else}
      <div class="px-3 py-2 text-gray-400 text-[12px]">
        {stateConfig[contextMenu.caseState]?.label || t.processing}...
      </div>
    {/if}
  </div>
{/if}
