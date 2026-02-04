<script>
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../wailsjs/runtime/runtime.js';
  import { ListCases, ListTemplates, StartCase, StopCase, RemoveCase, CreateCase, CreateAndRunCase, GetConfig, GetCaseOutputs, GetTemplateVariables, SaveProxyConfig, FetchRegistryTemplates, PullTemplate, RemoveTemplate, CopyTemplate, GetTemplateFiles, SaveTemplateFiles, GetMCPStatus, StartMCPServer, StopMCPServer, GetProvidersConfig, SaveProvidersConfig, SetDebugLogging, ListProfiles, GetActiveProfile, SetActiveProfile, CreateProfile, UpdateProfile, DeleteProfile, GetResourceSummary, GetBalances, ComposePreview, ComposeUp, ComposeDown, GetTerraformMirrorConfig, SaveTerraformMirrorConfig, TestTerraformEndpoints } from '../wailsjs/go/main/App.js';

  let cases = [];
  let templates = [];
  let logs = [];
  let config = { redcPath: '', projectPath: '', logPath: '', httpProxy: '', httpsProxy: '', noProxy: '', debugEnabled: false };
  let activeTab = 'dashboard';
  let selectedTemplate = '';
  let newCaseName = '';
  let isLoading = false;
  let error = '';
  let expandedCase = null;
  let caseOutputs = {};
  let deleteConfirm = { show: false, caseId: null, caseName: '' };
  let templateVariables = [];
  let variableValues = {};
  let proxyForm = { httpProxy: '', httpsProxy: '', noProxy: '' };
  let proxySaving = false;
  let debugEnabled = false;
  let debugSaving = false;
  let terraformMirror = { enabled: false, configPath: '', managed: false, fromEnv: false, providers: [] };
  let terraformMirrorForm = { enabled: false, configPath: '', setEnv: false, providers: { aliyun: true, tencent: false, volc: false } };
  let terraformMirrorSaving = false;
  let terraformMirrorError = '';
  let terraformInitHint = { show: false, message: '', detail: '' };
  let networkChecks = [];
  let networkCheckLoading = false;
  let networkCheckError = '';
  
  // Registry state
  let registryTemplates = [];
  let registryLoading = false;
  let registryError = '';
  let registrySearch = '';
  let pullingTemplates = {};
  let registryNotice = { type: '', message: '' };
  let registryNoticeTimer = null;

  // MCP state
  let mcpStatus = { running: false, mode: '', address: '', protocolVersion: '' };
  let mcpForm = { mode: 'sse', address: 'localhost:8080' };
  let mcpLoading = false;

  // Credentials state
  let providersConfig = { configPath: '', providers: [] };
  let credentialsLoading = false;
  let credentialsSaving = {};
  let editingProvider = null;
  let editFields = {};
  let customConfigPath = '';
  let profiles = [];
  let activeProfileId = '';
  let profileForm = { name: '', configPath: '', templateDir: '' };
  let profileLoading = false;
  let profileSaving = false;
  let profileError = '';

  // Local templates state
  let localTemplates = [];
  let localTemplatesLoading = false;
  let localTemplatesSearch = '';
  let localTemplateDetail = null;
  let localTemplateVars = [];
  let localTemplateVarsLoading = false;
  let deleteTemplateConfirm = { show: false, name: '' };
  let deletingTemplate = {};
  let cloneTemplateModal = { show: false, source: '', target: '' };
  let templateEditor = { show: false, name: '', files: {}, active: '', saving: false, error: '' };

  // Resources state
  let resourceSummary = [];
  let resourcesLoading = false;
  let resourcesError = '';
  let balanceResults = [];
  let balanceLoading = false;
  let balanceError = '';
  let balanceCooldown = 0;
  let balanceCooldownTimer = null;

  // Compose state
  let composeFilePath = 'redc-compose.yaml';
  let composeProfiles = '';
  let composeSummary = null;
  let composeLoading = false;
  let composeActionLoading = false;
  let composeError = '';

  // Create status state
  let createStatus = 'idle';
  let createStatusMessage = '';
  let createStatusDetail = '';
  let createStatusTimer = null;

  // i18n state
  let lang = localStorage.getItem('lang') || 'zh';
  const i18n = {
    zh: {
      dashboard: '仪表盘', console: '控制台', settings: '设置', credentials: '凭据管理', registry: '模板仓库', ai: 'AI 集成', localTemplates: '本地模板',
      sceneManage: '场景管理', templateRepo: '模板仓库', aiIntegration: 'AI 集成', localTmplManage: '本地模板',
      template: '模板', selectTemplate: '选择模板...', name: '名称', optional: '可选',
      create: '创建', createAndRun: '创建并运行', templateParams: '模板参数',
      creating: '正在创建中...', initializing: '初始化中...', createSuccess: '创建成功', createFailed: '创建失败',
      id: 'ID', type: '类型', state: '状态', time: '时间', actions: '操作',
      start: '启动', stop: '停止', delete: '删除', noScene: '暂无场景',
      running: '运行中', stopped: '已停止', error: '异常', created: '已创建',
      pending: '等待中', starting: '启动中', stopping: '停止中', removing: '删除中',
      processing: '处理中', loadingOutputs: '正在加载输出信息...', noOutput: '场景未运行，无输出信息',
      copied: '已复制', copy: '复制', terminal: 'Terminal', clear: '清空', waitOutput: '等待输出...',
      redcPath: 'RedC 路径', projectPath: '项目路径', logPath: '日志路径',
      proxyConfig: '代理配置', httpProxy: 'HTTP 代理', httpsProxy: 'HTTPS 代理', noProxyLabel: '不使用代理的地址 (NO_PROXY)',
      saving: '保存中...', saveProxy: '保存代理配置', proxyHint: '配置后将用于 Terraform 的网络请求',
      search: '搜索模板...', loading: '加载中...', refreshRepo: '刷新仓库', installed: '已安装',
      update: '更新', pull: '拉取', pulling: '拉取中...', noMatch: '未找到匹配的模板', clickRefresh: '点击"刷新仓库"加载模板列表',
      pullSuccess: '拉取成功', pullFailed: '拉取失败',
      mcpServer: 'MCP 服务器', mcpDesc: 'Model Context Protocol 服务',
      resources: '云资源', compose: '编排管理',
      resourceSummary: '资源汇总', resourceType: '资源类型', resourceCount: '数量',
      balancePlaceholder: '暂无余额数据，请点击“查询余额”',
      balanceQuery: '查询余额', balanceCooldown: '冷却中', balanceProvider: '云厂商', balanceAmount: '余额', balanceCurrency: '币种', balanceUpdatedAt: '更新时间',
      composeFile: 'Compose 文件', composeProfiles: 'Profiles（逗号分隔）', previewCompose: '预览编排',
      composeUp: '启动编排', composeDown: '销毁编排', composePreview: '编排预览',
      serviceName: '服务', serviceTemplate: '模板', serviceProvider: '云厂商', serviceDepends: '依赖', serviceReplicas: '副本',
      transportMode: '传输模式', listenAddr: '监听地址', protocolVersion: '协议版本', msgEndpoint: '消息端点',
      stopServer: '停止服务器', startServer: '启动服务器', stoppingServer: '停止中...', startingServer: '启动中...',
      aboutMcp: '关于 MCP', mcpInfo: 'Model Context Protocol (MCP) 是一种开放协议，允许 AI 助手与外部工具和数据源进行交互。启用 MCP 服务器后，您可以通过 Claude、Cursor 等支持 MCP 的 AI 工具直接管理 RedC 基础设施。',
      availableTools: '可用工具',
      stdioHint: 'STDIO 通过 stdin/stdout 交换数据，无需在 GUI 启动或停止。',
      configPath: '配置文件路径', defaultPath: '留空使用默认路径 ~/redc/config.yaml', loadConfig: '加载配置',
      currentConfig: '当前配置', securityTip: '安全提示：', securityInfo: '凭据以脱敏形式显示，编辑时需重新输入完整值。空字段不会覆盖已有配置。',
      edit: '编辑', cancel: '取消', save: '保存', notSet: '未设置', enterNew: '输入新值覆盖', clickLoad: '点击"加载配置"查看凭据',
      confirmDelete: '确认删除', cannotUndo: '此操作不可撤销', confirmDeleteScene: '确定要删除场景', region: '区域', credentialsJson: '凭据 JSON',
      profile: '凭据配置', profileName: 'Profile 名称', templateDir: '模板目录', profileManage: 'Profile 管理', activeProfile: '当前 Profile',
      createProfile: '新建 Profile', saveProfile: '保存 Profile', deleteProfile: '删除 Profile', selectProfile: '选择 Profile...',
      profileHint: 'Profile 将对应独立配置文件与模板目录',
      profileNameRequired: 'Profile 名称不能为空',
      profileCredentialsFrom: '凭据读取自', profileTemplateFrom: '模板目录来自', profileSwitchHint: '切换 Profile 会同时切换凭据与模板目录',
      selectTemplateErr: '请选择一个模板',
      // Local templates i18n
      version: '版本', author: '作者', module: '模块', description: '描述', viewParams: '查看参数',
      cloneTemplate: '复制模板',
      editTemplate: '编辑模板', templateFiles: '模板文件', saveTemplate: '保存模板',
      cloneTitle: '复制模板', cloneName: '新模板名称', cloneHint: '复制后可独立编辑',
      noLocalTemplates: '暂无本地模板', goToRegistry: '前往模板仓库拉取',
      confirmDeleteTemplate: '确定要删除模板', deleteWarning: '删除后需要重新从仓库拉取才能使用',
      deleting: '删除中...', refresh: '刷新', close: '关闭',
      paramName: '参数名', paramType: '类型', paramDesc: '描述', paramDefault: '默认值', paramRequired: '必填',
      noParams: '该模板没有可配置参数', loadingParams: '正在加载参数...',
      debugLogs: '调试日志', debugLogsDesc: '启用后控制台输出更详细的日志信息', enable: '开启', disable: '关闭',
      terraformMirror: 'Terraform 镜像加速', mirrorEnabled: '启用镜像', mirrorConfigPath: '配置文件路径',
      mirrorConfigHint: '留空使用默认路径', mirrorSetEnv: '设置 TF_CLI_CONFIG_FILE', mirrorSave: '保存镜像配置',
      mirrorAliyunPreset: '一键使用阿里云镜像', mirrorTencentPreset: '一键使用腾讯云镜像', mirrorVolcPreset: '一键使用火山云镜像',
      mirrorDetected: '检测到 Terraform 初始化网络异常', mirrorDetectedDesc: '可尝试启用国内镜像加速',
      mirrorApplyAliyun: '一键启用阿里云镜像', mirrorGoSettings: '前往设置', mirrorConfigFromEnv: '当前路径来自 TF_CLI_CONFIG_FILE',
      mirrorProviders: '适配云厂商', mirrorAliyun: '阿里云', mirrorTencent: '腾讯云', mirrorVolc: '火山云',
      networkCheck: '网络诊断', networkCheckBtn: '检测 Terraform 连接', networkChecking: '检测中...',
      networkEndpoint: '端点', networkStatus: '状态', networkLatency: '延迟', networkError: '错误',
    },
    en: {
      dashboard: 'Dashboard', console: 'Console', settings: 'Settings', credentials: 'Credentials', registry: 'Template Registry', ai: 'AI Integration', localTemplates: 'Local Templates',
      sceneManage: 'Scene Management', templateRepo: 'Template Registry', aiIntegration: 'AI Integration', localTmplManage: 'Local Templates',
      template: 'Template', selectTemplate: 'Select template...', name: 'Name', optional: 'Optional',
      create: 'Create', createAndRun: 'Create & Run', templateParams: 'Template Parameters',
      creating: 'Creating...', initializing: 'Initializing...', createSuccess: 'Created', createFailed: 'Create failed',
      id: 'ID', type: 'Type', state: 'State', time: 'Time', actions: 'Actions',
      start: 'Start', stop: 'Stop', delete: 'Delete', noScene: 'No scenes',
      running: 'Running', stopped: 'Stopped', error: 'Error', created: 'Created',
      pending: 'Pending', starting: 'Starting', stopping: 'Stopping', removing: 'Removing',
      processing: 'Processing', loadingOutputs: 'Loading outputs...', noOutput: 'Scene not running, no outputs',
      copied: 'Copied', copy: 'Copy', terminal: 'Terminal', clear: 'Clear', waitOutput: 'Waiting for output...',
      redcPath: 'RedC Path', projectPath: 'Project Path', logPath: 'Log Path',
      proxyConfig: 'Proxy Configuration', httpProxy: 'HTTP Proxy', httpsProxy: 'HTTPS Proxy', noProxyLabel: 'No Proxy Addresses (NO_PROXY)',
      saving: 'Saving...', saveProxy: 'Save Proxy Config', proxyHint: 'Used for Terraform network requests',
      search: 'Search templates...', loading: 'Loading...', refreshRepo: 'Refresh Registry', installed: 'Installed',
      update: 'Update', pull: 'Pull', pulling: 'Pulling...', noMatch: 'No matching templates', clickRefresh: 'Click "Refresh Registry" to load templates',
      pullSuccess: 'Pull success', pullFailed: 'Pull failed',
      mcpServer: 'MCP Server', mcpDesc: 'Model Context Protocol Service',
      resources: 'Cloud Resources', compose: 'Compose',
      resourceSummary: 'Resource Summary', resourceType: 'Resource Type', resourceCount: 'Count',
      balancePlaceholder: 'No balance data yet. Click "Query Balance"',
      balanceQuery: 'Query Balance', balanceCooldown: 'Cooling down', balanceProvider: 'Provider', balanceAmount: 'Balance', balanceCurrency: 'Currency', balanceUpdatedAt: 'Updated At',
      composeFile: 'Compose File', composeProfiles: 'Profiles (comma-separated)', previewCompose: 'Preview Compose',
      composeUp: 'Compose Up', composeDown: 'Compose Down', composePreview: 'Compose Preview',
      serviceName: 'Service', serviceTemplate: 'Template', serviceProvider: 'Provider', serviceDepends: 'Depends', serviceReplicas: 'Replicas',
      transportMode: 'Transport Mode', listenAddr: 'Listen Address', protocolVersion: 'Protocol Version', msgEndpoint: 'Message Endpoint',
      stopServer: 'Stop Server', startServer: 'Start Server', stoppingServer: 'Stopping...', startingServer: 'Starting...',
      aboutMcp: 'About MCP', mcpInfo: 'Model Context Protocol (MCP) is an open protocol that allows AI assistants to interact with external tools and data sources. With MCP server enabled, you can manage RedC infrastructure directly via Claude, Cursor and other MCP-compatible AI tools.',
      availableTools: 'Available Tools',
      stdioHint: 'STDIO uses stdin/stdout for data exchange. No GUI start/stop required.',
      configPath: 'Config File Path', defaultPath: 'Leave empty for default ~/redc/config.yaml', loadConfig: 'Load Config',
      currentConfig: 'Current config', securityTip: 'Security Notice:', securityInfo: 'Credentials are displayed in masked form. Re-enter full values when editing. Empty fields won\'t overwrite existing config.',
      edit: 'Edit', cancel: 'Cancel', save: 'Save', notSet: 'Not set', enterNew: 'Enter new value', clickLoad: 'Click "Load Config" to view credentials',
      confirmDelete: 'Confirm Delete', cannotUndo: 'This cannot be undone', confirmDeleteScene: 'Are you sure you want to delete scene', region: 'Region', credentialsJson: 'Credentials JSON',
      profile: 'Profiles', profileName: 'Profile Name', templateDir: 'Template Directory', profileManage: 'Profile Management', activeProfile: 'Active Profile',
      createProfile: 'Create Profile', saveProfile: 'Save Profile', deleteProfile: 'Delete Profile', selectProfile: 'Select profile...',
      profileHint: 'Each profile maps to its own config file and template directory',
      profileNameRequired: 'Profile name is required',
      profileCredentialsFrom: 'Credentials loaded from', profileTemplateFrom: 'Template directory from', profileSwitchHint: 'Switching profile will change credentials and template directory',
      selectTemplateErr: 'Please select a template',
      // Local templates i18n
      version: 'Version', author: 'Author', module: 'Module', description: 'Description', viewParams: 'View Params',
      cloneTemplate: 'Clone',
      editTemplate: 'Edit Template', templateFiles: 'Template Files', saveTemplate: 'Save Template',
      cloneTitle: 'Clone Template', cloneName: 'New Template Name', cloneHint: 'The copy is editable',
      noLocalTemplates: 'No local templates', goToRegistry: 'Go to registry to pull',
      confirmDeleteTemplate: 'Are you sure you want to delete template', deleteWarning: 'You need to pull from registry again to use it',
      deleting: 'Deleting...', refresh: 'Refresh', close: 'Close',
      paramName: 'Name', paramType: 'Type', paramDesc: 'Description', paramDefault: 'Default', paramRequired: 'Required',
      noParams: 'No configurable parameters', loadingParams: 'Loading parameters...',
      debugLogs: 'Debug Logs', debugLogsDesc: 'Show more verbose logs in console', enable: 'Enable', disable: 'Disable',
      terraformMirror: 'Terraform Mirror', mirrorEnabled: 'Enable mirror', mirrorConfigPath: 'Config file path',
      mirrorConfigHint: 'Leave empty to use default path', mirrorSetEnv: 'Set TF_CLI_CONFIG_FILE', mirrorSave: 'Save mirror config',
      mirrorAliyunPreset: 'Use Alibaba Cloud mirror', mirrorTencentPreset: 'Use Tencent Cloud mirror', mirrorVolcPreset: 'Use Volcengine mirror',
      mirrorDetected: 'Terraform init network issue detected', mirrorDetectedDesc: 'Try enabling a local mirror',
      mirrorApplyAliyun: 'Enable Alibaba Cloud mirror', mirrorGoSettings: 'Open settings', mirrorConfigFromEnv: 'Path is from TF_CLI_CONFIG_FILE',
      mirrorProviders: 'Providers', mirrorAliyun: 'Alibaba Cloud', mirrorTencent: 'Tencent Cloud', mirrorVolc: 'Volcengine',
      networkCheck: 'Network Diagnostics', networkCheckBtn: 'Test Terraform connectivity', networkChecking: 'Checking...',
      networkEndpoint: 'Endpoint', networkStatus: 'Status', networkLatency: 'Latency', networkError: 'Error',
    }
  };
  $: t = i18n[lang];

  function toggleLang() {
    lang = lang === 'zh' ? 'en' : 'zh';
    localStorage.setItem('lang', lang);
  }

  function openGitHub() {
    BrowserOpenURL('https://github.com/wgpsec/redc');
  }

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

  function stripAnsi(value) {
    if (!value) return '';
    return value.replace(/\x1B\[[0-9;]*m/g, '');
  }

  function normalizeVersion(value) {
    if (!value) return '';
    return String(value).trim().replace(/^v/i, '');
  }

  function compareVersions(a, b) {
    const va = normalizeVersion(a).split('.').map(part => parseInt(part, 10));
    const vb = normalizeVersion(b).split('.').map(part => parseInt(part, 10));
    const maxLen = Math.max(va.length, vb.length);
    for (let i = 0; i < maxLen; i += 1) {
      const na = Number.isFinite(va[i]) ? va[i] : 0;
      const nb = Number.isFinite(vb[i]) ? vb[i] : 0;
      if (na > nb) return 1;
      if (na < nb) return -1;
    }
    return 0;
  }

  function hasUpdate(tmpl) {
    if (!tmpl || !tmpl.installed) return false;
    if (!tmpl.latest || !tmpl.localVersion) return false;
    return compareVersions(tmpl.latest, tmpl.localVersion) > 0;
  }

  function setCreateStatus(status, message, detail = '') {
    createStatus = status;
    createStatusMessage = message || '';
    createStatusDetail = detail || '';
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    if (status === 'success') {
      createStatusTimer = setTimeout(() => {
        createStatus = 'idle';
        createStatusMessage = '';
        createStatusDetail = '';
      }, 3000);
    }
  }

  function updateCreateStatusFromLog(message) {
    const cleanMessage = stripAnsi(message);
    if (cleanMessage.includes('正在创建场景:') || cleanMessage.includes('正在创建并运行场景:')) {
      setCreateStatus('creating', t.creating, message);
      return;
    }
    if (cleanMessage.includes('场景初始化中:')) {
      setCreateStatus('initializing', t.initializing, message);
      return;
    }
    if (cleanMessage.includes('场景创建成功')) {
      setCreateStatus('success', t.createSuccess, message);
      return;
    }
    if (cleanMessage.includes('场景创建失败') || cleanMessage.includes('创建场景时发生错误')) {
      setCreateStatus('error', t.createFailed, message);
      detectTerraformInitIssue(cleanMessage);
      return;
    }
  }

  function detectTerraformInitIssue(message) {
    const lower = message.toLowerCase();
    const hit = lower.includes('registry.terraform.io') || lower.includes('failed to query available provider packages') || lower.includes('x509') || lower.includes('tls') || lower.includes('context deadline') || lower.includes('client.timeout') || lower.includes('could not connect');
    if (hit) {
      terraformInitHint = { show: true, message: t.mirrorDetected, detail: message };
    }
  }

  function dismissTerraformInitHint() {
    terraformInitHint = { show: false, message: '', detail: '' };
  }

  $: createBusy = createStatus === 'creating' || createStatus === 'initializing';

  $: stateConfig = {
    'running': { label: t.running, color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopped': { label: t.stopped, color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'error': { label: t.error, color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' },
    'created': { label: t.created, color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500' },
    'pending': { label: t.pending, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: t.starting, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'stopping': { label: t.stopping, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'removing': { label: t.removing, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' }
  };

  onMount(async () => {
    EventsOn('log', (message) => {
      logs = [...logs, { time: new Date().toLocaleTimeString(), message }];
      updateCreateStatusFromLog(message);
    });
    EventsOn('refresh', async () => {
      await refreshData();
    });
    await refreshData();
  });

  onDestroy(() => {
    EventsOff('log');
    EventsOff('refresh');
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    if (registryNoticeTimer) {
      clearTimeout(registryNoticeTimer);
      registryNoticeTimer = null;
    }
    if (balanceCooldownTimer) {
      clearInterval(balanceCooldownTimer);
      balanceCooldownTimer = null;
    }
  });

  async function refreshData() {
    isLoading = true;
    error = '';
    try {
      [cases, templates, config, terraformMirror] = await Promise.all([
        ListCases(),
        ListTemplates(),
        GetConfig(),
        GetTerraformMirrorConfig()
      ]);
      // Initialize proxy form with current config
      proxyForm = {
        httpProxy: config.httpProxy || '',
        httpsProxy: config.httpsProxy || '',
        noProxy: config.noProxy || ''
      };
      debugEnabled = !!config.debugEnabled;
      terraformMirrorForm = {
        enabled: !!terraformMirror.enabled,
        configPath: terraformMirror.configPath || '',
        setEnv: !!terraformMirror.fromEnv,
        providers: {
          aliyun: terraformMirror.providers?.includes('aliyun'),
          tencent: terraformMirror.providers?.includes('tencent'),
          volc: terraformMirror.providers?.includes('volc')
        }
      };
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    } finally {
      isLoading = false;
    }
  }

  async function handleSaveTerraformMirror() {
    terraformMirrorSaving = true;
    terraformMirrorError = '';
    try {
      const providers = Object.entries(terraformMirrorForm.providers)
        .filter(([, enabled]) => enabled)
        .map(([key]) => key);
      await SaveTerraformMirrorConfig(
        terraformMirrorForm.enabled,
        providers,
        terraformMirrorForm.configPath,
        terraformMirrorForm.setEnv
      );
      terraformMirror = await GetTerraformMirrorConfig();
      terraformMirrorForm = {
        enabled: !!terraformMirror.enabled,
        configPath: terraformMirror.configPath || '',
        setEnv: !!terraformMirror.fromEnv,
        providers: {
          aliyun: terraformMirror.providers?.includes('aliyun'),
          tencent: terraformMirror.providers?.includes('tencent'),
          volc: terraformMirror.providers?.includes('volc')
        }
      };
    } catch (e) {
      terraformMirrorError = e.message || String(e);
    } finally {
      terraformMirrorSaving = false;
    }
  }

  async function enableAliyunMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, aliyun: true }
    };
    await handleSaveTerraformMirror();
  }

  async function enableTencentMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, tencent: true }
    };
    await handleSaveTerraformMirror();
  }

  async function enableVolcMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, volc: true }
    };
    await handleSaveTerraformMirror();
  }

  async function runTerraformNetworkCheck() {
    networkCheckLoading = true;
    networkCheckError = '';
    try {
      networkChecks = await TestTerraformEndpoints();
    } catch (e) {
      networkCheckError = e.message || String(e);
    } finally {
      networkCheckLoading = false;
    }
  }

  async function handleSaveProxy() {
    proxySaving = true;
    try {
      await SaveProxyConfig(proxyForm.httpProxy, proxyForm.httpsProxy, proxyForm.noProxy);
      config.httpProxy = proxyForm.httpProxy;
      config.httpsProxy = proxyForm.httpsProxy;
      config.noProxy = proxyForm.noProxy;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      proxySaving = false;
    }
  }

  async function handleToggleDebug() {
    const nextValue = !debugEnabled;
    debugSaving = true;
    try {
      await SetDebugLogging(nextValue);
      debugEnabled = nextValue;
      config.debugEnabled = nextValue;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      debugSaving = false;
    }
  }

  async function handleStart(caseId) {
    // 立即更新本地状态为"启动中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'starting' } : c);
    try {
      await StartCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData(); // 出错时刷新恢复真实状态
    }
  }

  async function handleStop(caseId) {
    // 立即更新本地状态为"停止中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'stopping' } : c);
    try {
      await StopCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData();
    }
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
    // 立即更新本地状态为"删除中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'removing' } : c);
    try {
      await RemoveCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData();
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
      // Initialize values with defaults
      variableValues = {};
      for (const v of templateVariables) {
        variableValues[v.name] = v.defaultValue || '';
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
      // Build vars object from variableValues, only include non-empty values
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
      // 不需要立即刷新，后端完成后会发送 refresh 事件
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
      // Build vars object from variableValues, only include non-empty values
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateAndRunCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
      // 不需要立即刷新，后端完成后会发送 refresh 事件
    } catch (e) {
      error = e.message || String(e);
      setCreateStatus('error', t.createFailed, error);
    }
  }

  function clearLogs() {
    logs = [];
  }

  function getShortId(id) {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }

  function getStateConfig(state) {
    return stateConfig[state] || stateConfig['pending'];
  }

  async function toggleCaseExpand(caseId, state) {
    if (expandedCase === caseId) {
      expandedCase = null;
      return;
    }
    expandedCase = caseId;
    // Only load outputs for running cases
    if (state === 'running' && !caseOutputs[caseId]) {
      try {
        const outputs = await GetCaseOutputs(caseId);
        if (outputs) {
          caseOutputs[caseId] = outputs;
          caseOutputs = caseOutputs; // trigger reactivity
        }
      } catch (e) {
        console.error('Failed to load outputs:', e);
      }
    }
  }

  let copiedKey = null;
  async function copyToClipboard(value, key) {
    try {
      await navigator.clipboard.writeText(value);
      copiedKey = key;
      setTimeout(() => { copiedKey = null; }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  // Registry functions
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

  async function handlePullTemplate(templateName, force = false) {
    pullingTemplates[templateName] = true;
    pullingTemplates = pullingTemplates;
    setRegistryNotice('info', `${t.pulling} ${templateName}`, false);
    try {
      await PullTemplate(templateName, force);
      // Refresh registry templates after successful pull
      await loadRegistryTemplates();
      setRegistryNotice('success', `${t.pullSuccess}: ${templateName}`);
    } catch (e) {
      error = e.message || String(e);
      setRegistryNotice('error', `${t.pullFailed}: ${templateName}`);
    } finally {
      pullingTemplates[templateName] = false;
      pullingTemplates = pullingTemplates;
    }
  }

  $: filteredRegistryTemplates = registryTemplates
    .filter(t => 
      !registrySearch || 
      t.name.toLowerCase().includes(registrySearch.toLowerCase()) ||
      (t.author && t.author.toLowerCase().includes(registrySearch.toLowerCase())) ||
      (t.description && t.description.toLowerCase().includes(registrySearch.toLowerCase())) ||
      (t.tags && t.tags.some(tag => tag.toLowerCase().includes(registrySearch.toLowerCase())))
    )
    .sort((a, b) => {
      // Installed templates first
      if (a.installed && !b.installed) return -1;
      if (!a.installed && b.installed) return 1;
      // Then sort by name alphabetically
      return a.name.localeCompare(b.name);
    });

  // Listen for refresh events to update pulling status
  $: if (registryTemplates.length > 0) {
    // Reset pulling status when templates are refreshed
    for (const t of registryTemplates) {
      if (t.installed && pullingTemplates[t.name]) {
        pullingTemplates[t.name] = false;
      }
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
    resourcesLoading = true;
    resourcesError = '';
    try {
      resourceSummary = await GetResourceSummary() || [];
    } catch (e) {
      resourcesError = e.message || String(e);
      resourceSummary = [];
    } finally {
      resourcesLoading = false;
    }
  }

  async function queryBalances() {
    if (balanceCooldown > 0) return;
    balanceLoading = true;
    balanceError = '';
    try {
      balanceResults = await GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud']) || [];
      balanceCooldown = 5;
      if (balanceCooldownTimer) {
        clearInterval(balanceCooldownTimer);
      }
      balanceCooldownTimer = setInterval(() => {
        balanceCooldown = Math.max(0, balanceCooldown - 1);
        if (balanceCooldown === 0 && balanceCooldownTimer) {
          clearInterval(balanceCooldownTimer);
          balanceCooldownTimer = null;
        }
      }, 1000);
    } catch (e) {
      balanceError = e.message || String(e);
    } finally {
      balanceLoading = false;
    }
  }

  // Compose functions
  function parseComposeProfiles(value) {
    if (!value) return [];
    return value
      .split(',')
      .map(v => v.trim())
      .filter(Boolean);
  }

  async function previewCompose() {
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

  async function handleComposeUp() {
    composeActionLoading = true;
    composeError = '';
    try {
      await ComposeUp(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
    } finally {
      composeActionLoading = false;
    }
  }

  async function handleComposeDown() {
    composeActionLoading = true;
    composeError = '';
    try {
      await ComposeDown(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
    } finally {
      composeActionLoading = false;
    }
  }

  // Credentials functions
  async function loadProvidersConfig() {
    credentialsLoading = true;
    try {
      providersConfig = await GetProvidersConfig(customConfigPath);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsLoading = false;
    }
  }

  async function loadProfiles() {
    profileLoading = true;
    profileError = '';
    try {
      const [list, active] = await Promise.all([
        ListProfiles(),
        GetActiveProfile()
      ]);
      profiles = list || [];
      if (active && active.id) {
        activeProfileId = active.id;
        profileForm = {
          name: active.name || '',
          configPath: active.configPath || '',
          templateDir: active.templateDir || ''
        };
        customConfigPath = profileForm.configPath;
      }
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileLoading = false;
    }
  }

  async function handleProfileChange(id) {
    if (!id) return;
    profileLoading = true;
    profileError = '';
    try {
      const active = await SetActiveProfile(id);
      activeProfileId = active.id;
      profileForm = {
        name: active.name || '',
        configPath: active.configPath || '',
        templateDir: active.templateDir || ''
      };
      customConfigPath = profileForm.configPath;
      await loadProvidersConfig();
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileLoading = false;
    }
  }

  async function handleCreateProfile() {
    if (!profileForm.name) {
      profileError = t.profileNameRequired;
      return;
    }
    profileSaving = true;
    profileError = '';
    try {
      const created = await CreateProfile(profileForm.name, profileForm.configPath, profileForm.templateDir);
      profiles = await ListProfiles();
      await handleProfileChange(created.id);
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  async function handleSaveProfile() {
    if (!activeProfileId) return;
    if (!profileForm.name) {
      profileError = t.profileNameRequired;
      return;
    }
    profileSaving = true;
    profileError = '';
    try {
      const updated = await UpdateProfile(activeProfileId, profileForm.name, profileForm.configPath, profileForm.templateDir);
      profiles = await ListProfiles();
      activeProfileId = updated.id;
      profileForm = {
        name: updated.name || '',
        configPath: updated.configPath || '',
        templateDir: updated.templateDir || ''
      };
      customConfigPath = profileForm.configPath;
      await SetActiveProfile(activeProfileId);
      await loadProvidersConfig();
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  async function handleDeleteProfile() {
    if (!activeProfileId) return;
    profileSaving = true;
    profileError = '';
    try {
      await DeleteProfile(activeProfileId);
      await loadProfiles();
      if (activeProfileId) {
        await handleProfileChange(activeProfileId);
      }
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  function startEditProvider(provider) {
    editingProvider = provider.name;
    editFields = {};
    // Initialize edit fields with empty values (user must re-enter secrets)
    for (const key of Object.keys(provider.fields)) {
      // For non-secret fields (like region), pre-fill with current value
      if (!provider.hasSecrets || !provider.hasSecrets[key]) {
        editFields[key] = provider.fields[key] || '';
      } else {
        editFields[key] = '';
      }
    }
  }

  function cancelEditProvider() {
    editingProvider = null;
    editFields = {};
  }

  async function saveProviderCredentials(providerName) {
    credentialsSaving[providerName] = true;
    credentialsSaving = credentialsSaving;
    try {
      await SaveProvidersConfig(providerName, editFields, customConfigPath);
      editingProvider = null;
      editFields = {};
      await loadProvidersConfig();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsSaving[providerName] = false;
      credentialsSaving = credentialsSaving;
    }
  }

  function getFieldLabel(key) {
    const labels = {
      accessKey: 'Access Key',
      secretKey: 'Secret Key',
      secretId: 'Secret ID',
      region: '区域',
      credentials: '凭据 JSON',
      project: '项目 ID',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      subscriptionId: 'Subscription ID',
      tenantId: 'Tenant ID',
      user: '用户 OCID',
      tenancy: 'Tenancy OCID',
      fingerprint: '指纹',
      keyFile: '私钥文件路径',
      email: '邮箱',
      apiKey: 'API Key',
    };
    return labels[key] || key;
  }

  function isSecretField(key) {
    const secrets = ['accessKey', 'secretKey', 'secretId', 'credentials', 'clientId', 'clientSecret', 'subscriptionId', 'tenantId', 'user', 'tenancy', 'fingerprint', 'apiKey'];
    return secrets.includes(key);
  }

  // Local templates functions
  async function loadLocalTemplates() {
    localTemplatesLoading = true;
    try {
      localTemplates = await ListTemplates() || [];
    } catch (e) {
      error = e.message || String(e);
      localTemplates = [];
    } finally {
      localTemplatesLoading = false;
    }
  }

  async function showTemplateDetail(tmpl) {
    localTemplateDetail = tmpl;
    localTemplateVars = [];
    localTemplateVarsLoading = true;
    try {
      const vars = await GetTemplateVariables(tmpl.name);
      localTemplateVars = vars || [];
    } catch (e) {
      console.error('Failed to load template variables:', e);
      localTemplateVars = [];
    } finally {
      localTemplateVarsLoading = false;
    }
  }

  function closeTemplateDetail() {
    localTemplateDetail = null;
    localTemplateVars = [];
  }

  function showDeleteTemplateConfirm(name) {
    deleteTemplateConfirm = { show: true, name };
  }

  function cancelDeleteTemplate() {
    deleteTemplateConfirm = { show: false, name: '' };
  }

  async function confirmDeleteTemplate() {
    const name = deleteTemplateConfirm.name;
    deleteTemplateConfirm = { show: false, name: '' };
    deletingTemplate[name] = true;
    deletingTemplate = deletingTemplate;
    try {
      await RemoveTemplate(name);
      await loadLocalTemplates();
      // Also refresh main templates list
      templates = await ListTemplates() || [];
    } catch (e) {
      error = e.message || String(e);
    } finally {
      deletingTemplate[name] = false;
      deletingTemplate = deletingTemplate;
    }
  }

  async function handleCloneTemplate(tmpl) {
    cloneTemplateModal = { show: true, source: tmpl.name, target: `${tmpl.name}-copy` };
  }

  function cancelCloneTemplate() {
    cloneTemplateModal = { show: false, source: '', target: '' };
  }

  async function confirmCloneTemplate() {
    const targetName = cloneTemplateModal.target.trim();
    const sourceName = cloneTemplateModal.source;
    cloneTemplateModal = { show: false, source: '', target: '' };
    if (!targetName) return;
    try {
      await CopyTemplate(sourceName, targetName);
      await loadLocalTemplates();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function openTemplateEditor(tmpl) {
    templateEditor = { show: true, name: tmpl.name, files: {}, active: '', saving: false, error: '' };
    try {
      const files = await GetTemplateFiles(tmpl.name);
      const names = Object.keys(files || {});
      templateEditor = {
        ...templateEditor,
        files: files || {},
        active: names.length > 0 ? names[0] : '',
      };
    } catch (e) {
      templateEditor = { ...templateEditor, error: e.message || String(e) };
    }
  }

  function closeTemplateEditor() {
    templateEditor = { show: false, name: '', files: {}, active: '', saving: false, error: '' };
  }

  async function saveTemplateEditor() {
    if (!templateEditor.name) return;
    templateEditor = { ...templateEditor, saving: true, error: '' };
    try {
      await SaveTemplateFiles(templateEditor.name, templateEditor.files);
      templateEditor = { ...templateEditor, saving: false };
    } catch (e) {
      templateEditor = { ...templateEditor, saving: false, error: e.message || String(e) };
    }
  }

  $: filteredLocalTemplates = localTemplates
    .filter(t => 
      !localTemplatesSearch || 
      t.name.toLowerCase().includes(localTemplatesSearch.toLowerCase()) ||
      (t.description && t.description.toLowerCase().includes(localTemplatesSearch.toLowerCase())) ||
      (t.module && t.module.toLowerCase().includes(localTemplatesSearch.toLowerCase()))
    )
    .sort((a, b) => a.name.localeCompare(b.name));
</script>

<div class="h-screen flex bg-[#fafbfc]">
  <!-- Sidebar -->
  <aside class="w-44 bg-white border-r border-gray-100 flex flex-col">
    <div class="h-14 flex items-center px-4 border-b border-gray-100">
      <div class="flex items-center gap-2">
        <div class="w-6 h-6 rounded-md bg-gradient-to-br from-rose-500 to-red-600 flex items-center justify-center">
          <span class="text-white text-[10px] font-bold">R</span>
        </div>
        <span class="text-[14px] font-semibold text-gray-900">RedC</span>
      </div>
    </div>
    
    <nav class="flex-1 p-2">
      <div class="space-y-0.5">
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'dashboard' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'dashboard'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
          </svg>
          {t.dashboard}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'console' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'console'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
          {t.console}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'resources' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'resources'; loadResourceSummary(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 7.5l9 4.5 9-4.5M3 12l9 4.5 9-4.5M3 16.5l9 4.5 9-4.5" />
          </svg>
          {t.resources}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'compose' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'compose'; previewCompose(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h12A2.25 2.25 0 0120.25 6v12A2.25 2.25 0 0118 20.25H6A2.25 2.25 0 013.75 18V6z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 8h8M8 12h8M8 16h5" />
          </svg>
          {t.compose}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'settings' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'settings'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          {t.settings}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'credentials' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'credentials'; loadProfiles(); loadProvidersConfig(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          {t.credentials}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'registry' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'registry'; loadRegistryTemplates(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          {t.registry}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'localTemplates' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'localTemplates'; loadLocalTemplates(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
          {t.localTemplates}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'ai' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'ai'; loadMCPStatus(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
          </svg>
          {t.ai}
        </button>
      </div>
    </nav>

    <div class="p-2 border-t border-gray-100">
      <div class="flex items-center justify-between px-2 py-2">
        <span class="text-[10px] text-gray-400">v2.1.0</span>
        <div class="flex items-center gap-1">
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors text-[10px] font-medium"
            on:click={toggleLang}
            title={lang === 'zh' ? 'Switch to English' : '切换到中文'}
          >{lang === 'zh' ? 'EN' : '中'}</button>
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
            on:click={openGitHub}
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

  <!-- Main -->
  <div class="flex-1 flex flex-col min-w-0">
    <!-- Header -->
    <header class="h-14 bg-white border-b border-gray-100 flex items-center justify-between px-6">
      <h1 class="text-[15px] font-medium text-gray-900">
        {#if activeTab === 'dashboard'}{t.sceneManage}{:else if activeTab === 'console'}{t.console}{:else if activeTab === 'resources'}{t.resources}{:else if activeTab === 'compose'}{t.compose}{:else if activeTab === 'registry'}{t.templateRepo}{:else if activeTab === 'localTemplates'}{t.localTmplManage}{:else if activeTab === 'ai'}{t.aiIntegration}{:else if activeTab === 'credentials'}{t.credentials}{:else}{t.settings}{/if}
      </h1>
      <button 
        class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-50 text-gray-400 hover:text-gray-600 transition-colors"
        on:click={() => { refreshData(); if (activeTab === 'registry') loadRegistryTemplates(); if (activeTab === 'localTemplates') loadLocalTemplates(); if (activeTab === 'ai') loadMCPStatus(); if (activeTab === 'credentials') { loadProfiles(); loadProvidersConfig(); } if (activeTab === 'resources') loadResourceSummary(); if (activeTab === 'compose') previewCompose(); }}
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
        </svg>
      </button>
    </header>

    <!-- Content -->
    <main class="flex-1 overflow-auto p-6">
      {#if error}
        <div class="mb-5 flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
          <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
          <span class="text-[13px] text-red-700 flex-1">{error}</span>
          <button class="text-red-400 hover:text-red-600" on:click={() => error = ''}>
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
        <div class="space-y-5">
          <!-- Quick Create -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-end gap-4 mb-4">
              <div class="flex-1">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.template}</label>
                <select 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={selectedTemplate}
                  on:change={() => loadTemplateVariables(selectedTemplate)}
                >
                  <option value="">{t.selectTemplate}</option>
                  {#each templates || [] as tmpl}
                    <option value={tmpl.name}>{tmpl.name}</option>
                  {/each}
                </select>
              </div>
              <div class="w-48">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.name}</label>
                <input 
                  type="text" 
                  placeholder={t.optional}
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={newCaseName} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                on:click={handleCreate}
                disabled={createBusy}
              >
                {t.create}
              </button>
              <button 
                class="h-10 px-5 bg-emerald-500 text-white text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                on:click={handleCreateAndRun}
                disabled={createBusy}
              >
                {t.createAndRun}
              </button>
            </div>

            {#if createStatus !== 'idle'}
              <div class="mt-3 flex items-center gap-2 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-[12px]">
                {#if createStatus === 'creating' || createStatus === 'initializing'}
                  <div class="w-3.5 h-3.5 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
                  <span class="text-gray-700">{createStatusMessage}</span>
                {:else if createStatus === 'success'}
                  <span class="text-emerald-600">{createStatusMessage}</span>
                {:else if createStatus === 'error'}
                  <span class="text-red-600">{createStatusMessage}</span>
                {/if}
                {#if createStatusDetail}
                  <span class="text-gray-400 truncate">{createStatusDetail}</span>
                {/if}
              </div>
            {/if}

            {#if terraformInitHint.show}
              <div class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[12px] text-amber-700">
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
                    <div class="mt-2 flex flex-wrap gap-2">
                      <button
                        class="h-8 px-3 bg-amber-600 text-white text-[12px] font-medium rounded-md hover:bg-amber-700 transition-colors"
                        on:click={enableAliyunMirrorQuick}
                      >{t.mirrorApplyAliyun}</button>
                      <button
                        class="h-8 px-3 bg-white text-amber-700 text-[12px] font-medium rounded-md border border-amber-200 hover:bg-amber-100 transition-colors"
                        on:click={() => activeTab = 'settings'}
                      >{t.mirrorGoSettings}</button>
                    </div>
                  </div>
                  <button class="text-amber-400 hover:text-amber-600" on:click={dismissTerraformInitHint}>
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </div>
            {/if}
            
            <!-- Template Variables -->
            {#if templateVariables.length > 0}
              <div class="border-t border-gray-100 pt-4 mt-4">
                <div class="text-[12px] font-medium text-gray-500 mb-3">{t.templateParams}</div>
                <div class="grid grid-cols-2 gap-3">
                  {#each templateVariables as variable}
                    <div class="flex flex-col">
                      <label class="text-[11px] text-gray-500 mb-1">
                        {variable.name}
                        {#if variable.required}
                          <span class="text-red-500">*</span>
                        {/if}
                        {#if variable.description}
                          <span class="text-gray-400 ml-1">({variable.description})</span>
                        {/if}
                      </label>
                      <input 
                        type="text"
                        class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                        placeholder={variable.defaultValue || ''}
                        bind:value={variableValues[variable.name]}
                      />
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>

          <!-- Table -->
          <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="border-b border-gray-100">
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.id}</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.name}</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.type}</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.state}</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.time}</th>
                  <th class="text-right px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.actions}</th>
                </tr>
              </thead>
              <tbody>
                {#each cases || [] as c, i}
                  <tr 
                    class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer"
                    on:click={() => toggleCaseExpand(c.id, c.state)}
                  >
                    <td class="px-5 py-3.5">
                      <div class="flex items-center gap-2">
                        <svg class="w-4 h-4 text-gray-400 transition-transform {expandedCase === c.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                        </svg>
                        <code class="text-[12px] text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">{getShortId(c.id)}</code>
                      </div>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[13px] font-medium text-gray-900">{c.name}</span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[13px] text-gray-600">{c.type}</span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {getStateConfig(c.state).color}">
                        <span class="w-1.5 h-1.5 rounded-full {getStateConfig(c.state).dot}"></span>
                        {getStateConfig(c.state).label}
                      </span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[12px] text-gray-500">{c.createTime}</span>
                    </td>
                    <td class="px-5 py-3.5 text-right" on:click|stopPropagation>
                      <div class="inline-flex items-center gap-1">
                        {#if c.state === 'starting' || c.state === 'stopping' || c.state === 'removing'}
                          <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                            {stateConfig[c.state]?.label || t.processing}...
                          </span>
                        {:else if c.state !== 'running'}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                            on:click={() => handleStart(c.id)}
                          >{t.start}</button>
                        {:else}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                            on:click={() => handleStop(c.id)}
                          >{t.stop}</button>
                        {/if}
                        {#if c.state !== 'starting' && c.state !== 'stopping' && c.state !== 'removing'}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-gray-600 bg-gray-50 rounded-md hover:bg-gray-100 transition-colors"
                            on:click={() => showDeleteConfirm(c.id, c.name)}
                          >{t.delete}</button>
                        {/if}
                      </div>
                    </td>
                  </tr>
                  <!-- Expanded row for outputs -->
                  {#if expandedCase === c.id}
                    <tr class="bg-slate-50">
                      <td colspan="6" class="px-5 py-4">
                        <div class="pl-6">
                          {#if c.state === 'running'}
                            {#if caseOutputs[c.id]}
                              <div class="grid grid-cols-2 gap-3">
                                {#each Object.entries(caseOutputs[c.id]) as [key, value]}
                                  <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                                    <div class="flex items-center justify-between mb-1">
                                      <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                                      <button 
                                        class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                        on:click|stopPropagation={() => copyToClipboard(value, key)}
                                        title={t.copy}
                                      >
                                        {#if copiedKey === key}
                                          <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                          </svg>
                                          <span class="text-[10px] text-emerald-500">{t.copied}</span>
                                        {:else}
                                          <svg class="w-4 h-4 text-gray-400 hover:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                    <td colspan="6" class="py-16">
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
        </div>

      {:else if activeTab === 'console'}
        <div class="h-full flex flex-col bg-[#1e1e1e] rounded-xl overflow-hidden">
          <div class="flex items-center justify-between px-4 py-2.5 bg-[#252526] border-b border-[#3c3c3c]">
            <div class="flex items-center gap-2">
              <div class="flex gap-1.5">
                <span class="w-3 h-3 rounded-full bg-[#ff5f56]"></span>
                <span class="w-3 h-3 rounded-full bg-[#ffbd2e]"></span>
                <span class="w-3 h-3 rounded-full bg-[#27ca40]"></span>
              </div>
              <span class="text-[12px] text-gray-500 ml-2">{t.terminal}</span>
            </div>
            <button 
              class="text-[11px] text-gray-500 hover:text-gray-300 transition-colors"
              on:click={clearLogs}
            >{t.clear}</button>
          </div>
          <div class="flex-1 p-4 overflow-auto font-mono text-[12px] leading-5">
            {#each logs as log}
              <div class="flex">
                <span class="text-gray-600 select-none">[{log.time}]</span>
                <span class="text-gray-300 ml-2">{stripAnsi(log.message)}</span>
              </div>
            {:else}
              <div class="text-gray-600">$ {t.waitOutput}</div>
            {/each}
          </div>
        </div>

      {:else if activeTab === 'resources'}
        <div class="max-w-3xl space-y-5">
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between mb-4">
              <div>
                <h3 class="text-[14px] font-semibold text-gray-900">{t.resourceSummary}</h3>
                <p class="text-[12px] text-gray-500"></p>
              </div>
              <button
                class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={loadResourceSummary}
                disabled={resourcesLoading}
              >
                {resourcesLoading ? t.loading : t.refresh}
              </button>
            </div>

            {#if resourcesError}
              <div class="text-[12px] text-red-500">{resourcesError}</div>
            {:else if resourcesLoading}
              <div class="flex items-center justify-center h-24">
                <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
              </div>
            {:else}
              <div class="border border-gray-100 rounded-lg overflow-hidden">
                <table class="w-full text-[12px]">
                  <thead>
                    <tr class="bg-gray-50 border-b border-gray-100">
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.resourceType}</th>
                      <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.resourceCount}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each resourceSummary as r}
                      <tr class="border-b border-gray-50">
                        <td class="px-4 py-3 text-gray-700">{r.type}</td>
                        <td class="px-4 py-3 text-right text-gray-700">{r.count}</td>
                      </tr>
                    {:else}
                      <tr>
                        <td colspan="2" class="py-12 text-center text-[12px] text-gray-400">{t.noScene}</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>

          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between mb-4">
              <div>
                <h3 class="text-[14px] font-semibold text-gray-900">{t.balanceQuery}</h3>
                <p class="text-[12px] text-gray-500">{t.profileSwitchHint}</p>
              </div>
              <button
                class="h-9 px-4 bg-blue-600 text-white text-[12px] font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
                on:click={queryBalances}
                disabled={balanceLoading || balanceCooldown > 0}
              >
                {balanceLoading ? t.loading : balanceCooldown > 0 ? `${t.balanceCooldown} ${balanceCooldown}s` : t.balanceQuery}
              </button>
            </div>

            {#if balanceError}
              <div class="text-[12px] text-red-500">{balanceError}</div>
            {:else}
              <div class="border border-gray-100 rounded-lg overflow-hidden">
                <table class="w-full text-[12px]">
                  <thead>
                    <tr class="bg-gray-50 border-b border-gray-100">
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceProvider}</th>
                      <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.balanceAmount}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceCurrency}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceUpdatedAt}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each balanceResults as b}
                      <tr class="border-b border-gray-50">
                        <td class="px-4 py-3 text-gray-700">{b.provider}</td>
                        <td class="px-4 py-3 text-right text-gray-700">{b.amount}</td>
                        <td class="px-4 py-3 text-gray-700">{b.currency}</td>
                        <td class="px-4 py-3 text-gray-500">{b.updatedAt}</td>
                      </tr>
                      {#if b.error}
                        <tr class="border-b border-gray-50">
                          <td colspan="4" class="px-4 pb-3 text-[11px] text-amber-600">{b.error}</td>
                        </tr>
                      {/if}
                    {:else}
                      <tr>
                        <td colspan="4" class="py-12 text-center text-[12px] text-gray-400">{t.balancePlaceholder}</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        </div>

      {:else if activeTab === 'compose'}
        <div class="max-w-4xl space-y-5">
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.composeFile}</label>
                <input
                  type="text"
                  placeholder="redc-compose.yaml"
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={composeFilePath}
                />
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.composeProfiles}</label>
                <input
                  type="text"
                  placeholder="prod,dev"
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={composeProfiles}
                />
              </div>
            </div>
            <div class="mt-4 flex flex-wrap gap-2">
              <button
                class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={previewCompose}
                disabled={composeLoading}
              >
                {composeLoading ? t.loading : t.previewCompose}
              </button>
              <button
                class="h-9 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50"
                on:click={handleComposeUp}
                disabled={composeActionLoading}
              >
                {composeActionLoading ? t.processing : t.composeUp}
              </button>
              <button
                class="h-9 px-4 bg-red-500 text-white text-[12px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
                on:click={handleComposeDown}
                disabled={composeActionLoading}
              >
                {composeActionLoading ? t.processing : t.composeDown}
              </button>
            </div>
            {#if composeError}
              <div class="mt-3 text-[12px] text-red-500">{composeError}</div>
            {/if}
          </div>

          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="text-[14px] font-semibold text-gray-900 mb-4">{t.composePreview}</div>
            {#if composeLoading}
              <div class="flex items-center justify-center h-24">
                <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
              </div>
            {:else if composeSummary && composeSummary.services && composeSummary.services.length > 0}
              <div class="border border-gray-100 rounded-lg overflow-hidden">
                <table class="w-full text-[12px]">
                  <thead>
                    <tr class="bg-gray-50 border-b border-gray-100">
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceName}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceTemplate}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceProvider}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceDepends}</th>
                      <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.serviceReplicas}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each composeSummary.services as svc}
                      <tr class="border-b border-gray-50">
                        <td class="px-4 py-3 text-gray-700">{svc.name}</td>
                        <td class="px-4 py-3 text-gray-700">{svc.template}</td>
                        <td class="px-4 py-3 text-gray-700">{svc.provider || '-'}</td>
                        <td class="px-4 py-3 text-gray-700">{(svc.dependsOn && svc.dependsOn.length > 0) ? svc.dependsOn.join(', ') : '-'}</td>
                        <td class="px-4 py-3 text-right text-gray-700">{svc.replicas || 1}</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {:else}
              <div class="py-12 text-center text-[12px] text-gray-400">{t.noScene}</div>
            {/if}
          </div>
        </div>

      {:else if activeTab === 'settings'}
        <div class="max-w-xl space-y-4">
          <!-- 基本信息 -->
          <div class="bg-white rounded-xl border border-gray-100 divide-y divide-gray-100">
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">{t.redcPath}</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.redcPath || '-'}</div>
            </div>
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">{t.projectPath}</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.projectPath || '-'}</div>
            </div>
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">{t.logPath}</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.logPath || '-'}</div>
            </div>
          </div>

          <!-- 代理配置 -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="text-[14px] font-medium text-gray-900 mb-4">{t.proxyConfig}</div>
            <div class="space-y-4">
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.httpProxy}</label>
                <input 
                  type="text" 
                  placeholder="http://127.0.0.1:7890" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={proxyForm.httpProxy} 
                />
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.httpsProxy}</label>
                <input 
                  type="text" 
                  placeholder="http://127.0.0.1:7890" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={proxyForm.httpsProxy} 
                />
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.noProxyLabel}</label>
                <input 
                  type="text" 
                  placeholder="localhost,127.0.0.1,.local" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={proxyForm.noProxy} 
                />
              </div>
              <div class="pt-2">
                <button 
                  class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  on:click={handleSaveProxy}
                  disabled={proxySaving}
                >
                  {proxySaving ? t.saving : t.saveProxy}
                </button>
                <span class="ml-3 text-[12px] text-gray-500">{t.proxyHint}</span>
              </div>
            </div>
          </div>

          <!-- Terraform 镜像加速 -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-start justify-between mb-4">
              <div>
                <div class="text-[14px] font-medium text-gray-900">{t.terraformMirror}</div>
                <div class="text-[12px] text-gray-500 mt-1">{t.mirrorConfigHint}</div>
              </div>
              <button
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
                class:bg-emerald-500={terraformMirrorForm.enabled}
                class:bg-gray-300={!terraformMirrorForm.enabled}
                on:click={() => terraformMirrorForm = { ...terraformMirrorForm, enabled: !terraformMirrorForm.enabled }}
                aria-label={t.mirrorEnabled}
              >
                <span
                  class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  class:translate-x-6={terraformMirrorForm.enabled}
                  class:translate-x-1={!terraformMirrorForm.enabled}
                ></span>
              </button>
            </div>
            <div class="space-y-4">
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorProviders}</label>
                <div class="flex flex-wrap items-center gap-3 text-[12px] text-gray-700">
                  <label class="inline-flex items-center gap-2">
                    <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.aliyun} />
                    <span>{t.mirrorAliyun}</span>
                  </label>
                  <label class="inline-flex items-center gap-2">
                    <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.tencent} />
                    <span>{t.mirrorTencent}</span>
                  </label>
                  <label class="inline-flex items-center gap-2">
                    <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.volc} />
                    <span>{t.mirrorVolc}</span>
                  </label>
                </div>
                <div class="mt-2 text-[11px] text-gray-500">
                  阿里云: https://mirrors.aliyun.com/terraform/ · 腾讯云: https://mirrors.tencent.com/terraform/ · 火山云: https://mirrors.volces.com/terraform/
                </div>
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorConfigPath}</label>
                <input
                  type="text"
                  placeholder={terraformMirror.configPath || t.mirrorConfigHint}
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={terraformMirrorForm.configPath}
                />
                {#if terraformMirror.fromEnv}
                  <div class="mt-1 text-[11px] text-amber-600">{t.mirrorConfigFromEnv}</div>
                {/if}
              </div>
              <div class="flex items-center gap-2 text-[12px] text-gray-600">
                <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.setEnv} />
                <span>{t.mirrorSetEnv}</span>
              </div>
              <div class="pt-1 flex flex-wrap gap-2 items-center">
                <button
                  class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                  on:click={handleSaveTerraformMirror}
                  disabled={terraformMirrorSaving}
                >
                  {terraformMirrorSaving ? t.saving : t.mirrorSave}
                </button>
                <button
                  class="h-9 px-4 bg-amber-500 text-white text-[12px] font-medium rounded-lg hover:bg-amber-600 transition-colors"
                  on:click={enableAliyunMirrorQuick}
                >
                  {t.mirrorAliyunPreset}
                </button>
                <button
                  class="h-9 px-4 bg-sky-500 text-white text-[12px] font-medium rounded-lg hover:bg-sky-600 transition-colors"
                  on:click={enableTencentMirrorQuick}
                >
                  {t.mirrorTencentPreset}
                </button>
                <button
                  class="h-9 px-4 bg-violet-500 text-white text-[12px] font-medium rounded-lg hover:bg-violet-600 transition-colors"
                  on:click={enableVolcMirrorQuick}
                >
                  {t.mirrorVolcPreset}
                </button>
                {#if terraformMirrorError}
                  <span class="text-[12px] text-red-500">{terraformMirrorError}</span>
                {:else if terraformMirror.managed}
                  <span class="text-[12px] text-emerald-600">OK</span>
                {/if}
              </div>
            </div>
          </div>

          <!-- 网络诊断 -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between">
              <div class="text-[14px] font-medium text-gray-900">{t.networkCheck}</div>
              <button
                class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={runTerraformNetworkCheck}
                disabled={networkCheckLoading}
              >
                {networkCheckLoading ? t.networkChecking : t.networkCheckBtn}
              </button>
            </div>
            {#if networkCheckError}
              <div class="mt-3 text-[12px] text-red-500">{networkCheckError}</div>
            {/if}
            {#if networkChecks.length > 0}
              <div class="mt-4 border border-gray-100 rounded-lg overflow-hidden">
                <table class="w-full text-[12px]">
                  <thead>
                    <tr class="bg-gray-50 border-b border-gray-100">
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.networkEndpoint}</th>
                      <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.networkStatus}</th>
                      <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.networkLatency}</th>
                      <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.networkError}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each networkChecks as item}
                      <tr class="border-b border-gray-50">
                        <td class="px-4 py-3 text-gray-700">{item.name}</td>
                        <td class="px-4 py-3 text-right {item.ok ? 'text-emerald-600' : 'text-red-600'}">{item.ok ? 'OK' : item.status || '-'}</td>
                        <td class="px-4 py-3 text-right text-gray-700">{item.latencyMs} ms</td>
                        <td class="px-4 py-3 text-gray-500 truncate" title={item.error}>{item.error || '-'}</td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>

          <!-- 调试日志 -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-[14px] font-medium text-gray-900">{t.debugLogs}</div>
                <div class="text-[12px] text-gray-500 mt-1">{t.debugLogsDesc}</div>
              </div>
              <button
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                class:bg-emerald-500={debugEnabled}
                class:bg-gray-300={!debugEnabled}
                on:click={handleToggleDebug}
                disabled={debugSaving}
                aria-label={debugEnabled ? t.disable : t.enable}
              >
                <span
                  class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  class:translate-x-6={debugEnabled}
                  class:translate-x-1={!debugEnabled}
                ></span>
              </button>
            </div>
          </div>
        </div>

      {:else if activeTab === 'registry'}
        <div class="space-y-5">
          <!-- Search -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center gap-4">
              <div class="flex-1 relative">
                <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input 
                  type="text" 
                  placeholder={t.search}
                  class="w-full h-10 pl-10 pr-4 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={registrySearch} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={loadRegistryTemplates}
                disabled={registryLoading}
              >
                {registryLoading ? t.loading : t.refreshRepo}
              </button>
            </div>
            {#if registryNotice.message}
              <div class="mt-3 flex items-center gap-2 rounded-lg border px-3 py-2 text-[12px]
                {registryNotice.type === 'success' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : registryNotice.type === 'error' ? 'bg-red-50 border-red-100 text-red-700' : 'bg-amber-50 border-amber-100 text-amber-700'}">
                {#if registryNotice.type === 'info'}
                  <div class="w-3.5 h-3.5 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></div>
                {/if}
                <span class="flex-1 truncate">{registryNotice.message}</span>
                <button class="text-gray-400 hover:text-gray-600" on:click={() => setRegistryNotice('', '')}>
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            {/if}
          </div>

          {#if registryError}
            <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
              <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              <span class="text-[13px] text-red-700 flex-1">{registryError}</span>
              <button class="text-red-400 hover:text-red-600" on:click={() => registryError = ''}>
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          {/if}

          {#if registryLoading}
            <div class="flex items-center justify-center h-64">
              <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
            </div>
          {:else}
            <!-- Template Grid -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each filteredRegistryTemplates as tmpl}
                <div class="bg-white rounded-xl border border-gray-100 p-5 hover:shadow-md transition-shadow">
                  <div class="flex items-start justify-between mb-3">
                    <div class="flex-1 min-w-0">
                      <h3 class="text-[14px] font-semibold text-gray-900 truncate">{tmpl.name}</h3>
                      <p class="text-[12px] text-gray-500 mt-0.5">v{tmpl.latest}</p>
                    </div>
                    {#if tmpl.installed}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[11px] font-medium rounded-full">
                        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                        </svg>
                        {t.installed}
                      </span>
                    {/if}
                  </div>
                  
                  {#if tmpl.description}
                    <p class="text-[12px] text-gray-600 mb-3 line-clamp-2">{tmpl.description}</p>
                  {/if}
                  
                  {#if tmpl.tags && tmpl.tags.length > 0}
                    <div class="flex flex-wrap gap-1 mb-3">
                      {#each tmpl.tags.slice(0, 3) as tag}
                        <span class="px-2 py-0.5 bg-gray-100 text-gray-600 text-[10px] rounded-full">{tag}</span>
                      {/each}
                      {#if tmpl.tags.length > 3}
                        <span class="px-2 py-0.5 bg-gray-100 text-gray-400 text-[10px] rounded-full">+{tmpl.tags.length - 3}</span>
                      {/if}
                    </div>
                  {/if}
                  
                  <div class="flex items-center justify-between pt-3 border-t border-gray-100">
                    <div class="text-[11px] text-gray-400">
                      {#if tmpl.author}by {tmpl.author}{/if}
                    </div>
                    {#if pullingTemplates[tmpl.name]}
                      <span class="inline-flex items-center gap-2 px-3 py-1.5 text-[12px] font-medium text-amber-600">
                        <span class="w-3 h-3 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></span>
                        {t.pulling}
                      </span>
                    {:else if tmpl.installed && hasUpdate(tmpl)}
                      <button 
                        class="px-3 py-1.5 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                        on:click={() => handlePullTemplate(tmpl.name, true)}
                      >{t.update}</button>
                    {:else if !tmpl.installed}
                      <button 
                        class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
                        on:click={() => handlePullTemplate(tmpl.name, false)}
                      >{t.pull}</button>
                    {/if}
                  </div>
                </div>
              {:else}
                <div class="col-span-full py-16 text-center">
                  <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                  <p class="text-[13px] text-gray-400">
                    {#if registrySearch}
                      {t.noMatch}
                    {:else}
                      {t.clickRefresh}
                    {/if}
                  </p>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'ai'}
        <div class="max-w-2xl space-y-5">
          <!-- MCP Status Card -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center">
                  <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
                  </svg>
                </div>
                <div>
                  <h2 class="text-[14px] font-semibold text-gray-900">{t.mcpServer}</h2>
                  <p class="text-[12px] text-gray-500">{t.mcpDesc}</p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                {#if mcpStatus.running}
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-50 text-emerald-600 text-[12px] font-medium rounded-full">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                    {t.running}
                  </span>
                {:else}
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-50 text-gray-500 text-[12px] font-medium rounded-full">
                    <span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
                    {t.stopped}
                  </span>
                {/if}
              </div>
            </div>

            {#if mcpStatus.running}
              <!-- Running status info -->
              <div class="bg-gray-50 rounded-lg p-4 mb-4">
                <div class="grid grid-cols-2 gap-4 text-[12px]">
                  <div>
                    <span class="text-gray-500">{t.transportMode}</span>
                    <p class="font-medium text-gray-900 mt-0.5">SSE (HTTP)</p>
                  </div>
                  <div>
                    <span class="text-gray-500">{t.listenAddr}</span>
                    <p class="font-mono font-medium text-gray-900 mt-0.5">{mcpStatus.address || '-'}</p>
                  </div>
                  <div>
                    <span class="text-gray-500">{t.protocolVersion}</span>
                    <p class="font-medium text-gray-900 mt-0.5">{mcpStatus.protocolVersion}</p>
                  </div>
                  <div>
                    <span class="text-gray-500">{t.msgEndpoint}</span>
                    <p class="font-mono font-medium text-gray-900 mt-0.5 text-[11px]">http://{mcpStatus.address}/message</p>
                  </div>
                </div>
              </div>
              <button 
                class="w-full h-10 bg-red-500 text-white text-[13px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
                on:click={handleStopMCP}
                disabled={mcpLoading}
              >
                {mcpLoading ? t.stoppingServer : t.stopServer}
              </button>
            {:else}
              <!-- Configuration form -->
              <div class="space-y-4 mb-4">
                <div>
                  <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.transportMode}</label>
                  <div class="inline-flex items-center h-10 px-4 text-[13px] font-medium rounded-lg border bg-gray-900 text-white border-gray-900">
                    SSE (HTTP)
                  </div>
                </div>
                <div>
                  <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.listenAddr}</label>
                  <input 
                    type="text" 
                    placeholder="localhost:8080" 
                    class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                    bind:value={mcpForm.address} 
                  />
                </div>
              </div>
              <button 
                class="w-full h-10 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={handleStartMCP}
                disabled={mcpLoading}
              >
                {mcpLoading ? t.startingServer : t.startServer}
              </button>
            {/if}
          </div>

          <!-- MCP Info Card -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <h3 class="text-[14px] font-semibold text-gray-900 mb-3">{t.aboutMcp}</h3>
            <p class="text-[12px] text-gray-600 leading-relaxed mb-4">
              {t.mcpInfo}
            </p>
            <div class="bg-gray-50 rounded-lg p-4">
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-2">{t.availableTools}</div>
              <div class="grid grid-cols-2 gap-2 text-[12px]">
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  list_templates
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  list_cases
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  plan_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  start_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  stop_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  kill_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  get_case_status
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  exec_command
                </div>
              </div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'credentials'}
        <div class="max-w-3xl space-y-5">
          <!-- Profile مدیریت -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between mb-4">
              <div>
                <h3 class="text-[14px] font-semibold text-gray-900">{t.profileManage}</h3>
                <p class="text-[12px] text-gray-500">{t.profileHint}</p>
              </div>
              <div class="text-[12px] text-gray-500">
                {t.activeProfile}: <span class="font-medium text-gray-700">{profiles.find(p => p.id === activeProfileId)?.name || '-'}</span>
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profile}</label>
                <select
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={activeProfileId}
                  on:change={(e) => handleProfileChange(e.target.value)}
                >
                  <option value="" disabled>{t.selectProfile}</option>
                  {#each profiles as p}
                    <option value={p.id}>{p.name}</option>
                  {/each}
                </select>
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profileName}</label>
                <input
                  type="text"
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={profileForm.name}
                />
              </div>
              <div class="md:col-span-2">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.configPath}</label>
                <input
                  type="text"
                  placeholder={t.defaultPath}
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={profileForm.configPath}
                  on:input={() => { customConfigPath = profileForm.configPath; }}
                />
              </div>
              <div class="md:col-span-2">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.templateDir}</label>
                <input
                  type="text"
                  placeholder={t.defaultPath}
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={profileForm.templateDir}
                />
              </div>
            </div>

            {#if profileError}
              <div class="mt-3 text-[12px] text-red-600">{profileError}</div>
            {/if}

            <div class="mt-4 flex flex-wrap gap-2">
              <button
                class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={handleCreateProfile}
                disabled={profileSaving}
              >
                {t.createProfile}
              </button>
              <button
                class="h-9 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50"
                on:click={handleSaveProfile}
                disabled={profileSaving || !activeProfileId}
              >
                {t.saveProfile}
              </button>
              <button
                class="h-9 px-4 bg-red-50 text-red-600 text-[12px] font-medium rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
                on:click={handleDeleteProfile}
                disabled={profileSaving || !activeProfileId}
              >
                {t.deleteProfile}
              </button>
              <button
                class="h-9 px-4 bg-gray-100 text-gray-700 text-[12px] font-medium rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
                on:click={loadProvidersConfig}
                disabled={credentialsLoading}
              >
                {credentialsLoading ? t.loading : t.loadConfig}
              </button>
            </div>

            {#if providersConfig.configPath}
              <div class="mt-3 text-[12px] text-gray-500">
                {t.currentConfig}: <span class="font-mono">{providersConfig.configPath}</span>
              </div>
            {/if}
            <div class="mt-2 text-[12px] text-gray-500">
              {t.profileCredentialsFrom}: <span class="font-mono">{profileForm.configPath || providersConfig.configPath || '-'}</span>
            </div>
            <div class="text-[12px] text-gray-500">
              {t.profileTemplateFrom}: <span class="font-mono">{profileForm.templateDir || '-'}</span>
            </div>
            <div class="text-[11px] text-gray-400 mt-1">
              {t.profileSwitchHint}
            </div>
          </div>

          <!-- Security Notice -->
          <div class="flex items-start gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
            <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <div class="text-[12px] text-amber-800">
              <strong>{t.securityTip}</strong>{t.securityInfo}
            </div>
          </div>

          {#if credentialsLoading}
            <div class="flex items-center justify-center h-32">
              <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
            </div>
          {:else}
            <!-- Provider Cards -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              {#each providersConfig.providers || [] as provider}
                <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
                  <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
                    <h3 class="text-[14px] font-semibold text-gray-900">{provider.name}</h3>
                    {#if editingProvider === provider.name}
                      <div class="flex gap-2">
                        <button 
                          class="px-3 py-1 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
                          on:click={cancelEditProvider}
                        >{t.cancel}</button>
                        <button 
                          class="px-3 py-1 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
                          on:click={() => saveProviderCredentials(provider.name)}
                          disabled={credentialsSaving[provider.name]}
                        >
                          {credentialsSaving[provider.name] ? t.saving : t.save}
                        </button>
                      </div>
                    {:else}
                      <button 
                        class="px-3 py-1 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                        on:click={() => startEditProvider(provider)}
                      >{t.edit}</button>
                    {/if}
                  </div>
                  <div class="p-5 space-y-3">
                    {#each Object.entries(provider.fields) as [key, value]}
                      <div>
                        <label class="block text-[11px] font-medium text-gray-500 mb-1">
                          {getFieldLabel(key)}
                          {#if provider.hasSecrets && provider.hasSecrets[key]}
                            <span class="ml-1 text-amber-500">🔒</span>
                          {/if}
                        </label>
                        {#if editingProvider === provider.name}
                          {#if isSecretField(key)}
                            <input 
                              type="password"
                              placeholder={t.enterNew}
                              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                              bind:value={editFields[key]}
                            />
                          {:else}
                            <input 
                              type="text"
                              placeholder={value || t.notSet}
                              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                              bind:value={editFields[key]}
                            />
                          {/if}
                        {:else}
                          <div class="h-9 px-3 flex items-center text-[12px] bg-gray-50 rounded-lg font-mono {value ? 'text-gray-900' : 'text-gray-400'}">
                            {value || t.notSet}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                </div>
              {:else}
                <div class="col-span-full py-16 text-center">
                  <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                  </svg>
                  <p class="text-[13px] text-gray-400">{t.clickLoad}</p>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'localTemplates'}
        <div class="space-y-5">
          <!-- Search and Actions -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center gap-4">
              <div class="flex-1 relative">
                <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input 
                  type="text" 
                  placeholder={t.search}
                  class="w-full h-10 pl-10 pr-4 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={localTemplatesSearch} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={loadLocalTemplates}
                disabled={localTemplatesLoading}
              >
                {localTemplatesLoading ? t.loading : t.refresh}
              </button>
            </div>
          </div>

          {#if localTemplatesLoading}
            <div class="flex items-center justify-center h-64">
              <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
            </div>
          {:else}
            <!-- Template Table -->
            <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table class="w-full table-fixed">
                <thead>
                  <tr class="border-b border-gray-100">
                    <th class="text-left px-4 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[140px]">{t.name}</th>
                    <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[60px]">{t.version}</th>
                    <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[140px]">{t.author}</th>
                    <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[180px]">{t.module}</th>
                    <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[320px]">{t.description}</th>
                    <th class="text-right pl-4 pr-6 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[220px]">{t.actions}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each filteredLocalTemplates as tmpl}
                    <tr class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors">
                      <td class="px-4 py-3.5">
                        <span class="text-[13px] font-medium text-gray-900 break-all">{tmpl.name}</span>
                      </td>
                      <td class="px-3 py-3.5">
                        <span class="text-[13px] text-gray-600">{tmpl.version || '-'}</span>
                      </td>
                      <td class="px-3 py-3.5">
                        <span class="text-[13px] text-gray-600 break-words whitespace-normal block" title={tmpl.user || '-'}>{tmpl.user || '-'}</span>
                      </td>
                      <td class="px-3 py-3.5">
                        {#if tmpl.module}
                          <span class="px-2 py-0.5 bg-blue-50 text-blue-600 text-[11px] font-medium rounded-full inline-block break-words whitespace-normal max-w-full" title={tmpl.module}>{tmpl.module}</span>
                        {:else}
                          <span class="text-[13px] text-gray-400">-</span>
                        {/if}
                      </td>
                      <td class="px-3 py-3.5 w-[320px]">
                        <span class="text-[12px] text-gray-500 break-words whitespace-normal" title={tmpl.description}>{tmpl.description || '-'}</span>
                      </td>
                      <td class="pl-4 pr-6 py-3.5 text-right w-[220px]">
                        <div class="flex flex-col gap-2 items-end">
                          <div class="flex items-center gap-3">
                            <button 
                              class="w-[90px] px-2.5 py-1 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
                              on:click={() => handleCloneTemplate(tmpl)}
                            >{t.cloneTemplate}</button>
                            <button 
                              class="w-[90px] px-2.5 py-1 text-[12px] font-medium text-indigo-700 bg-indigo-50 rounded-md hover:bg-indigo-100 transition-colors"
                              on:click={() => openTemplateEditor(tmpl)}
                            >{t.editTemplate}</button>
                          </div>
                          <div class="flex items-center gap-3">
                            <button 
                              class="w-[90px] px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                              on:click={() => showTemplateDetail(tmpl)}
                            >{t.viewParams}</button>
                            {#if deletingTemplate[tmpl.name]}
                              <span class="w-[90px] px-2.5 py-1 text-[12px] font-medium text-amber-600">{t.deleting}</span>
                            {:else}
                              <button 
                                class="w-[90px] px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                                on:click={() => showDeleteTemplateConfirm(tmpl.name)}
                              >{t.delete}</button>
                            {/if}
                          </div>
                        </div>
                      </td>
                    </tr>
                  {:else}
                    <tr>
                      <td colspan="6" class="py-16">
                        <div class="flex flex-col items-center text-gray-400">
                          <svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
                          </svg>
                          <p class="text-[13px]">{t.noLocalTemplates}</p>
                          <button 
                            class="mt-2 text-[12px] text-blue-600 hover:underline"
                            on:click={() => { activeTab = 'registry'; loadRegistryTemplates(); }}
                          >{t.goToRegistry}</button>
                        </div>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/if}
    </main>
  </div>
</div>

<!-- Delete Confirmation Modal -->
{#if deleteConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelDelete}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
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
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelDelete}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          on:click={confirmDelete}
        >{t.delete}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Template Confirmation Modal -->
{#if deleteTemplateConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelDeleteTemplate}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmDelete}</h3>
            <p class="text-[13px] text-gray-500">{t.deleteWarning}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmDeleteTemplate} <span class="font-medium text-gray-900">"{deleteTemplateConfirm.name}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelDeleteTemplate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          on:click={confirmDeleteTemplate}
        >{t.delete}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Clone Template Modal -->
{#if cloneTemplateModal.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelCloneTemplate}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16h8M8 12h8m-6 8h6a2 2 0 002-2V8a2 2 0 00-2-2h-2l-2-2H8a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.cloneTitle}</h3>
            <p class="text-[13px] text-gray-500">{t.cloneHint}</p>
          </div>
        </div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.cloneName}</label>
        <input
          type="text"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={cloneTemplateModal.target}
        />
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelCloneTemplate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors"
          on:click={confirmCloneTemplate}
        >{t.cloneTemplate}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Template Detail Drawer -->
{#if localTemplateDetail}
  <div class="fixed inset-0 bg-black/50 flex justify-end z-50" on:click={closeTemplateDetail}>
    <div class="w-full max-w-2xl bg-white h-full overflow-auto shadow-xl" on:click|stopPropagation>
      <div class="sticky top-0 bg-white border-b border-gray-100 px-6 py-4 flex items-center justify-between">
        <div>
          <h2 class="text-[16px] font-semibold text-gray-900">{localTemplateDetail.name}</h2>
          <p class="text-[12px] text-gray-500 mt-0.5">v{localTemplateDetail.version || '-'}</p>
        </div>
        <button 
          class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          on:click={closeTemplateDetail}
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div class="p-6 space-y-6">
        <!-- Template Info -->
        <div class="space-y-3">
          {#if localTemplateDetail.description}
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.description}</div>
              <p class="text-[13px] text-gray-700">{localTemplateDetail.description}</p>
            </div>
          {/if}
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.author}</div>
              <p class="text-[13px] text-gray-900">{localTemplateDetail.user || '-'}</p>
            </div>
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.module}</div>
              {#if localTemplateDetail.module}
                <span class="px-2 py-0.5 bg-blue-50 text-blue-600 text-[12px] font-medium rounded-full">{localTemplateDetail.module}</span>
              {:else}
                <p class="text-[13px] text-gray-400">-</p>
              {/if}
            </div>
          </div>
        </div>

        <!-- Template Parameters -->
        <div>
          <div class="text-[14px] font-semibold text-gray-900 mb-3">{t.templateParams}</div>
          {#if localTemplateVarsLoading}
            <div class="flex items-center justify-center py-8">
              <div class="w-5 h-5 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
              <span class="ml-2 text-[13px] text-gray-500">{t.loadingParams}</span>
            </div>
          {:else if localTemplateVars.length === 0}
            <div class="py-8 text-center text-[13px] text-gray-400">
              {t.noParams}
            </div>
          {:else}
            <div class="border border-gray-100 rounded-lg overflow-x-auto">
              <table class="w-full text-[12px] min-w-[520px]">
                <thead>
                  <tr class="bg-gray-50 border-b border-gray-100">
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramName}</th>
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramType}</th>
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramDefault}</th>
                    <th class="text-center px-4 py-2.5 font-semibold text-gray-600">{t.paramRequired}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each localTemplateVars as v}
                    <tr class="border-b border-gray-50 hover:bg-gray-50/50">
                      <td class="px-4 py-3">
                        <div class="font-medium text-gray-900">{v.name}</div>
                        {#if v.description}
                          <div class="text-[11px] text-gray-500 mt-0.5">{v.description}</div>
                        {/if}
                      </td>
                      <td class="px-4 py-3">
                        <code class="px-1.5 py-0.5 bg-gray-100 text-gray-700 rounded text-[11px]">{v.type}</code>
                      </td>
                      <td class="px-4 py-3">
                        {#if v.defaultValue}
                          <code class="text-gray-600">{v.defaultValue}</code>
                        {:else}
                          <span class="text-gray-400">-</span>
                        {/if}
                      </td>
                      <td class="px-4 py-3 text-center">
                        {#if v.required}
                          <span class="inline-flex items-center justify-center w-5 h-5 bg-emerald-100 text-emerald-600 rounded-full">
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                          </span>
                        {:else}
                          <span class="inline-flex items-center justify-center w-5 h-5 bg-gray-100 text-gray-400 rounded-full">
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14" />
                            </svg>
                          </span>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Template Editor Modal -->
{#if templateEditor.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={closeTemplateEditor}>
    <div class="bg-white rounded-xl shadow-xl max-w-4xl w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.editTemplate}</h3>
          <p class="text-[12px] text-gray-500">{templateEditor.name}</p>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
            on:click={closeTemplateEditor}
          >{t.close}</button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
            on:click={saveTemplateEditor}
            disabled={templateEditor.saving}
          >{templateEditor.saving ? t.saving : t.saveTemplate}</button>
        </div>
      </div>
      <div class="flex h-[520px]">
        <div class="w-52 border-r border-gray-100 overflow-auto">
          <div class="px-4 py-3 text-[12px] font-semibold text-gray-600">{t.templateFiles}</div>
          {#each Object.keys(templateEditor.files) as fname}
            <button
              class="w-full text-left px-4 py-2 text-[12px] transition-colors {templateEditor.active === fname ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
              on:click={() => templateEditor = { ...templateEditor, active: fname }}
            >{fname}</button>
          {/each}
        </div>
        <div class="flex-1 p-4">
          {#if templateEditor.error}
            <div class="text-[12px] text-red-500 mb-2">{templateEditor.error}</div>
          {/if}
          {#if templateEditor.active}
            <textarea
              class="w-full h-full text-[12px] font-mono bg-gray-50 border border-gray-100 rounded-lg p-3 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
              bind:value={templateEditor.files[templateEditor.active]}
            ></textarea>
          {:else}
            <div class="text-[12px] text-gray-400">{t.noParams}</div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

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
