<script>
  import { onMount } from 'svelte';
  import { GetOnboardingStatus, SetOnboardingDismissed } from '../../../wailsjs/go/main/App.js';

  let { t, onTabChange = () => {} } = $props();

  let status = $state({ credentialsConfigured: false, templatesInstalled: false, scenesCreated: false, dismissed: false });
  let loading = $state(true);

  let completedCount = $derived(
    (status.credentialsConfigured ? 1 : 0) +
    (status.templatesInstalled ? 1 : 0) +
    (status.scenesCreated ? 1 : 0)
  );

  let allComplete = $derived(completedCount === 3);

  let currentStep = $derived(
    !status.credentialsConfigured ? 1 :
    !status.templatesInstalled ? 2 :
    !status.scenesCreated ? 3 : 3
  );

  const steps = $derived([
    { done: status.credentialsConfigured, title: t.gsStep1Title || '配置云厂商凭据', desc: t.gsStep1Desc || '在凭据管理中配置至少一个云厂商的 API 凭据', doneText: t.gsStep1Done || '凭据已配置', action: t.gsGoCredentials || '前往凭据管理', tab: 'credentials' },
    { done: status.templatesInstalled, title: t.gsStep2Title || '下载场景模板', desc: t.gsStep2Desc || '从模板仓库下载预定义的云基础设施模板', doneText: t.gsStep2Done || '模板已下载', action: t.gsGoRegistry || '前往模板仓库', tab: 'registry' },
    { done: status.scenesCreated, title: t.gsStep3Title || '创建你的第一个场景', desc: t.gsStep3Desc || '基于模板一键部署云基础设施', doneText: t.gsStep3Done || '场景已创建', action: t.gsGoCases || '前往场景管理', tab: 'cases' },
  ]);

  onMount(loadStatus);

  async function loadStatus() {
    try {
      loading = true;
      status = await GetOnboardingStatus();
    } catch (e) {
      console.error('Failed to load onboarding status:', e);
      status.dismissed = true;
    } finally {
      loading = false;
    }
  }

  async function handleDismiss() {
    try {
      await SetOnboardingDismissed();
      status.dismissed = true;
    } catch (e) {
      console.error('Failed to dismiss onboarding:', e);
    }
  }

  export function refresh() {
    loadStatus();
  }
</script>

{#if !loading && !status.dismissed}
  {#if allComplete}
    <div class="bg-gradient-to-r from-green-50 to-emerald-50 border border-green-200 rounded-xl px-5 py-3.5 flex items-center justify-between mb-3">
      <div class="flex items-center gap-2.5">
        <div class="w-5 h-5 rounded-full bg-emerald-500 flex items-center justify-center flex-shrink-0">
          <svg class="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        </div>
        <span class="text-[13px] text-green-800 font-medium">{t.gsComplete || '设置完成！你已准备好使用 RedC'}</span>
      </div>
      <button class="text-green-300 hover:text-green-500 transition-colors cursor-pointer" onclick={handleDismiss} aria-label="关闭">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {:else}
    <div class="bg-gradient-to-br from-sky-50 to-blue-50 border border-sky-200 rounded-xl p-5 mb-3 relative">
      <button
        class="absolute top-3 right-3 text-sky-300 hover:text-sky-500 transition-colors cursor-pointer"
        onclick={handleDismiss}
        aria-label="关闭引导"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <div class="flex items-center gap-2 mb-3.5">
        <svg class="w-5 h-5 text-sky-500" fill="currentColor" viewBox="0 0 20 20">
          <path d="M10 2l2.9 6.1 6.6.8-4.75 4.6 1.05 6.5L10 16.9 4.2 20l1.05-6.5L.5 8.9l6.6-.8L10 2z"/>
        </svg>
        <span class="text-[15px] font-semibold text-sky-900">{t.gsTitle || '快速上手'}</span>
        <span class="text-[11px] text-sky-600 bg-sky-100 px-2 py-0.5 rounded-full">{completedCount}/3 {t.gsProgress || '已完成'}</span>
      </div>

      <div class="space-y-0">
        {#each steps as step, i}
          {@const stepNum = i + 1}
          {@const isCurrent = !step.done && stepNum === currentStep}
          <div class="flex items-start gap-3 py-2.5 {i < 2 ? 'border-b border-sky-100' : ''} {isCurrent ? 'bg-white/50 -mx-3 px-3 rounded-lg' : ''}">
            {#if step.done}
              <div class="w-[22px] h-[22px] rounded-full bg-emerald-500 flex items-center justify-center flex-shrink-0 mt-0.5">
                <svg class="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                </svg>
              </div>
            {:else}
              <div class="w-[22px] h-[22px] rounded-full border-2 {isCurrent ? 'border-sky-500 bg-white' : 'border-gray-300 bg-white'} flex items-center justify-center flex-shrink-0 mt-0.5">
                <span class="text-[11px] font-semibold {isCurrent ? 'text-sky-500' : 'text-gray-400'}">{stepNum}</span>
              </div>
            {/if}

            <div class="flex-1 min-w-0">
              <div class="text-[13px] {step.done ? 'text-gray-400 line-through' : isCurrent ? 'font-medium text-sky-900' : 'text-gray-400'}">
                {step.title}
              </div>
              {#if step.done}
                <div class="text-[12px] text-gray-400 mt-0.5">{step.doneText}</div>
              {:else if isCurrent}
                <div class="text-[12px] text-gray-500 mt-0.5">{step.desc}</div>
              {/if}
            </div>

            {#if isCurrent}
              <button
                class="text-[12px] text-sky-500 border border-sky-300 hover:bg-sky-50 px-3 py-1 rounded-md cursor-pointer whitespace-nowrap transition-colors flex-shrink-0"
                onclick={() => onTabChange(step.tab)}
              >
                {step.action} →
              </button>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}
{/if}
