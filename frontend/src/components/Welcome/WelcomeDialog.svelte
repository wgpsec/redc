<script lang="ts">
  import { SetShowWelcomeDialog } from '../../../wailsjs/go/main/App.js';
  import Modal from '../UI/Modal.svelte';

  let { t, show = false, onClose } = $props();
  
  let dontShowAgain = $state(false);
  let currentPage = $state(0);

  $effect(() => {
    if (show) {
      dontShowAgain = false;
      currentPage = 0;
    }
  });

  async function handleClose() {
    if (dontShowAgain) {
      try {
        await SetShowWelcomeDialog(false);
      } catch (e) {
        console.error('Failed to save welcome dialog setting:', e);
      }
    }
    onClose?.();
  }

  function nextPage() {
    if (currentPage < 3) {
      currentPage += 1;
    }
  }

  function prevPage() {
    if (currentPage > 0) {
      currentPage -= 1;
    }
  }
</script>

<Modal show={show} onclose={handleClose}>
    <!-- Dialog -->
    <div class="relative bg-white rounded-xl border border-gray-100 max-w-md w-full mx-4 overflow-hidden">
      <!-- Header -->
      <div class="px-5 py-4 border-b border-gray-100">
        <h2 class="text-[15px] font-medium text-gray-900">
          {currentPage === 0 ? (t.welcomeTitle || '欢迎使用 RedC') : currentPage === 1 ? (t.welcomeEcosystemTitle || 'WgpSec Infra 生态') : currentPage === 2 ? (t.welcomeWindowsIssue || 'Windows 控制台窗口说明') : (t.welcomeProxyTitle || '代理配置说明')}
        </h2>
      </div>
      
      <!-- Content - Page 0: Features -->
      {#if currentPage === 0}
        <div class="px-5 py-4 space-y-3">
          <!-- Feature 1 -->
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-blue-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.welcomeFeature1 || '一键部署云资源'}</h3>
              <p class="text-[12px] text-gray-500">{t.welcomeFeature1Desc || '支持多种云厂商，快速创建和管理云资源场景'}</p>
            </div>
          </div>
          
          <!-- Feature 2 -->
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-emerald-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.welcomeFeature2 || '成本优化分析'}</h3>
              <p class="text-[12px] text-gray-500">{t.welcomeFeature2Desc || 'AI 智能分析资源使用，提供成本优化建议'}</p>
            </div>
          </div>
          
          <!-- Feature 3 -->
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-gray-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.welcomeFeature3 || '本地模板管理'}</h3>
              <p class="text-[12px] text-gray-500">{t.welcomeFeature3Desc || '支持自定义模板，满足个性化需求'}</p>
            </div>
          </div>
        </div>
      {:else if currentPage === 1}
        <!-- Ecosystem page -->
        <div class="px-5 py-4 space-y-3">
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-red-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.ecosystemRedcName || 'RedC'}</h3>
              <p class="text-[12px] text-gray-500">{t.ecosystemRedcDesc || '控制面 — 编排和管理云基础设施的 GUI 工具'}</p>
            </div>
          </div>
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-amber-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.ecosystemTemplateName || 'redc-template'}</h3>
              <p class="text-[12px] text-gray-500">{t.ecosystemTemplateDesc || '场景仓库 — 提供预配置的多云 Terraform 模板'}</p>
            </div>
          </div>
          <div class="flex gap-3">
            <div class="flex-shrink-0 w-8 h-8 bg-cyan-50 rounded-lg flex items-center justify-center">
              <svg class="w-4 h-4 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.384 3.183A2.25 2.25 0 013 16.268V7.732a2.25 2.25 0 013.036-2.085l5.384 3.183m0 0l5.384-3.183A2.25 2.25 0 0119.8 7.732v8.536a2.25 2.25 0 01-2.996 2.085l-5.384-3.183z" />
              </svg>
            </div>
            <div>
              <h3 class="text-[13px] font-medium text-gray-900">{t.ecosystemF8xName || 'f8x'}</h3>
              <p class="text-[12px] text-gray-500">{t.ecosystemF8xDesc || '装配引擎 — 在目标主机上自动安装工具和配置环境'}</p>
            </div>
          </div>
          <div class="pt-2 border-t border-gray-100">
            <p class="text-[11px] text-gray-400">{t.ecosystemFlow || 'RedC 拉取模板 → 部署云资源 → f8x 装配环境，全程自动化'}</p>
          </div>
        </div>
      {:else if currentPage === 2}
        <!-- Content - Page 2: Windows Issue -->
        <div class="px-5 py-4">
          <div class="bg-amber-50 border border-amber-100 rounded-lg p-4">
            <div class="flex gap-3">
              <div class="flex-shrink-0 w-8 h-8 bg-amber-100 rounded-lg flex items-center justify-center">
                <svg class="w-4 h-4 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <div>
                <h3 class="text-[13px] font-medium text-amber-800">{t.welcomeWindowsIssueTitle || 'Windows 用户请注意'}</h3>
                <p class="text-[12px] text-amber-700 mt-1">{t.welcomeWindowsIssueDesc || 'Windows 系统中的控制台窗口问题说明'}</p>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <!-- Content - Page 3: Proxy Config -->
        <div class="px-5 py-4 space-y-3">
          <div class="bg-blue-50 border border-blue-100 rounded-lg p-4">
            <div class="flex gap-3">
              <div class="flex-shrink-0 w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center">
                <svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z" />
                </svg>
              </div>
              <div>
                <h3 class="text-[13px] font-medium text-blue-800">{t.welcomeProxyTitle || '代理配置说明'}</h3>
                <p class="text-[12px] text-blue-700 mt-1">{t.welcomeProxyDesc || '由于中国大陆网络环境原因，建议配置代理以提高云厂商 API 连接速度'}</p>
              </div>
            </div>
          </div>

          <div class="space-y-2">
            <p class="text-[12px] text-gray-500">{t.welcomeProxyWhere || '在哪里配置：'}</p>
            <ul class="text-[12px] text-gray-600 space-y-1 list-disc list-inside">
              <li>{t.welcomeProxyPath || '设置 → 代理配置'}</li>
              <li>{t.welcomeProxyEffect || '配置后将用于 Terraform 的网络请求和模板下载'}</li>
            </ul>
          </div>
        </div>
      {/if}
      
      <!-- Footer -->
      <div class="px-5 py-3 border-t border-gray-100 flex items-center justify-between">
        {#if currentPage === 0}
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              bind:checked={dontShowAgain}
              class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1"
            />
            <span class="text-[12px] text-gray-500">{t.welcomeDontShow || '下次不显示'}</span>
          </label>
          <button
            class="h-9 px-4 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors"
            onclick={nextPage}
          >
            {t.welcomeNext || '下一步'}
          </button>
        {:else if currentPage < 3}
          <button
            class="h-9 px-4 text-gray-700 bg-white border border-gray-300 text-[13px] font-medium rounded-lg hover:bg-gray-50 transition-colors"
            onclick={prevPage}
          >
            {t.welcomePrev || '上一步'}
          </button>
          <button
            class="h-9 px-4 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors"
            onclick={nextPage}
          >
            {t.welcomeNext || '下一步'}
          </button>
        {:else}
          <button
            class="h-9 px-4 text-gray-700 bg-white border border-gray-300 text-[13px] font-medium rounded-lg hover:bg-gray-50 transition-colors"
            onclick={prevPage}
          >
            {t.welcomePrev || '上一步'}
          </button>
          <button
            class="h-9 px-4 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors"
            onclick={handleClose}
          >
            {t.welcomeGotIt || '知道了'}
          </button>
        {/if}
      </div>
    </div>
</Modal>
