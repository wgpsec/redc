<script>
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

  let { t } = $props();

  function openLink(url) {
    BrowserOpenURL(url);
  }
</script>

<div class="max-w-4xl mx-auto">
  <!-- Header -->
  <div class="bg-white rounded-xl border border-gray-100 p-8 mb-6">
    <div class="flex items-center gap-4 mb-6">
      <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-rose-500 to-red-600 flex items-center justify-center shadow-lg">
        <span class="text-white text-3xl font-bold">C</span>
      </div>
      <div>
        <h1 class="text-2xl font-bold text-gray-900 mb-1">RedC</h1>
        <p class="text-sm text-gray-500">Red Team Cloud Infrastructure Management Platform</p>
      </div>
    </div>
    
    <div class="flex items-center gap-3 text-sm text-gray-600 mb-3">
      <span class="px-3 py-1 bg-gray-100 rounded-full font-medium">v2.3.0</span>
    </div>
    
    <div class="text-sm text-gray-600">
      <div class="mb-1">{t.developedBy || '开发者'}:</div>
      <div class="flex flex-wrap gap-3">
        <button
          class="flex items-center gap-2 px-3 py-1.5 bg-gray-50 hover:bg-gray-100 rounded-lg transition-colors"
          onclick={() => openLink('https://github.com/No-Github')}
        >
          <svg class="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
          <span class="font-medium">r0fus0d</span>
        </button>
        <button
          class="flex items-center gap-2 px-3 py-1.5 bg-gray-50 hover:bg-gray-100 rounded-lg transition-colors"
          onclick={() => openLink('https://github.com/keac')}
        >
          <svg class="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
          </svg>
          <span class="font-medium">keac</span>
        </button>
      </div>
    </div>
  </div>

  <!-- Introduction -->
  <div class="bg-white rounded-xl border border-gray-100 p-6 mb-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{t.aboutIntro || '项目简介'}</h2>
    <div class="space-y-3 text-sm text-gray-600 leading-relaxed">
      <p>
        {t.aboutDesc1 || 'RedC 是一个专为红队设计的云基础设施管理平台，旨在简化和自动化红队在云环境中的基础设施部署和管理工作。'}
      </p>
      <p>
        {t.aboutDesc2 || '通过 RedC，您可以快速部署各种红队场景，包括 C2 服务器、钓鱼平台、漏洞环境等，支持多云平台（阿里云、腾讯云、华为云、火山云等）。'}
      </p>
      <p>
        {t.aboutDesc3 || 'RedC 基于 Terraform 构建，提供了友好的图形界面，让您无需深入了解 Terraform 语法即可轻松管理云基础设施。'}
      </p>
    </div>
  </div>

  <!-- Features -->
  <div class="bg-white rounded-xl border border-gray-100 p-6 mb-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{t.coreFeatures || '核心特性'}</h2>
    <div class="grid grid-cols-2 gap-4">
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-emerald-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.multiCloud || '多云支持'}</h3>
          <p class="text-xs text-gray-500">{t.multiCloudDesc || '支持阿里云、腾讯云、华为云、火山云等主流云平台'}</p>
        </div>
      </div>
      
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-blue-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.quickDeploy || '快速部署'}</h3>
          <p class="text-xs text-gray-500">{t.quickDeployDesc || '一键部署各种红队场景，节省时间和精力'}</p>
        </div>
      </div>
      
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-purple-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.templateManage || '模板管理'}</h3>
          <p class="text-xs text-gray-500">{t.templateManageDesc || '丰富的模板库，支持自定义和分享模板'}</p>
        </div>
      </div>
      
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-amber-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.scheduledTasks || '定时任务'}</h3>
          <p class="text-xs text-gray-500">{t.scheduledTasksFeatureDesc || '支持定时启动和停止场景，自动化管理'}</p>
        </div>
      </div>
      
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-rose-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.costEstimate || '成本估算'}</h3>
          <p class="text-xs text-gray-500">{t.costEstimateFeatureDesc || '实时估算云资源成本，控制预算'}</p>
        </div>
      </div>
      
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-indigo-50 flex items-center justify-center flex-shrink-0">
          <svg class="w-4 h-4 text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
          </svg>
        </div>
        <div>
          <h3 class="text-sm font-medium text-gray-900 mb-1">{t.aiIntegration || 'AI 集成'}</h3>
          <p class="text-xs text-gray-500">{t.aiIntegrationFeatureDesc || '支持 MCP 协议，与 AI 助手无缝集成'}</p>
        </div>
      </div>
    </div>
  </div>

  <!-- Links -->
  <div class="bg-white rounded-xl border border-gray-100 p-6 mb-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{t.links || '相关链接'}</h2>
    <div class="space-y-3">
      <button
        class="w-full flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors group"
        onclick={() => openLink('https://github.com/wgpsec/redc')}
      >
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-gray-900 flex items-center justify-center">
            <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
            </svg>
          </div>
          <div class="text-left">
            <div class="text-sm font-medium text-gray-900">GitHub</div>
            <div class="text-xs text-gray-500">github.com/wgpsec/redc</div>
          </div>
        </div>
        <svg class="w-5 h-5 text-gray-400 group-hover:text-gray-600 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
      </button>

      <button
        class="w-full flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors group"
        onclick={() => openLink('https://redc.wgpsec.org')}
      >
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-rose-500 to-red-600 flex items-center justify-center">
            <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
            </svg>
          </div>
          <div class="text-left">
            <div class="text-sm font-medium text-gray-900">{t.documentation || '文档'}</div>
            <div class="text-xs text-gray-500">redc.wgpsec.org</div>
          </div>
        </div>
        <svg class="w-5 h-5 text-gray-400 group-hover:text-gray-600 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
      </button>

      <button
        class="w-full flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors group"
        onclick={() => openLink('https://www.wgpsec.org')}
      >
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-blue-500 flex items-center justify-center">
            <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
            </svg>
          </div>
          <div class="text-left">
            <div class="text-sm font-medium text-gray-900">WgpSec {t.team || '团队'}</div>
            <div class="text-xs text-gray-500">www.wgpsec.org</div>
          </div>
        </div>
        <svg class="w-5 h-5 text-gray-400 group-hover:text-gray-600 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
      </button>
    </div>
  </div>

  <!-- License -->
  <div class="bg-white rounded-xl border border-gray-100 p-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-4">{t.license || '开源协议'}</h2>
    <div class="flex items-center gap-3 text-sm text-gray-600">
      <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <span>{t.licenseDesc || 'RedC 采用 MIT 协议开源，您可以自由使用、修改和分发。'}</span>
    </div>
  </div>
</div>
