<script>

  import { onMount, onDestroy } from 'svelte';
  import Modal from '../UI/Modal.svelte';
  import { FetchRegistryTemplates, PullTemplate, ListTemplates, FetchTemplateReadme } from '../../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { normalizeVersion, compareVersions, hasUpdate } from '../../utils/version.js';
  import { toast } from '../../lib/toast.js';

  // Registry state
let { t, lang } = $props();
  let registryTemplates = $state([]);
  let registryLoading = $state(false);
  let registryError = $state('');
  let registrySearch = $state('');
  let pullingTemplates = $state({});
  let registryNotice = $state({ type: '', message: '' });
  let registryNoticeTimer = null;
  let templates = $state([]);

  // Readme modal state
  let readmeModal = $state({ show: false, content: '', html: '', loading: false, templateName: '' });

  // Flash feedback for successful pull/update
  let justPulled = $state({});

  // Simple markdown to HTML converter
  function parseMarkdown(md) {
    if (!md) return '';
    
    // First escape HTML (but preserve code blocks placeholder)
    const codeBlocks = [];
    let idx = 0;
    
    // Replace code blocks with placeholders to protect them
    md = md.replace(/```[\s\S]*?```/g, (match) => {
      const placeholder = `__CODEBLOCK_${idx}__`;
      codeBlocks.push(match);
      idx++;
      return placeholder;
    });
    
    // Now escape HTML in the rest
    md = md.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    
    // Step 3: Restore code blocks with escaped content
    codeBlocks.forEach((block, i) => {
      // Extract code content (remove ``` and optional language)
      let code = block.replace(/^```[a-z]*\n?/, '').replace(/```$/, '').trim();
      // Escape any remaining markdown characters in code
      code = code.replace(/^# /gm, '&#35; ').replace(/^\* /gm, '&#42; ').replace(/^- /gm, '&#45; ');
      const codeHtml = `<pre class="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto my-3 text-[12px] font-mono leading-relaxed"><code>${code}</code></pre>`;
      md = md.replace(`__CODEBLOCK_${i}__`, codeHtml);
    });
    
    // Process inline code
    md = md.replace(/`([^`]+)`/g, '<code class="bg-gray-100 px-1.5 py-0.5 rounded text-[12px] font-mono text-pink-600">$1</code>');
    
    // Process headers (only at line start)
    md = md.replace(/^#### (.*$)/gm, '<h4 class="text-sm font-semibold mt-5 mb-2 text-gray-800">$1</h4>');
    md = md.replace(/^### (.*$)/gm, '<h3 class="text-sm font-semibold mt-5 mb-2 text-gray-800">$1</h3>');
    md = md.replace(/^## (.*$)/gm, '<h2 class="text-base font-bold mt-6 mb-3 text-gray-900">$1</h2>');
    md = md.replace(/^# (.*$)/gm, '<h1 class="text-lg font-bold mt-6 mb-3 text-gray-900">$1</h1>');
    
    // Process bold and italic
    md = md.replace(/\*\*\*(.*?)\*\*\*/g, '<strong><em>$1</em></strong>');
    md = md.replace(/\*\*(.*?)\*\*/g, '<strong class="font-semibold">$1</strong>');
    md = md.replace(/\*(.*?)\*/g, '<em>$1</em>');
    md = md.replace(/___(.*?)___/g, '<strong><em>$1</em></strong>');
    md = md.replace(/__(.*?)__/g, '<strong class="font-semibold">$1</strong>');
    md = md.replace(/_(.*?)_/g, '<em>$1</em>');
    
    // Process links
    md = md.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-blue-600 hover:text-blue-800 hover:underline underline-offset-2" target="_blank" rel="noopener">$1</a>');
    
    // Process blockquotes
    md = md.replace(/^> (.*$)/gm, '<blockquote class="border-l-4 border-gray-300 pl-4 py-1 my-3 text-gray-600 italic">$1</blockquote>');

    // Process tables - extract table blocks and convert to HTML
    const tableBlocks = [];
    let tableIdx = 0;
    md = md.replace(/(^\|.+\|[ \t]*\n\|[\s:|-]+\|[ \t]*\n(\|.+\|[ \t]*\n?)*)/gm, (match) => {
      const lines = match.trim().split('\n').filter(l => l.trim());
      if (lines.length < 2) return match;

      // Parse header
      const headerCells = lines[0].split('|').filter((_, i, arr) => i > 0 && i < arr.length - 1).map(c => c.trim());

      // Parse alignment from separator row
      const separators = lines[1].split('|').filter((_, i, arr) => i > 0 && i < arr.length - 1).map(c => c.trim());
      const aligns = separators.map(s => {
        if (s.startsWith(':') && s.endsWith(':')) return 'center';
        if (s.endsWith(':')) return 'right';
        return 'left';
      });

      // Build header HTML
      const thCells = headerCells.map((cell, i) =>
        `<th class="px-3 py-2 text-left text-[11px] font-semibold text-gray-900 bg-gray-50" style="text-align:${aligns[i] || 'left'}">${cell}</th>`
      ).join('');

      // Build body rows
      const bodyRows = lines.slice(2).map(line => {
        const cells = line.split('|').filter((_, i, arr) => i > 0 && i < arr.length - 1).map(c => c.trim());
        const tds = cells.map((cell, i) =>
          `<td class="px-3 py-2 text-[12px] text-gray-700 border-t border-gray-100" style="text-align:${aligns[i] || 'left'}">${cell}</td>`
        ).join('');
        return `<tr class="hover:bg-gray-50/50">${tds}</tr>`;
      }).join('');

      const tableHtml = `<div class="my-3 overflow-x-auto rounded-lg border border-gray-200"><table class="w-full border-collapse text-[12px]"><thead><tr>${thCells}</tr></thead><tbody>${bodyRows}</tbody></table></div>`;
      const placeholder = `__TABLE_${tableIdx}__`;
      tableBlocks.push(tableHtml);
      tableIdx++;
      return placeholder;
    });

    // Process horizontal rules
    md = md.replace(/^---$/gm, '<hr class="my-6 border-gray-200">');
    md = md.replace(/^\*\*\*$/gm, '<hr class="my-6 border-gray-200">');
    
    // Process unordered lists - more specific pattern to avoid matching code
    md = md.replace(/^(\* |-)(?!\*)(.*$)/gm, '<li class="ml-4 list-disc text-gray-700">$2</li>');
    
    // Process ordered lists
    md = md.replace(/^\d+\.(?!\.)(.*$)/gm, '<li class="ml-4 list-decimal text-gray-700">$1</li>');
    
    // Remove newlines between list items to allow proper grouping
    md = md.replace(/<\/li>\n<li/g, '</li><li');
    md = md.replace(/<\/li>\s*<br>/g, '</li>');
    md = md.replace(/<br>\s*<li/g, '<li');
    
    // Wrap consecutive list items in ul/ol tags
    md = md.replace(/(<li[^>]*>[^<]*<\/li>)+/g, (match) => {
      // Clean up any remaining <br> tags
      match = match.replace(/<br\s*\/?>/g, '');
      if (match.includes('list-disc')) {
        return `<ul class="my-2">${match}</ul>`;
      } else {
        return `<ol class="my-2 list-inside">${match}</ol>`;
      }
    });
    
    // Process paragraphs - split by double newlines
    const paragraphs = md.split(/\n\n+/);
    let result = paragraphs.map(p => {
      p = p.trim();
      if (!p) return '';
      // Skip if already wrapped in HTML tags (including lists) or is a placeholder
      if (p.match(/^<(h[1-4]|ul|ol|pre|blockquote|hr|div)/i)) return p;
      if (p.match(/^__TABLE_\d+__$/)) return p;
      // Wrap in paragraph
      return `<p class="my-2 text-gray-700 leading-relaxed">${p.replace(/\n/g, '<br>')}</p>`;
    }).join('\n');

    // Restore table placeholders
    tableBlocks.forEach((html, i) => {
      result = result.replace(`__TABLE_${i}__`, html);
    });

    return result;
  }

  // Filter tab state
  let filterTab = $state('all'); // 'all' | 'installed' | 'notInstalled' | 'updatable'

  // Batch operation state
  let selectedTemplates = $state(new Set());
  let batchOperating = $state(false);
  let batchPullConfirm = $state({ show: false, count: 0 });
  let batchUpdateConfirm = $state({ show: false, count: 0 });

  // Stats
  let installedCount = $derived(registryTemplates.filter(t => t.installed).length);
  let updatableCount = $derived(registryTemplates.filter(t => t.installed && hasUpdate(t)).length);
  let notInstalledCount = $derived(registryTemplates.filter(t => !t.installed).length);

  let filteredRegistryTemplates = $derived(registryTemplates
    .filter(t => {
      // Tab filter
      if (filterTab === 'installed' && !t.installed) return false;
      if (filterTab === 'notInstalled' && t.installed) return false;
      if (filterTab === 'updatable' && !(t.installed && hasUpdate(t))) return false;
      // Search filter
      if (!registrySearch) return true;
      const q = registrySearch.toLowerCase();
      return t.name.toLowerCase().includes(q) ||
        (t.author && t.author.toLowerCase().includes(q)) ||
        (t.description && t.description.toLowerCase().includes(q)) ||
        (t.description_en && t.description_en.toLowerCase().includes(q)) ||
        (t.tags && t.tags.some(tag => tag.toLowerCase().includes(q)));
    })
    .sort((a, b) => {
      if (a.installed && !b.installed) return -1;
      if (!a.installed && b.installed) return 1;
      return a.name.localeCompare(b.name);
    }));

  let allSelected = $derived(filteredRegistryTemplates.length > 0 && selectedTemplates.size === filteredRegistryTemplates.length);

  let someSelected = $derived(selectedTemplates.size > 0 && selectedTemplates.size < filteredRegistryTemplates.length);

  let hasSelection = $derived(selectedTemplates.size > 0);

  let canPullTemplates = $derived(Array.from(selectedTemplates).filter(name => {
    const tmpl = registryTemplates.find(t => t.name === name);
    return tmpl && !tmpl.installed;
  }));

  let canUpdateTemplates = $derived(Array.from(selectedTemplates).filter(name => {
    const tmpl = registryTemplates.find(t => t.name === name);
    return tmpl && tmpl.installed && hasUpdate(tmpl);
  }));

  // All updatable templates (for "update all" shortcut)
  let allUpdatableTemplates = $derived(registryTemplates.filter(t => t.installed && hasUpdate(t)).map(t => t.name));


  function setRegistryNotice(type, message, autoClear = true) {
    registryNotice = { type, message };
    if (registryNoticeTimer) {
      clearTimeout(registryNoticeTimer);
      registryNoticeTimer = null;
    }
    if (autoClear && message) {
      registryNoticeTimer = setTimeout(() => {
        registryNotice = { type: '', message: '' };
      }, 3000);
    }
  }

  async function loadRegistryTemplates() {
    registryLoading = true;
    registryError = '';
    try {
      registryTemplates = await FetchRegistryTemplates('');
    } catch (e) {
      registryError = e.message || String(e);
      registryTemplates = [];
    } finally {
      registryLoading = false;
    }
  }

  async function syncLocalTemplates() {
    try {
      const list = await ListTemplates();
      templates = list || [];
    } catch (e) {
      console.error('Failed to sync local templates:', e);
    }
  }

  async function handlePullTemplate(templateName, force = false) {
    pullingTemplates[templateName] = true;
    pullingTemplates = pullingTemplates;
    setRegistryNotice('info', `${t.pulling} ${templateName}`, false);
    try {
      await PullTemplate(templateName, force);
      // Refresh registry and local templates after successful pull
      await loadRegistryTemplates();
      await syncLocalTemplates();
      registryTemplates = (registryTemplates || []).map((tmpl) => {
        if (tmpl.name !== templateName) return tmpl;
        const latest = tmpl.latest || tmpl.localVersion;
        return {
          ...tmpl,
          installed: true,
          localVersion: latest || tmpl.localVersion
        };
      });
      // Flash feedback
      justPulled[templateName] = true;
      justPulled = justPulled;
      setTimeout(() => { delete justPulled[templateName]; justPulled = justPulled; }, 2000);
      toast.success(`${t.pullSuccess}: ${templateName}`);
      setRegistryNotice('success', `${t.pullSuccess}: ${templateName}`);
    } catch (e) {
      toast.error(`${t.pullFailed}: ${templateName}`);
      setRegistryNotice('error', `${t.pullFailed}: ${templateName}`);
    } finally {
      pullingTemplates[templateName] = false;
      pullingTemplates = pullingTemplates;
    }
  }

  async function handleShowReadme(templateName) {
    readmeModal = { show: true, content: '', html: '', loading: true, templateName };
    try {
      const content = await FetchTemplateReadme(templateName, lang || 'zh');
      const html = parseMarkdown(content);
      readmeModal = { ...readmeModal, content, html, loading: false };
    } catch (e) {
      readmeModal = { ...readmeModal, content: e.message || String(e), html: `<p class="text-red-500">${e.message || String(e)}</p>`, loading: false };
    }
  }

  function closeReadmeModal() {
    readmeModal = { show: false, content: '', html: '', loading: false, templateName: '' };
  }

  // Listen for refresh events to update pulling status
  $effect(() => {
    if (registryTemplates.length > 0) {
      // Reset pulling status when templates are refreshed
      for (const t of registryTemplates) {
        if (t.installed && pullingTemplates[t.name]) {
          pullingTemplates[t.name] = false;
        }
      }
    }
  });

  // ============================================================================
  // Batch Operation Functions
  // ============================================================================

  function toggleSelectAll() {
    if (allSelected) {
      selectedTemplates = new Set();
    } else {
      selectedTemplates = new Set(filteredRegistryTemplates.map(t => t.name));
    }
  }

  function toggleSelectTemplate(templateName) {
    const newSet = new Set(selectedTemplates);
    if (newSet.has(templateName)) {
      newSet.delete(templateName);
    } else {
      newSet.add(templateName);
    }
    selectedTemplates = newSet;
  }

  function showBatchPullConfirm() {
    batchPullConfirm = { show: true, count: canPullTemplates.length };
  }

  function cancelBatchPull() {
    batchPullConfirm = { show: false, count: 0 };
  }

  async function confirmBatchPull() {
    batchPullConfirm = { show: false, count: 0 };
    batchOperating = true;
    const targets = _pullAllMode
      ? registryTemplates.filter(tmpl => !tmpl.installed).map(tmpl => tmpl.name)
      : canPullTemplates;
    _pullAllMode = false;

    try {
      await Promise.all(targets.map(name => handlePullTemplate(name, false)));
      selectedTemplates = new Set();
    } catch (e) {
      setRegistryNotice('error', e.message || String(e));
    } finally {
      batchOperating = false;
      await loadRegistryTemplates();
    }
  }

  function showBatchUpdateConfirm() {
    batchUpdateConfirm = { show: true, count: canUpdateTemplates.length };
  }

  function cancelBatchUpdate() {
    batchUpdateConfirm = { show: false, count: 0 };
  }

  async function confirmBatchUpdate() {
    batchUpdateConfirm = { show: false, count: 0 };
    batchOperating = true;

    try {
      await Promise.all(canUpdateTemplates.map(name => handlePullTemplate(name, true)));
      selectedTemplates = new Set();
    } catch (e) {
      setRegistryNotice('error', e.message || String(e));
    } finally {
      batchOperating = false;
      await loadRegistryTemplates();
    }
  }

  async function handleUpdateAll() {
    if (allUpdatableTemplates.length === 0) return;
    batchUpdateConfirm = { show: true, count: allUpdatableTemplates.length };
    // Override confirmBatchUpdate to use allUpdatableTemplates
    _updateAllMode = true;
  }

  let _updateAllMode = $state(false);

  async function doConfirmBatchUpdate() {
    batchUpdateConfirm = { show: false, count: 0 };
    batchOperating = true;
    const targets = _updateAllMode ? allUpdatableTemplates : canUpdateTemplates;
    _updateAllMode = false;

    try {
      await Promise.all(targets.map(name => handlePullTemplate(name, true)));
      selectedTemplates = new Set();
    } catch (e) {
      setRegistryNotice('error', e.message || String(e));
    } finally {
      batchOperating = false;
      await loadRegistryTemplates();
    }
  }

  onMount(() => {
    loadRegistryTemplates();
  });

  onDestroy(() => {
    if (registryNoticeTimer) {
      clearTimeout(registryNoticeTimer);
      registryNoticeTimer = null;
    }
  });

  // Scenario type label from tags
  const tagLabelMap = $derived({
    'c2': { label: 'C2', color: 'bg-red-50 text-red-600' },
    'proxy': { label: t.tagProxy || '代理', color: 'bg-purple-50 text-purple-600' },
    'tunnel': { label: t.tagTunnel || '隧道', color: 'bg-purple-50 text-purple-600' },
    'phishing': { label: t.tagPhishing || '钓鱼', color: 'bg-orange-50 text-orange-600' },
    'range': { label: t.tagRange || '靶场', color: 'bg-cyan-50 text-cyan-600' },
    'recon': { label: t.tagRecon || '侦查', color: 'bg-blue-50 text-blue-600' },
    'scan': { label: t.tagScan || '扫描', color: 'bg-blue-50 text-blue-600' },
    'mail': { label: t.tagMail || '邮件', color: 'bg-amber-50 text-amber-600' },
    'vpn': { label: 'VPN', color: 'bg-green-50 text-green-600' },
    'docker': { label: 'Docker', color: 'bg-sky-50 text-sky-600' },
    'ddos': { label: 'DDoS', color: 'bg-rose-50 text-rose-600' },
    'dns': { label: 'DNS', color: 'bg-indigo-50 text-indigo-600' },
    'infra': { label: t.tagInfra || '基础设施', color: 'bg-gray-100 text-gray-600' },
    'base': { label: t.tagBase || '基础', color: 'bg-gray-100 text-gray-600' },
    'collaborate': { label: t.tagCollaborate || '协作', color: 'bg-teal-50 text-teal-600' },
    'file': { label: t.tagFile || '文件', color: 'bg-lime-50 text-lime-600' },
  });

  function scenarioLabels(tags) {
    if (!tags || tags.length === 0) return [];
    const results = [];
    for (const tag of tags) {
      const key = tag.toLowerCase();
      if (tagLabelMap[key]) {
        results.push(tagLabelMap[key]);
      }
    }
    return results;
  }

  // Handle clicks inside README modal to open external links
  function handleReadmeClick(e) {
    const link = e.target.closest('a[href]');
    if (link) {
      e.preventDefault();
      BrowserOpenURL(link.href);
    }
  }

  // Batch pull all not-installed templates
  async function pullAllNotInstalled() {
    const targets = registryTemplates.filter(t => !t.installed).map(t => t.name);
    if (targets.length === 0) return;
    batchPullConfirm = { show: true, count: targets.length };
    _pullAllMode = true;
  }

  let _pullAllMode = $state(false);

  // Export refresh function for parent component
  export function refresh() {
    loadRegistryTemplates();
  }

</script>

<div class="space-y-4">
  <!-- Toolbar: search + actions -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center gap-3">
    <div class="flex-1 relative w-full sm:w-auto">
      <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input 
        type="text" 
        placeholder={t.search}
        class="w-full h-9 pl-10 pr-8 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
        bind:value={registrySearch} 
      />
      {#if registrySearch}
        <button onclick={() => registrySearch = ''} class="absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 rounded-full bg-gray-200 hover:bg-gray-300 flex items-center justify-center transition-colors cursor-pointer">
          <svg class="w-2.5 h-2.5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      {#if updatableCount > 0}
        <button 
          class="h-9 px-3.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
          onclick={handleUpdateAll}
          disabled={batchOperating}
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
          {t.updateAll || '全部更新'} ({updatableCount})
        </button>
      {/if}
      <button 
        class="h-9 px-3.5 text-[12px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
        onclick={toggleSelectAll}
        disabled={registryLoading || filteredRegistryTemplates.length === 0}
      >
        {allSelected ? t.clearSelection : t.selectAll || '全选'}
      </button>
      <button 
        class="h-9 px-3.5 text-[12px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
        onclick={loadRegistryTemplates}
        disabled={registryLoading}
      >
        <svg class="w-3.5 h-3.5 {registryLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
        {registryLoading ? t.loading : t.refreshRepo}
      </button>
    </div>
  </div>

  <!-- Filter tabs -->
  <div class="flex items-center gap-4">
    <div class="flex items-center gap-1 bg-gray-100 rounded-lg p-0.5">
      <button 
        class="px-3 py-1.5 text-[12px] font-medium rounded-md transition-colors cursor-pointer {filterTab === 'all' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
        onclick={() => { filterTab = 'all'; selectedTemplates = new Set(); }}
      >{t.filterAll || '全部'} ({registryTemplates.length})</button>
      <button 
        class="px-3 py-1.5 text-[12px] font-medium rounded-md transition-colors cursor-pointer {filterTab === 'installed' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
        onclick={() => { filterTab = 'installed'; selectedTemplates = new Set(); }}
      >{t.installed} ({installedCount})</button>
      <button 
        class="px-3 py-1.5 text-[12px] font-medium rounded-md transition-colors cursor-pointer {filterTab === 'notInstalled' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
        onclick={() => { filterTab = 'notInstalled'; selectedTemplates = new Set(); }}
      >{t.notInstalled || '未安装'} ({notInstalledCount})</button>
      {#if updatableCount > 0}
        <button 
          class="px-3 py-1.5 text-[12px] font-medium rounded-md transition-colors cursor-pointer {filterTab === 'updatable' ? 'bg-white text-gray-900 shadow-sm' : 'text-amber-600 hover:text-amber-700'}"
          onclick={() => { filterTab = 'updatable'; selectedTemplates = new Set(); }}
        >{t.updatable || '可更新'} ({updatableCount})</button>
      {/if}
    </div>
  </div>

  <!-- Notice -->
  {#if registryNotice.message}
    <div class="flex items-center gap-2 rounded-lg border px-3 py-2 text-[12px]
      {registryNotice.type === 'success' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : registryNotice.type === 'error' ? 'bg-red-50 border-red-100 text-red-700' : 'bg-amber-50 border-amber-100 text-amber-700'}">
      {#if registryNotice.type === 'info'}
        <div class="w-3.5 h-3.5 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></div>
      {/if}
      <span class="flex-1 truncate">{registryNotice.message}</span>
      <button class="text-gray-400 hover:text-gray-600 transition-colors cursor-pointer" onclick={() => setRegistryNotice('', '')} aria-label={t.close}>
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  {#if registryError}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{registryError}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => registryError = ''} aria-label={t.close}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  {#if registryLoading}
    <div class="flex items-center justify-center h-64">
      <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Batch Operations Bar -->
    {#if hasSelection}
      <div class="flex items-center justify-between bg-gray-50 rounded-lg px-4 py-2.5">
        <div class="flex items-center gap-3">
          <span class="text-[12px] font-medium text-gray-700">
            {t.selected} {selectedTemplates.size} {t.items}
          </span>
          <button
            class="text-[11px] text-gray-500 hover:text-gray-700 underline cursor-pointer"
            onclick={() => { selectedTemplates = new Set(); }}
          >{t.clearSelection}</button>
        </div>
        <div class="flex items-center gap-2">
          {#if canPullTemplates.length > 0}
            <button
              class="px-3 h-7 text-[11px] font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer"
              onclick={showBatchPullConfirm}
              disabled={batchOperating}
            >{t.batchPull} ({canPullTemplates.length})</button>
          {/if}
          {#if canUpdateTemplates.length > 0}
            <button
              class="px-3 h-7 text-[11px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors disabled:opacity-50 cursor-pointer"
              onclick={showBatchUpdateConfirm}
              disabled={batchOperating}
            >{t.batchUpdate} ({canUpdateTemplates.length})</button>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Template Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      {#each filteredRegistryTemplates as tmpl}
        <div class="bg-white rounded-xl border {justPulled[tmpl.name] ? 'border-emerald-400 ring-2 ring-emerald-200' : tmpl.installed ? 'border-emerald-200 bg-emerald-50/30' : 'border-gray-100'} p-4 hover:shadow-md transition-all duration-500">
          <div class="flex gap-3">
            <!-- Checkbox -->
            <div class="pt-0.5 flex-shrink-0">
              <input
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
                checked={selectedTemplates.has(tmpl.name)}
                onchange={() => toggleSelectTemplate(tmpl.name)}
                onclick={(e) => e.stopPropagation()}
              />
            </div>
            <!-- Content -->
            <div class="flex-1 min-w-0">
              <div class="flex items-start justify-between mb-2">
                <div class="flex-1 min-w-0">
                  <h3 class="text-[13px] font-semibold text-gray-900 truncate">{tmpl.name}</h3>
                  <div class="flex items-center gap-1.5 mt-0.5">
                    <p class="text-[11px] text-gray-400">v{tmpl.latest}</p>
                    {#each scenarioLabels(tmpl.tags) as st}
                      <span class="px-1.5 py-0 text-[9px] font-medium rounded {st.color}">{st.label}</span>
                    {/each}
                  </div>
                </div>
                {#if tmpl.installed}
                  {#if hasUpdate(tmpl)}
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-amber-50 text-amber-600 text-[10px] font-medium rounded-full flex-shrink-0">
                      <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                      {t.updatable || '可更新'}
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[10px] font-medium rounded-full flex-shrink-0">
                      <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
                      {t.installed}
                    </span>
                  {/if}
                {/if}
              </div>
              
              {#if tmpl.description}
                <p class="text-[11px] text-gray-500 mb-2 line-clamp-2">{lang === 'en' ? (tmpl.description_en || tmpl.description) : tmpl.description}</p>
              {/if}
              
              {#if tmpl.tags && tmpl.tags.length > 0}
                <div class="flex flex-wrap gap-1 mb-2">
                  {#each tmpl.tags.slice(0, 3) as tag}
                    <span class="px-1.5 py-0.5 bg-gray-100 text-gray-500 text-[10px] rounded">{tag}</span>
                  {/each}
                  {#if tmpl.tags.length > 3}
                    <span class="px-1.5 py-0.5 bg-gray-100 text-gray-400 text-[10px] rounded">+{tmpl.tags.length - 3}</span>
                  {/if}
                </div>
              {/if}
              
              <div class="flex items-center justify-between pt-2 border-t border-gray-100">
                <div class="text-[10px] text-gray-400">
                  {#if tmpl.author}by {tmpl.author}{/if}
                  {#if tmpl.installed && hasUpdate(tmpl)}
                    <span class="ml-1.5 text-amber-500 font-medium">v{tmpl.localVersion} → v{tmpl.latest}</span>
                  {/if}
                </div>
                {#if pullingTemplates[tmpl.name]}
                  <span class="inline-flex items-center gap-1.5 text-[11px] text-amber-600">
                    <span class="w-3 h-3 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></span>
                    {t.pulling}
                  </span>
                {:else}
                  <div class="flex gap-1.5">
                    <button 
                      class="px-2.5 h-7 text-[11px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors cursor-pointer"
                      onclick={() => handleShowReadme(tmpl.name)}
                    >{t.viewReadme || '查看'}</button>
                    {#if tmpl.installed && hasUpdate(tmpl)}
                      <button 
                        class="px-2.5 h-7 text-[11px] font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 transition-colors cursor-pointer"
                        onclick={() => handlePullTemplate(tmpl.name, true)}
                      >{t.update}</button>
                    {:else if !tmpl.installed}
                      <button 
                        class="px-2.5 h-7 text-[11px] font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 transition-colors cursor-pointer"
                        onclick={() => handlePullTemplate(tmpl.name, false)}
                      >{t.pull}</button>
                    {/if}
                  </div>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div class="col-span-full py-16 text-center">
          <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          {#if registrySearch}
            <p class="text-[13px] text-gray-500 mb-2">{t.noMatch}</p>
            <button class="text-[12px] text-gray-500 hover:text-gray-700 underline cursor-pointer" onclick={() => registrySearch = ''}>{t.clearSearch || '清除搜索'}</button>
          {:else if filterTab === 'not-installed'}
            {#if registryTemplates.filter(tmpl => !tmpl.installed).length === 0}
              <p class="text-[13px] text-gray-500 mb-3">{t.allInstalled || '所有模板已安装'}</p>
            {:else}
              <p class="text-[13px] text-gray-500 mb-3">{t.notInstalledHint || '尚有模板未拉取'}</p>
              <button 
                class="h-8 px-4 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors cursor-pointer"
                onclick={pullAllNotInstalled}
              >{t.pullAll || '一键全部拉取'} ({registryTemplates.filter(tmpl => !tmpl.installed).length})</button>
            {/if}
          {:else if filterTab !== 'all'}
            <p class="text-[13px] text-gray-500 mb-2">{t.noMatchFilter || '当前筛选无结果'}</p>
            <button class="text-[12px] text-gray-500 hover:text-gray-700 underline cursor-pointer" onclick={() => filterTab = 'all'}>{t.showAll || '显示全部'}</button>
          {:else}
            <p class="text-[13px] text-gray-500 mb-2">{t.clickRefresh}</p>
            <button class="mt-2 h-8 px-4 text-[12px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer" onclick={loadRegistryTemplates}>{t.refreshRepo}</button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Batch Pull Confirmation Modal -->
<Modal show={batchPullConfirm.show} onclose={cancelBatchPull} class="overflow-visible">
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-gray-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchPull}</h3>
            <p class="text-[13px] text-gray-500">{t.pulling}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchPullMessage} <span class="font-medium text-gray-900">{batchPullConfirm.count}</span> {t.templates}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelBatchPull}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
          onclick={confirmBatchPull}
        >{t.pull}</button>
      </div>
    </div>
</Modal>

<!-- Batch Update Confirmation Modal -->
<Modal show={batchUpdateConfirm.show} onclose={cancelBatchUpdate} class="overflow-visible">
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-gray-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchUpdate}</h3>
            <p class="text-[13px] text-gray-500">{t.update}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchUpdateMessage} <span class="font-medium text-gray-900">{batchUpdateConfirm.count}</span> {t.templates}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelBatchUpdate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
          onclick={doConfirmBatchUpdate}
        >{t.update}</button>
      </div>
    </div>
</Modal>

<Modal show={readmeModal.show} onclose={closeReadmeModal} class="overflow-visible">
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-3xl w-full mx-4 max-h-[80vh] flex flex-col" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h2 id="readme-modal-title" class="text-[15px] font-medium text-gray-900">{t.readme || 'README'}</h2>
          <p class="text-[12px] text-gray-500">{readmeModal.templateName}</p>
        </div>
        <button class="text-gray-400 hover:text-gray-600 transition-colors cursor-pointer" onclick={closeReadmeModal} aria-label={t.close || '关闭'}>
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      <div class="px-6 py-4 overflow-auto flex-1">
        {#if readmeModal.loading}
          <div class="flex items-center justify-center py-8">
            <svg class="animate-spin h-6 w-6 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
        {:else}
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <div class="text-[13px] text-gray-700" onclick={handleReadmeClick}>
            {@html readmeModal.html || readmeModal.content}
          </div>
        {/if}
      </div>
    </div>
</Modal>
