<script lang="ts">
  import { onMount } from 'svelte';
  import { ListCustomDeployments, StartCustomDeployment, StopCustomDeployment, DeleteCustomDeployment, BatchStartCustomDeployments, BatchStopCustomDeployments, BatchDeleteCustomDeployments, GetDeploymentPlanPreview, SetCaseTags, GetAllCaseTags, GetAllTagNames, CloneCustomDeployment } from '../../../wailsjs/go/main/App';
  import { toast } from '../../lib/toast.js';
  import SSHModal from '../Cases/SSHModal.svelte';
  import ScheduleDialog from '../Cases/ScheduleDialog.svelte';
  import ELK from 'elkjs/lib/elk.bundled.js';

  let { t, onSelectDeployment = () => {}, onRefresh = () => {}, onTabChange = () => {} } = $props();

  interface CustomDeployment {
    id: string;
    name: string;
    template_name: string;
    state: string;
    created_at: string;
    updated_at: string;
    outputs?: Record<string, any>;
    config?: {
      provider: string;
      region: string;
      instance_type: string;
    };
  }

  let deployments = $state<CustomDeployment[]>([]);
  let loading = $state(false);
  let error = $state('');
  let expandedDeploymentId = $state('');
  let deploymentOutputs = $state<Record<string, any>>({});
  let selectedDeploymentIds = $state<Set<string>>(new Set());
  let batchMode = $state(false);
  let batchOperating = $state(false);
  let batchDeleteConfirm = $state({ show: false, count: 0 });
  let batchStopConfirm = $state({ show: false, count: 0 });
  let copiedKey = $state<string | null>(null);
  let copiedAllKey = $state<string | null>(null);
  let pollInterval: number | null = null;
  
  // SSH Modal state
  let sshModal = $state<{ show: boolean; deploymentId: string; deploymentName: string }>({ 
    show: false, 
    deploymentId: '', 
    deploymentName: '' 
  });
  
  // Schedule Dialog state
  let scheduleDialog = $state<{ show: boolean; deploymentId: string; deploymentName: string; action: string }>({ 
    show: false, 
    deploymentId: '', 
    deploymentName: '', 
    action: '' 
  });

  // Plan preview state
  let planPreviewModal = $state({ show: false, deploymentName: '', deploymentId: '', loading: false, error: '', data: null as any, isSpot: false });
  let elkNodes = $state<any[]>([]);
  let elkEdges = $state<any[]>([]);
  let svgViewBox = $state('0 0 800 600');
  let topoZoom = $state(1);

  // Tag state
  let allTagNames = $state<string[]>([]);
  let caseTags = $state<Record<string, string[]>>({});
  let selectedTag = $state('');
  let tagEditId = $state<string | null>(null);
  let tagInput = $state('');
  let filteredDeployments = $derived(selectedTag ? deployments.filter(d => (caseTags[d.id] || []).includes(selectedTag)) : deployments);

  // Clone dialog state
  let cloneDialog = $state({ show: false, deploymentId: '', cloneName: '', sourceName: '' });
  let cloneLoading = $state(false);

  // Context menu state
  let contextMenu = $state({ show: false, x: 0, y: 0, id: '', name: '', state: '', templateName: '' });

  const tagColors = [
    'bg-blue-100 text-blue-700', 'bg-purple-100 text-purple-700', 'bg-pink-100 text-pink-700',
    'bg-indigo-100 text-indigo-700', 'bg-teal-100 text-teal-700', 'bg-orange-100 text-orange-700',
    'bg-cyan-100 text-cyan-700', 'bg-rose-100 text-rose-700', 'bg-lime-100 text-lime-700',
    'bg-amber-100 text-amber-700',
  ];
  function getTagColor(tag: string) {
    let hash = 0;
    for (let i = 0; i < tag.length; i++) hash = ((hash << 5) - hash + tag.charCodeAt(i)) | 0;
    return tagColors[Math.abs(hash) % tagColors.length];
  }

  async function loadTags() {
    try {
      [caseTags, allTagNames] = await Promise.all([GetAllCaseTags(), GetAllTagNames()]);
    } catch { caseTags = {}; allTagNames = []; }
  }

  async function addTagToDeployment(id: string, tag: string) {
    if (!tag.trim()) return;
    const tags = [...(caseTags[id] || [])];
    if (tags.includes(tag.trim())) return;
    tags.push(tag.trim());
    await SetCaseTags(id, tags);
    await loadTags();
    tagInput = '';
  }

  async function removeTagFromDeployment(id: string, tag: string) {
    const tags = (caseTags[id] || []).filter(t => t !== tag);
    await SetCaseTags(id, tags);
    await loadTags();
  }

  // Running time ticker
  let nowTick = $state(Date.now());
  let tickTimer: number | null = null;

  function formatElapsed(fromStr: string): string {
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

  // 状态颜色配置（与创建部署页面一致）
  const stateConfig = $derived<Record<string, { label: string; color: string; bg: string; dot: string }>>({
    'pending': { label: t.pending || '待部署', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: t.starting || '启动中', color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500 animate-pulse' },
    'running': { label: t.running || '运行中', color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopping': { label: t.stopping || '停止中', color: 'text-orange-600', bg: 'bg-orange-50', dot: 'bg-orange-500 animate-pulse' },
    'stopped': { label: t.stopped || '已停止', color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'removing': { label: t.removing || '删除中', color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500 animate-pulse' },
    'error': { label: t.error || '错误', color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' },
    'terminated': { label: t.terminated || '已回收', color: 'text-red-700', bg: 'bg-red-50', dot: 'bg-red-600 animate-pulse' }
  });

  // 云厂商映射
  const providerLabels = $derived<Record<string, string>>({
    'alicloud': t.Aliyun || '阿里云',
    'tencentcloud': t.TencentCloud || '腾讯云',
    'aws': t.AWS || 'AWS',
    'volcengine': t.Volcengine || '火山引擎',
    'huaweicloud': t.HuaweiCloud || '华为云'
  });

  // 截断 ID 显示（参考预定义场景页面）
  function getShortId(id: string): string {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }

  async function loadDeployments() {
    loading = true;
    error = '';
    try {
      const result = await ListCustomDeployments();
      deployments = (result || []) as any;
      await loadTags();
      
      // 检查是否有中间态的部署，如果有则启动轮询
      checkAndStartPolling();
    } catch (err) {
      error = `加载部署列表失败: ${err}`;
      console.error('Failed to load deployments:', err);
    } finally {
      loading = false;
    }
  }

  // 检查是否有中间态的部署
  function hasTransitioningDeployments(): boolean {
    return deployments.some(d => 
      d.state === 'starting' || 
      d.state === 'stopping' || 
      d.state === 'removing'
    );
  }

  // 检查并启动/停止轮询
  function checkAndStartPolling() {
    if (hasTransitioningDeployments()) {
      // 如果有中间态的部署且轮询未启动，则启动轮询
      if (!pollInterval) {
        console.log('启动部署状态轮询');
        pollInterval = window.setInterval(() => {
          loadDeploymentsQuietly();
        }, 3000); // 每3秒轮询一次
      }
    } else {
      // 如果没有中间态的部署，停止轮询
      if (pollInterval) {
        console.log('停止部署状态轮询');
        clearInterval(pollInterval);
        pollInterval = null;
      }
    }
  }

  // 静默加载（不显示 loading 状态）
  async function loadDeploymentsQuietly() {
    try {
      const result = await ListCustomDeployments();
      deployments = (result || []) as any;
      
      // 检查是否还需要继续轮询
      checkAndStartPolling();
    } catch (err) {
      console.error('轮询加载部署列表失败:', err);
      // 轮询失败时停止轮询
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    }
  }

  function handleSelectDeployment(deployment: CustomDeployment) {
    if (batchMode) {
      toggleDeploymentSelection(deployment.id);
    } else {
      toggleDeploymentExpand(deployment.id);
    }
  }

  async function toggleDeploymentExpand(deploymentId: string) {
    if (expandedDeploymentId === deploymentId) {
      expandedDeploymentId = '';
    } else {
      expandedDeploymentId = deploymentId;
      // 查找部署对象
      const deployment = deployments.find(d => d.id === deploymentId);
      if (deployment && deployment.outputs) {
        deploymentOutputs[deploymentId] = deployment.outputs;
      }
    }
  }

  function copyToClipboard(value: string, key: string) {
    navigator.clipboard.writeText(value).then(() => {
      copiedKey = key;
      setTimeout(() => {
        copiedKey = null;
      }, 2000);
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  }

  function copyAllOutputs(outputs: Record<string, any>) {
    const text = Object.entries(outputs)
      .map(([key, value]) => `${key}=${value}`)
      .join('\n');
    navigator.clipboard.writeText(text).then(() => {
      copiedAllKey = expandedDeploymentId;
      setTimeout(() => {
        copiedAllKey = null;
      }, 2000);
    }).catch(err => {
      console.error('Failed to copy all outputs:', err);
    });
  }

  async function handleStart(deploymentId: string) {
    // 立即更新本地状态为"启动中"
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'starting' } : d
    );
    
    try {
      await StartCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      toast.error(`启动失败: ${err.message || err}`);
      // 失败后重新加载以恢复正确状态
      await loadDeployments();
    }
  }

  function showCloneDialog(deploymentId: string, deploymentName: string) {
    cloneDialog = { show: true, deploymentId, cloneName: deploymentName + '-clone', sourceName: deploymentName };
  }

  function cancelClone() {
    cloneDialog = { show: false, deploymentId: '', cloneName: '', sourceName: '' };
  }

  async function confirmClone() {
    const { deploymentId, cloneName } = cloneDialog;
    cloneDialog = { show: false, deploymentId: '', cloneName: '', sourceName: '' };
    cloneLoading = true;
    try {
      await CloneCustomDeployment(deploymentId, cloneName);
      await loadDeployments();
    } catch (err: any) {
      toast.error(`${t.cloneFailed || '克隆失败'}: ${err.message || err}`);
    } finally {
      cloneLoading = false;
    }
  }

  async function handleStop(deploymentId: string, deploymentName: string) {
    stopConfirm = { show: true, deploymentId, deploymentName };
  }

  function cancelStop() {
    stopConfirm = { show: false, deploymentId: '', deploymentName: '' };
  }

  async function confirmStop() {
    const { deploymentId } = stopConfirm;
    stopConfirm = { show: false, deploymentId: '', deploymentName: '' };
    
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'stopping' } : d
    );
    
    try {
      await StopCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      toast.error(`停止失败: ${err.message || err}`);
      await loadDeployments();
    }
  }

  let deleteConfirm = $state({ show: false, deploymentId: '', deploymentName: '' });
  let stopConfirm = $state({ show: false, deploymentId: '', deploymentName: '' });

  async function handleDelete(deploymentId: string, deploymentName: string) {
    deleteConfirm = { show: true, deploymentId, deploymentName };
  }

  function cancelDelete() {
    deleteConfirm = { show: false, deploymentId: '', deploymentName: '' };
  }

  async function confirmDelete() {
    const { deploymentId, deploymentName } = deleteConfirm;
    deleteConfirm = { show: false, deploymentId: '', deploymentName: '' };
    
    // 立即更新本地状态为"删除中"
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'removing' } : d
    );
    
    try {
      await DeleteCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      toast.error(`删除失败: ${err.message || err}`);
      // 失败后重新加载以恢复正确状态
      await loadDeployments();
    }
  }

  // AI 分析错误 - 跳转到 AI 对话页面
  function handleAIAnalysis(deploymentId: string, errorMessage: string, provider: string, templateName: string) {
    if (!errorMessage) {
      toast.warning(t.noErrorToAnalyze || '没有错误信息可以分析');
      return;
    }
    
    localStorage.setItem('ai-chat-pending-error', JSON.stringify({
      error: errorMessage,
      templateName,
      provider,
      source: 'customDeployment'
    }));
    onTabChange('aiChat');
  }

  function showSSHModal(deploymentId: string, deploymentName: string) {
    sshModal = { show: true, deploymentId, deploymentName };
  }

  function openContextMenu(e: MouseEvent, d: CustomDeployment) {
    e.preventDefault();
    const x = Math.min(e.clientX, window.innerWidth - 200);
    const y = Math.min(e.clientY, window.innerHeight - 300);
    contextMenu = { show: true, x, y, id: d.id, name: d.name, state: d.state, templateName: d.template_name };
  }

  function closeContextMenu() {
    contextMenu = { show: false, x: 0, y: 0, id: '', name: '', state: '', templateName: '' };
  }

  function ctxAction(fn: () => void) {
    fn();
    closeContextMenu();
  }

  function showScheduleDialog(deploymentId: string, deploymentName: string, action: string) {
    scheduleDialog = { show: true, deploymentId, deploymentName, action };
  }

  function handleScheduleSuccess() {
    scheduleDialog = { show: false, deploymentId: '', deploymentName: '', action: '' };
    // 可以添加成功提示
  }

  // Plan preview topology
  async function handlePlanPreview(deploymentId: string, deploymentName: string, isSpot: boolean = false) {
    planPreviewModal = { show: true, deploymentName, deploymentId, loading: true, error: '', data: null, isSpot };
    elkNodes = [];
    elkEdges = [];
    topoZoom = 1;
    try {
      const data = await GetDeploymentPlanPreview(deploymentId);
      if (data && data.hasChanges && data.resources && data.resources.length > 0) {
        await layoutTopology(data);
      }
      planPreviewModal = { ...planPreviewModal, data, loading: false };
    } catch (e: any) {
      planPreviewModal = { ...planPreviewModal, loading: false, error: e.message || String(e) };
    }
  }

  function getActionColor(actions: string[]) {
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

  function getNodeLabel(resource: any) {
    const parts = (resource.type || '').split('_');
    if (parts.length <= 1) return resource.type || '';
    const rest = parts.slice(1).map((w: string) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
    return rest;
  }

  function getNodeDetail(resource: any) {
    const d = resource?.detail;
    if (!d) return '';
    const parts: string[] = [];
    if (d.instance_type) parts.push(d.instance_type);
    if (d.cidr) parts.push(d.cidr);
    if (d.rule) parts.push(d.rule);
    if (d.ingress) parts.push('in: ' + d.ingress);
    if (d.egress && !d.ingress) parts.push('out: ' + d.egress);
    return parts.join(' | ').substring(0, 60);
  }

  async function layoutTopology(data: any) {
    try {
      const elk = new ELK();
      const NODE_W = 200;
      const hasDetail = data.resources.some((r: any) => r.detail && Object.keys(r.detail).length > 0);
      const NODE_H = hasDetail ? 58 : 48;
      const addrSet = new Set(data.resources.map((r: any) => r.address));
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
        children: data.resources.map((r: any) => ({ id: r.address, width: NODE_W, height: NODE_H })),
        edges: (data.edges || [])
          .filter((e: any) => addrSet.has(e.from) && addrSet.has(e.to))
          .map((e: any, i: number) => ({ id: `e${i}`, sources: [e.from], targets: [e.to] })),
      };

      const layout = await elk.layout(graph);
      const resMap: Record<string, any> = {};
      data.resources.forEach((r: any) => { resMap[r.address] = r; });

      const newNodes = (layout.children || []).map((n: any) => ({
        id: n.id, x: n.x, y: n.y, w: n.width, h: n.height,
        resource: resMap[n.id],
        color: getActionColor(resMap[n.id]?.actions),
        label: getNodeLabel(resMap[n.id] || {}),
        detailText: getNodeDetail(resMap[n.id] || {}),
      }));

      const newEdges = (layout.edges || []).map((e: any) => {
        const sections = e.sections || [];
        if (sections.length > 0) {
          const s = sections[0];
          return { id: e.id, startPoint: s.startPoint, endPoint: s.endPoint, bendPoints: s.bendPoints || [] };
        }
        const src = newNodes.find((n: any) => n.id === (e.sources?.[0]));
        const tgt = newNodes.find((n: any) => n.id === (e.targets?.[0]));
        if (src && tgt) {
          return { id: e.id, startPoint: { x: src.x + src.w / 2, y: src.y + src.h }, endPoint: { x: tgt.x + tgt.w / 2, y: tgt.y }, bendPoints: [] };
        }
        return null;
      }).filter(Boolean);

      const padding = 40;
      const maxX = Math.max(...newNodes.map((n: any) => n.x + n.w), 400) + padding * 2;
      const maxY = Math.max(...newNodes.map((n: any) => n.y + n.h), 200) + padding * 2;
      elkNodes = newNodes;
      elkEdges = newEdges;
      svgViewBox = `0 0 ${maxX} ${maxY}`;
    } catch (e) {
      console.error('ELK layout failed:', e);
    }
  }

  function edgePath(edge: any) {
    const { startPoint, endPoint, bendPoints } = edge;
    if (bendPoints.length === 0) {
      const midY = (startPoint.y + endPoint.y) / 2;
      return `M ${startPoint.x} ${startPoint.y} C ${startPoint.x} ${midY}, ${endPoint.x} ${midY}, ${endPoint.x} ${endPoint.y}`;
    }
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
    const id = planPreviewModal.deploymentId;
    planPreviewModal = { show: false, deploymentName: '', deploymentId: '', loading: false, error: '', data: null, isSpot: false };
    handleStart(id);
  }

  function toggleDeploymentSelection(id: string) {
    if (selectedDeploymentIds.has(id)) {
      const newSet = new Set(selectedDeploymentIds);
      newSet.delete(id);
      selectedDeploymentIds = newSet;
    } else {
      const newSet = new Set(selectedDeploymentIds);
      newSet.add(id);
      selectedDeploymentIds = newSet;
    }
  }

  function toggleBatchMode() {
    batchMode = !batchMode;
    if (!batchMode) {
      selectedDeploymentIds = new Set();
    }
  }

  function selectAll() {
    selectedDeploymentIds = new Set(deployments.map(d => d.id));
  }

  function deselectAll() {
    selectedDeploymentIds = new Set();
  }

  function getSelectedDeployments(): CustomDeployment[] {
    return deployments.filter(d => selectedDeploymentIds.has(d.id));
  }

  function showBatchDeleteConfirm() {
    batchDeleteConfirm = { show: true, count: selectedDeploymentIds.size };
  }

  function cancelBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
  }

  async function confirmBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchDeleteCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  function showBatchStopConfirm() {
    batchStopConfirm = { show: true, count: selectedDeploymentIds.size };
  }

  function cancelBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
  }

  async function confirmBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchStopCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  async function handleBatchStart() {
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchStartCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  // Export batch selection info for parent component
  export function getBatchSelection() {
    return {
      mode: batchMode,
      selectedIds: Array.from(selectedDeploymentIds),
      selectedDeployments: getSelectedDeployments()
    };
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    try {
      const date = new Date(dateStr);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return dateStr;
    }
  }

  function handleRefresh() {
    loadDeployments();
    onRefresh();
  }

  onMount(() => {
    loadDeployments();
    tickTimer = window.setInterval(() => { nowTick = Date.now(); }, 60000);
    
    // 组件卸载时清理轮询
    return () => {
      if (tickTimer) { clearInterval(tickTimer); tickTimer = null; }
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    };
  });

  // 导出刷新方法供父组件调用
  export function refresh() {
    loadDeployments();
  }
</script>

<div class="deployment-list">
  <div class="list-header">
    <div class="header-left">
      <h3>{t.customDeploymentList || '自定义部署列表'}</h3>
      {#if batchMode && selectedDeploymentIds.size > 0}
        <span class="selection-count">已选择 {selectedDeploymentIds.size} 项</span>
      {/if}
    </div>
    <div class="header-actions">
      <button 
        class="btn-batch-mode" 
        class:active={batchMode}
        onclick={toggleBatchMode}
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
        </svg>
        {batchMode ? (t.exitBatch || '退出批量') : (t.batchOperation || '批量操作')}
      </button>
      {#if batchMode}
        <button class="btn-select-all" onclick={selectAll}>{t.selectAll || '全选'}</button>
        <button class="btn-deselect-all" onclick={deselectAll}>{t.deselectAll || '取消全选'}</button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-md hover:bg-emerald-100 transition-colors disabled:opacity-50"
          onclick={handleBatchStart}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchStart || '批量启动'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-md hover:bg-amber-100 transition-colors disabled:opacity-50"
          onclick={showBatchStopConfirm}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchStop || '批量停止'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-red-700 bg-red-50 border border-red-200 rounded-md hover:bg-red-100 transition-colors disabled:opacity-50"
          onclick={showBatchDeleteConfirm}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchDelete || '批量删除'}
        </button>
      {/if}
      <button class="btn-refresh" onclick={handleRefresh} disabled={loading}>
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {t.refresh || '刷新'}
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>加载中...</p>
    </div>
  {:else if error}
    <div class="error-message">
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p>{error}</p>
      <button class="btn-retry" onclick={loadDeployments}>{t.retry || '重试'}</button>
    </div>
  {:else if deployments.length === 0}
    <div class="empty-state">
      <svg class="icon-large" viewBox="0 0 24 24" fill="none" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
      </svg>
      <p>暂无自定义部署</p>
      <p class="hint">创建一个新的自定义部署开始使用</p>
    </div>
  {:else}
    <div class="table-container">
      <!-- Tag Filter Bar -->
      {#if allTagNames.length > 0}
        <div class="px-5 py-2.5 border-b border-gray-100 flex items-center gap-2 flex-wrap">
          <span class="text-[11px] text-gray-400 mr-1">{t.tags || '标签'}:</span>
          <button
            class="px-2 py-0.5 text-[11px] rounded-full transition-colors {selectedTag === '' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
            onclick={() => { selectedTag = ''; }}
          >{t.tagFilterAll || '全部'}</button>
          {#each allTagNames as tag}
            <button
              class="px-2 py-0.5 text-[11px] rounded-full transition-colors {selectedTag === tag ? 'bg-gray-900 text-white' : getTagColor(tag)}"
              onclick={() => { selectedTag = selectedTag === tag ? '' : tag; }}
            >{tag}</button>
          {/each}
        </div>
      {/if}
      <table class="deployment-table">
        <thead>
          <tr>
            {#if batchMode}
              <th class="checkbox-col"></th>
            {/if}
            <th>{t.name || '名称'}</th>
            <th>{t.template || '模板'}</th>
            <th>{t.provider || '云厂商'}</th>
            <th>{t.region || '地域'}</th>
            <th>{t.status || '状态'}</th>
            <th>{t.createdAt || '创建时间'}</th>
            <th class="text-right">{t.action || '操作'}</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredDeployments as deployment (deployment.id)}
            <tr 
              class:selected={!batchMode && expandedDeploymentId === deployment.id}
              class:batch-selected={batchMode && selectedDeploymentIds.has(deployment.id)}
              onclick={() => handleSelectDeployment(deployment)}
              oncontextmenu={(e) => openContextMenu(e, deployment)}
            >
              {#if batchMode}
                <td class="checkbox-col" onclick={(e) => e.stopPropagation()}>
                  <input 
                    type="checkbox" 
                    checked={selectedDeploymentIds.has(deployment.id)}
                    onchange={() => toggleDeploymentSelection(deployment.id)}
                  />
                </td>
              {/if}
              <td class="name-cell">
                <div class="name-content">
                  <div class="flex items-center gap-2">
                    <svg class="w-4 h-4 text-gray-400 transition-transform {expandedDeploymentId === deployment.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                    <span class="name">{deployment.name}</span>
                    {#each caseTags[deployment.id] || [] as tag}
                      <span class="px-1.5 py-0 text-[10px] rounded-full {getTagColor(tag)}">{tag}</span>
                    {/each}
                  </div>
                  <code class="id">{getShortId(deployment.id)}</code>
                </div>
              </td>
              <td>{deployment.template_name || '-'}</td>
              <td>
                {deployment.config?.provider ? providerLabels[deployment.config.provider] || deployment.config.provider : '-'}
              </td>
              <td>{deployment.config?.region || '-'}</td>
              <td>
                <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {stateConfig[deployment.state]?.color || 'text-gray-600'}">
                  <span class="w-1.5 h-1.5 rounded-full {stateConfig[deployment.state]?.dot || 'bg-gray-400'}"></span>
                  {stateConfig[deployment.state]?.label || deployment.state}
                </span>
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                {#if (deployment.config as any)?.is_spot_instance}
                  <span class="ml-1.5 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded">
                    {t.spotInstance || '抢占式'}
                  </span>
                {/if}
              </td>
              <td class="date-cell">
                {formatDate(deployment.created_at)}
                {#if deployment.state === 'running' && deployment.updated_at}
                  <span class="ml-1.5 text-[11px] text-emerald-600 font-medium" title={t.runningTime || '运行时间'}>⏱ {formatElapsed(deployment.updated_at)}</span>
                {/if}
              </td>
              <td class="px-5 py-3.5 text-right" onclick={(e) => e.stopPropagation()}>
                <div class="inline-flex items-center gap-1">
                  {#if deployment.state !== 'starting' && deployment.state !== 'stopping' && deployment.state !== 'removing'}
                    <!-- Tag edit button -->
                    <button
                      class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                      onclick={() => { tagEditId = tagEditId === deployment.id ? null : deployment.id; tagInput = ''; }}
                      title={t.tags || '标签'}
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
                      </svg>
                    </button>
                    <!-- Clone button -->
                    <button
                      class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                      onclick={() => showCloneDialog(deployment.id, deployment.name)}
                      title={t.cloneCase || '克隆场景'}
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" />
                      </svg>
                    </button>
                  {/if}
                  {#if deployment.state === 'starting' || deployment.state === 'stopping' || deployment.state === 'removing'}
                    <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                      {stateConfig[deployment.state]?.label || '处理中'}...
                    </span>
                  {:else if deployment.state === 'error'}
                    <span class="px-2.5 py-1 text-[12px] font-medium text-red-600">
                      {stateConfig[deployment.state]?.label || '错误'}
                    </span>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                      onclick={() => handleStart(deployment.id)}
                    >{t.retry || '重试'}</button>
                  {:else if deployment.state !== 'running'}
                    <!-- 预览按钮 -->
                    {#if deployment.state === 'stopped' || deployment.state === 'pending'}
                      <button 
                        class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                        onclick={() => handlePlanPreview(deployment.id, deployment.name, !!(deployment.config as any)?.is_spot_instance)}
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
                      onclick={() => showScheduleDialog(deployment.id, deployment.name, 'start')}
                      title="定时启动"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                      onclick={() => handleStart(deployment.id)}
                    >{t.start || '启动'}</button>
                  {:else}
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                      onclick={() => showSSHModal(deployment.id, deployment.name)}
                      title="SSH 运维"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
                      </svg>
                    </button>
                    <!-- 定时停止按钮 -->
                    <button 
                      class="p-1.5 text-gray-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                      onclick={() => showScheduleDialog(deployment.id, deployment.name, 'stop')}
                      title="定时停止"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                      onclick={() => handleStop(deployment.id, deployment.name)}
                    >{t.stop || '停止'}</button>
                  {/if}
                  {#if deployment.state !== 'starting' && deployment.state !== 'stopping' && deployment.state !== 'removing'}
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                      onclick={() => handleDelete(deployment.id, deployment.name)}
                    >{t.delete || '删除'}</button>
                  {/if}
                </div>
              </td>
            </tr>
            <!-- Tag edit row -->
            {#if tagEditId === deployment.id}
              <tr class="bg-blue-50/50">
                <td colspan="8" class="px-5 py-2.5">
                  <div class="flex items-center gap-2 flex-wrap pl-6">
                    <span class="text-[11px] text-gray-500">{t.tags || '标签'}:</span>
                    {#each caseTags[deployment.id] || [] as tag}
                      <span class="inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[11px] rounded-full {getTagColor(tag)}">
                        {tag}
                        <button class="ml-0.5 hover:opacity-70 cursor-pointer" onclick={() => removeTagFromDeployment(deployment.id, tag)}>×</button>
                      </span>
                    {/each}
                    <div class="inline-flex items-center gap-1">
                      <input
                        type="text"
                        class="w-24 text-[11px] px-2 py-0.5 border border-gray-200 rounded-full focus:outline-none focus:ring-1 focus:ring-blue-400"
                        placeholder={t.tagPlaceholder || '输入标签名'}
                        bind:value={tagInput}
                        onkeydown={(e) => { if (e.key === 'Enter') { addTagToDeployment(deployment.id, tagInput); } }}
                        list="tagSuggestionsDep"
                      />
                      <datalist id="tagSuggestionsDep">
                        {#each allTagNames.filter(t => !(caseTags[deployment.id] || []).includes(t)) as suggestion}
                          <option value={suggestion} />
                        {/each}
                      </datalist>
                      <button
                        class="text-[11px] px-2 py-0.5 bg-blue-500 text-white rounded-full hover:bg-blue-600 disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
                        disabled={!tagInput.trim()}
                        onclick={() => addTagToDeployment(deployment.id, tagInput)}
                      >+</button>
                    </div>
                  </div>
                </td>
              </tr>
            {/if}
            <!-- Expanded row for outputs or error -->
            {#if expandedDeploymentId === deployment.id}
              <tr class="bg-slate-50">
                <td colspan="7" class="px-5 py-4">
                  <div class="pl-6">
                    {#if deployment.state === 'error'}
                      <div class="bg-red-50 border border-red-200 rounded-lg p-4">
                        <div class="flex items-start gap-3">
                          <svg class="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                          </svg>
                          <div class="flex-1">
                            <h4 class="text-[13px] font-semibold text-red-900">{t.deploymentFailed || '部署失败'}</h4>
                            <p class="text-[12px] text-red-700 mt-1">{t.checkConfigRetry || '请检查配置后重试。错误详情：'}</p>
                            <pre class="mt-2 p-3 bg-white rounded border border-red-200 text-[11px] text-red-800 overflow-x-auto whitespace-pre-wrap">{deployment.outputs?.error_message || '未知错误'}</pre>
                            
                            <button 
                              class="mt-3 px-3 py-1.5 text-[12px] font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded-md hover:bg-blue-100 transition-colors flex items-center gap-1.5"
                              onclick={() => handleAIAnalysis(deployment.id, deployment.outputs?.error_message || '', deployment.config?.provider || '', deployment.template_name || '')}
                            >
                              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                              </svg>
                              {t.aiAnalyzeError || 'AI 分析错误原因'}
                            </button>
                          </div>
                        </div>
                      </div>
                    {:else if deployment.state === 'running'}
                      {#if deploymentOutputs[deployment.id] && Object.keys(deploymentOutputs[deployment.id]).length > 0}
                        <div class="flex items-center justify-between mb-3">
                          <span class="text-[12px] font-medium text-gray-700">{t.outputInfo || '输出信息'}</span>
                          <button
                            class="px-2 py-1 text-[11px] font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-1"
                            onclick={() => copyAllOutputs(deploymentOutputs[deployment.id])}
                          >
                            {#if copiedAllKey === deployment.id}
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
                          {#each Object.entries(deploymentOutputs[deployment.id]) as [key, value]}
                            <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                              <div class="flex items-center justify-between mb-1">
                                <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                                <button 
                                  class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                  onclick={(e) => { e.stopPropagation(); copyToClipboard(String(value), key); }}
                                >
                                  {#if copiedKey === key}
                                    <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                    </svg>
                                    <span class="text-[10px] text-emerald-500">{t.copied || '已复制'}</span>
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
                        <div class="text-[13px] text-gray-500">{t.noOutput || '部署未运行，无输出信息'}</div>
                      {/if}
                    {:else}
                      <div class="text-[13px] text-gray-500">{t.noOutput || '部署未运行，无输出信息'}</div>
                    {/if}
                </td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- SSH Modal -->
{#if sshModal.show}
  <SSHModal
    t={t}
    caseId={sshModal.deploymentId}
    caseName={sshModal.deploymentName}
    onClose={() => { sshModal = { show: false, deploymentId: '', deploymentName: '' }; }}
  />
{/if}

<!-- Schedule Dialog -->
{#if scheduleDialog.show}
  <ScheduleDialog
    t={t}
    caseId={scheduleDialog.deploymentId}
    caseName={scheduleDialog.deploymentName}
    action={scheduleDialog.action}
    onClose={() => { scheduleDialog = { show: false, deploymentId: '', deploymentName: '', action: '' }; }}
    onSuccess={handleScheduleSuccess}
  />
{/if}

<style>
  .deployment-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #e5e7eb;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .list-header h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #111827;
  }

  .selection-count {
    padding: 4px 12px;
    background: #dbeafe;
    color: #1e40af;
    border-radius: 12px;
    font-size: 13px;
    font-weight: 500;
  }

  .btn-batch-mode {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 14px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-batch-mode:hover {
    background: #f3f4f6;
  }

  .btn-batch-mode.active {
    background: #3b82f6;
    border-color: #3b82f6;
    color: white;
  }

  .btn-select-all,
  .btn-deselect-all {
    padding: 8px 12px;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 13px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-select-all:hover,
  .btn-deselect-all:hover {
    background: #f3f4f6;
  }

  .btn-refresh {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: #f3f4f6;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 14px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-refresh:hover:not(:disabled) {
    background: #e5e7eb;
  }

  .btn-refresh:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .icon {
    width: 16px;
    height: 16px;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #6b7280;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #e5e7eb;
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #dc2626;
  }

  .error-message .icon {
    width: 48px;
    height: 48px;
    margin-bottom: 12px;
  }

  .error-message p {
    margin: 0 0 16px 0;
    text-align: center;
  }

  .btn-retry {
    padding: 8px 20px;
    background: #dc2626;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-retry:hover {
    background: #b91c1c;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: #6b7280;
  }

  .icon-large {
    width: 64px;
    height: 64px;
    margin-bottom: 16px;
    opacity: 0.5;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
  }

  .empty-state .hint {
    margin-top: 8px;
    font-size: 14px;
    color: #9ca3af;
  }

  .table-container {
    flex: 1;
    overflow: auto;
  }

  .deployment-table {
    width: 100%;
    border-collapse: collapse;
  }

  .deployment-table thead {
    position: sticky;
    top: 0;
    background: #f9fafb;
    z-index: 1;
  }

  .deployment-table th {
    padding: 12px 16px;
    text-align: left;
    font-size: 12px;
    font-weight: 600;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #e5e7eb;
  }

  .deployment-table tbody tr {
    cursor: pointer;
    transition: background 0.15s;
  }

  .deployment-table tbody tr:hover {
    background: #f9fafb;
  }

  .deployment-table tbody tr.selected {
    background: #eff6ff;
  }

  .deployment-table tbody tr.batch-selected {
    background: #dbeafe;
  }

  .checkbox-col {
    width: 40px;
    text-align: center;
  }

  .checkbox-col input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .deployment-table td {
    padding: 12px 16px;
    font-size: 14px;
    color: #374151;
    border-bottom: 1px solid #f3f4f6;
  }

  .name-cell {
    max-width: 250px;
  }

  .name-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .name {
    font-weight: 500;
    color: #111827;
  }

  .id {
    font-size: 12px;
    color: #9ca3af;
    font-family: monospace;
    background: #f3f4f6;
    padding: 2px 6px;
    border-radius: 4px;
  }

  .state-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
    background: currentColor;
    color: white;
    opacity: 0.9;
  }

  .date-cell {
    color: #6b7280;
    font-size: 13px;
  }
</style>

<!-- Batch Delete Confirmation Modal -->
{#if batchDeleteConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-visible" onclick={cancelBatchDelete}>
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
        <p class="text-[14px] text-gray-600 mb-4">
          {t.confirmBatchDeleteMessage || '确定要删除选中的'} {batchDeleteConfirm.count} {t.scenes || '个场景'}?
        </p>
        <div class="flex justify-end gap-2">
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
    </div>
  </div>
{/if}

<!-- Stop Confirmation Modal -->
{#if stopConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-visible" onclick={cancelStop}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-orange-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-orange-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">确认停止</h3>
            <p class="text-[13px] text-gray-500">资源将会被销毁</p>
          </div>
        </div>
        <p class="text-[14px] text-gray-600 mb-4">
          确定要停止部署 "{stopConfirm.deploymentName}" 吗？
        </p>
        <div class="flex justify-end gap-2">
          <button 
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
            onclick={cancelStop}
          >取消</button>
          <button 
            class="px-4 py-2 text-[13px] font-medium text-white bg-orange-500 rounded-lg hover:bg-orange-600 transition-colors"
            onclick={confirmStop}
          >停止</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Batch Stop Confirmation Modal -->
{#if batchStopConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-visible" onclick={cancelBatchStop}>
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
            <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
          </div>
        </div>
        <p class="text-[14px] text-gray-600 mb-4">
          {t.confirmBatchStopMessage || '确定要停止选中的'} {batchStopConfirm.count} {t.scenes || '个场景'}?
        </p>
        <div class="flex justify-end gap-2">
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
    </div>
  </div>
{/if}

<!-- Single Delete Confirmation Modal -->
{#if deleteConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-visible" onclick={cancelDelete}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">确认删除</h3>
            <p class="text-[13px] text-gray-500">此操作不可撤销</p>
          </div>
        </div>
        <p class="text-[14px] text-gray-600 mb-4">
          确定要删除部署 "{deleteConfirm.deploymentName}" 吗？
        </p>
        <div class="flex justify-end gap-2">
          <button 
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
            onclick={cancelDelete}
          >取消</button>
          <button 
            class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            onclick={confirmDelete}
          >删除</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Plan Preview Modal -->
{#if planPreviewModal.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => planPreviewModal = { ...planPreviewModal, show: false }}>
    <div class="bg-white rounded-2xl shadow-2xl border border-gray-200 w-[92vw] max-w-[1100px] max-h-[90vh] flex flex-col" onclick={(e) => e.stopPropagation()}>
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.planPreview || '资源拓扑预览'}</h3>
          {#if planPreviewModal.isSpot || planPreviewModal.data?.isSpotInstance}
            <span class="px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded">
              {t.spotInstance || '抢占式'}
            </span>
          {/if}
        </div>
        <span class="text-[12px] text-gray-400 truncate max-w-[200px]">{planPreviewModal.deploymentName}</span>
        <button
          class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors cursor-pointer"
          onclick={() => planPreviewModal = { ...planPreviewModal, show: false }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
                    {@const firstRes = planPreviewModal.data.resources.find((r: any) => r.type === ts.type)}
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
              <!-- Zoom controls -->
              <div class="absolute top-2 right-2 z-10 flex items-center gap-1 bg-white/90 backdrop-blur rounded-lg border border-gray-200 shadow-sm px-1 py-0.5">
                <button class="w-7 h-7 flex items-center justify-center text-gray-500 hover:text-gray-800 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => topoZoom = Math.max(0.3, topoZoom - 0.15)} title="缩小">
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M5 12h14" /></svg>
                </button>
                <span class="text-[11px] text-gray-400 min-w-[36px] text-center">{Math.round(topoZoom * 100)}%</span>
                <button class="w-7 h-7 flex items-center justify-center text-gray-500 hover:text-gray-800 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => topoZoom = Math.min(3, topoZoom + 0.15)} title="放大">
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M12 5v14m-7-7h14" /></svg>
                </button>
                <button class="w-7 h-7 flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors cursor-pointer" onclick={() => topoZoom = 1} title="重置">
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" /></svg>
                </button>
              </div>
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="bg-gray-50 rounded-xl border border-gray-200 overflow-auto" style="max-height: 65vh;"
                onwheel={(e) => { e.preventDefault(); topoZoom = Math.min(3, Math.max(0.3, topoZoom + (e.deltaY > 0 ? -0.08 : 0.08))); }}>
                <svg viewBox={svgViewBox} preserveAspectRatio="xMidYMid meet" class="w-full" style="min-height: 350px; transform: scale({topoZoom}); transform-origin: center top;">
                  <defs>
                    <marker id="deploy-arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
                      <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
                    </marker>
                  </defs>

                  {#each elkEdges as edge}
                    <path d={edgePath(edge)} fill="none" stroke="#cbd5e1" stroke-width="1.5" marker-end="url(#deploy-arrowhead)" class="transition-colors hover:stroke-blue-400" />
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
          >{t.start || '启动'}</button>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Clone Dialog -->
{#if cloneDialog.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelClone}>
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
          bind:value={cloneDialog.cloneName}
          onkeydown={(e) => { if (e.key === 'Enter' && cloneDialog.cloneName.trim()) confirmClone(); }}
          autofocus
        />
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelClone}
        >{t.cancel || '取消'}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          onclick={confirmClone}
          disabled={!cloneDialog.cloneName.trim()}
        >{t.clone || '克隆'}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Clone Loading Toast -->
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
    {#if contextMenu.state !== 'starting' && contextMenu.state !== 'stopping' && contextMenu.state !== 'removing'}
      {#if contextMenu.state === 'running'}
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => showSSHModal(contextMenu.id, contextMenu.name))}>
          <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
          {t.sshOperations || 'SSH 运维'}
        </button>
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => handleSelectDeployment({ id: contextMenu.id, name: contextMenu.name, state: contextMenu.state, template_name: contextMenu.templateName } as CustomDeployment))}>
          <svg class="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 9.776c.112-.017.227-.026.344-.026h15.812c.117 0 .232.009.344.026m-16.5 0a2.25 2.25 0 00-1.883 2.542l.857 6a2.25 2.25 0 002.227 1.932H19.05a2.25 2.25 0 002.227-1.932l.857-6a2.25 2.25 0 00-1.883-2.542m-16.5 0V6A2.25 2.25 0 016 3.75h3.879a1.5 1.5 0 011.06.44l2.122 2.12a1.5 1.5 0 001.06.44H18A2.25 2.25 0 0120.25 9v.776" /></svg>
          {t.viewOutput || '查看输出'}
        </button>
        <div class="border-t border-gray-100 my-1"></div>
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-amber-600" onclick={() => ctxAction(() => handleStop(contextMenu.id, contextMenu.name))}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
          {t.stop || '停止'}
        </button>
      {:else}
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-emerald-600" onclick={() => ctxAction(() => handleStart(contextMenu.id))}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" /></svg>
          {t.start || '启动'}
        </button>
        <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => handlePlanPreview(contextMenu.id, contextMenu.name, false))}>
          <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
          {t.planPreviewBtn || '预览'}
        </button>
      {/if}
      <div class="border-t border-gray-100 my-1"></div>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => showCloneDialog(contextMenu.id, contextMenu.name))}>
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" /></svg>
        {t.cloneCase || '克隆场景'}
      </button>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => ctxAction(() => { tagEditId = contextMenu.id; tagInput = ''; })}>
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" /></svg>
        {t.tags || '标签'}
      </button>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-gray-700" onclick={() => { const name = contextMenu.name; const tmpl = contextMenu.templateName; closeContextMenu(); localStorage.setItem('ai-chat-pending-error', JSON.stringify({ error: `部署: ${name} (${tmpl})`, templateName: tmpl, provider: '', source: 'context-menu' })); onTabChange('aiChat'); }}>
        <svg class="w-4 h-4 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" /></svg>
        {t.sendToAI || '发送到 AI'}
      </button>
      <div class="border-t border-gray-100 my-1"></div>
      <button class="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2.5 text-red-600" onclick={() => ctxAction(() => handleDelete(contextMenu.id, contextMenu.name))}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
        {t.delete || '删除'}
      </button>
    {:else}
      <div class="px-3 py-2 text-gray-400 text-[12px]">
        {contextMenu.state === 'starting' ? (t.starting || '正在启动') : contextMenu.state === 'stopping' ? (t.stopping || '正在停止') : (t.removing || '正在删除')}...
      </div>
    {/if}
  </div>
{/if}
