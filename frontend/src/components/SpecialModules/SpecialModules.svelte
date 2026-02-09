<script>
  let { t } = $props();
  let specialModuleTab = $state('vulhub');
</script>

<div class="w-full max-w-4xl mx-auto">
  <div class="flex flex-wrap gap-3 sm:gap-4 mb-6">
    <button
      class="flex-1 min-w-[100px] px-3 sm:px-4 py-2.5 rounded-lg text-[13px] font-medium transition-all
        {specialModuleTab === 'vulhub' ? 'bg-gray-900 text-white' : 'bg-white text-gray-600 hover:bg-gray-50 border border-gray-200'}"
      onclick={() => specialModuleTab = 'vulhub'}
    >
      {t.vulhubSupport}
    </button>
    <button
      class="flex-1 min-w-[100px] px-3 sm:px-4 py-2.5 rounded-lg text-[13px] font-medium transition-all
        {specialModuleTab === 'c2' ? 'bg-gray-900 text-white' : 'bg-white text-gray-600 hover:bg-gray-50 border border-gray-200'}"
      onclick={() => specialModuleTab = 'c2'}
    >
      {t.c2Scenes}
    </button>
    <button
      class="flex-1 min-w-[100px] px-3 sm:px-4 py-2.5 rounded-lg text-[13px] font-medium transition-all
        {specialModuleTab === 'ai' ? 'bg-gray-900 text-white' : 'bg-white text-gray-600 hover:bg-gray-50 border border-gray-200'}"
      onclick={() => specialModuleTab = 'ai'}
    >
      {t.aiScenes}
    </button>
  </div>

  {#if specialModuleTab === 'vulhub'}
    <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-6 md:p-8 text-center">
      <div class="w-14 h-14 sm:w-16 sm:h-16 mx-auto mb-3 sm:mb-4 rounded-full bg-gradient-to-br from-orange-400 to-red-500 flex items-center justify-center">
        <svg class="w-7 h-7 sm:w-8 sm:h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 2.754 1.626 2.754H9.75c0 1.427.502 2.672 1.327 3.376.825.704 1.626 1.626 1.626 1.626H19.5c1.409 0 2.5-1.122 2.5-2.5 0-.826-.217-1.626-1.626-2.5l-9.303-3.376zM12 15.75h.007v.008H12v-.008zM7.5 15.75h.007v.008H7.5v-.008zm4.5 0h.007v.008h-.007v-.008zm4.5 0h.007v.008h-.007v-.008z" />
        </svg>
      </div>
      <h3 class="text-[16px] sm:text-[18px] font-semibold text-gray-900 mb-2">{t.vulhubSupport}</h3>
      <p class="text-[13px] sm:text-[14px] text-gray-500 mb-4 sm:mb-6">Vulhub 漏洞环境支持模块</p>
      <div class="bg-gray-50 rounded-lg p-4 sm:p-6 text-left">
        <p class="text-[12px] sm:text-[13px] text-gray-600">此模块功能开发中...</p>
      </div>
    </div>
  {:else if specialModuleTab === 'c2'}
    <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-6 md:p-8 text-center">
      <div class="w-14 h-14 sm:w-16 sm:h-16 mx-auto mb-3 sm:mb-4 rounded-full bg-gradient-to-br from-blue-500 to-cyan-500 flex items-center justify-center">
        <svg class="w-7 h-7 sm:w-8 sm:h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
        </svg>
      </div>
      <h3 class="text-[16px] sm:text-[18px] font-semibold text-gray-900 mb-2">{t.c2Scenes}</h3>
      <p class="text-[13px] sm:text-[14px] text-gray-500 mb-4 sm:mb-6">C2 场景管理模块</p>
      <div class="bg-gray-50 rounded-lg p-4 sm:p-6 text-left">
        <p class="text-[12px] sm:text-[13px] text-gray-600">此模块功能开发中...</p>
      </div>
    </div>
  {:else if specialModuleTab === 'ai'}
    <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-6 md:p-8 text-center">
      <div class="w-14 h-14 sm:w-16 sm:h-16 mx-auto mb-3 sm:mb-4 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
        <svg class="w-7 h-7 sm:w-8 sm:h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
        </svg>
      </div>
      <h3 class="text-[16px] sm:text-[18px] font-semibold text-gray-900 mb-2">{t.aiScenes}</h3>
      <p class="text-[13px] sm:text-[14px] text-gray-500 mb-4 sm:mb-6">AI 场景管理模块</p>
      <div class="bg-gray-50 rounded-lg p-4 sm:p-6 text-left">
        <p class="text-[12px] sm:text-[13px] text-gray-600">此模块功能开发中...</p>
      </div>
    </div>
  {/if}
</div>
