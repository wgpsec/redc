<script>
  import { onMount, onDestroy } from 'svelte';
  import { editorTheme } from './theme.js';
  import { EditorView, basicSetup } from 'codemirror';
  import { EditorState } from '@codemirror/state';
  import { indentUnit } from '@codemirror/language';
  import { keymap } from '@codemirror/view';
  import { indentWithTab } from '@codemirror/commands';
  import { terraform } from 'codemirror-lang-terraform';

  // Props
  let { 
    filename = '', 
    value = $bindable(''), 
    readonly = false,
    onchange
  } = $props();

  // Internal state
  let editorContainer = $state();
  let editorView = $state(null);
  let useFallbackEditor = $state(false);
  let isLoading = $state(false);
  let loadError = $state(null);
  let fallbackValue = $state(value);

  /**
   * Check if a file is a Terraform file based on its extension
   * @param {string} filename - The filename to check
   * @returns {boolean} True if the file is a Terraform file (.tf or .tfvars)
   */
  function isTerraformFile(filename) {
    if (!filename) return false;
    return filename.endsWith('.tf') || filename.endsWith('.tfvars');
  }

  /**
   * Create the editor extensions based on file type
   * @param {string} filename - The current filename
   * @returns {Array} Array of CodeMirror extensions
   */
  function createExtensions(filename) {
    const extensions = [
      basicSetup,
      editorTheme, // Apply custom theme
      indentUnit.of('  '), // Configure indentation to use 2 spaces
      keymap.of([indentWithTab]), // Configure Tab key to insert spaces
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          const newContent = update.state.doc.toString();
          handleContentChange(newContent);
        }
      }),
    ];

    // Add Terraform syntax highlighting for .tf and .tfvars files
    if (isTerraformFile(filename)) {
      try {
        extensions.push(terraform());
      } catch (error) {
        console.error('Failed to load Terraform syntax highlighting:', error);
        // Continue without syntax highlighting
      }
    }

    // Add readonly configuration if needed
    if (readonly) {
      extensions.push(EditorState.readOnly.of(true));
    }

    return extensions;
  }

  /**
   * Initialize the CodeMirror editor
   */
  async function initializeEditor() {
    if (!editorContainer) {
      console.error('Editor container not found');
      return;
    }

    try {
      const extensions = createExtensions(filename);

      const state = EditorState.create({
        doc: value || '',
        extensions,
      });

      editorView = new EditorView({
        state,
        parent: editorContainer,
      });
    } catch (error) {
      console.error('Failed to initialize CodeMirror editor:', error);
      loadError = error.message || 'Failed to initialize editor';
      useFallbackEditor = true;
      // Fallback will be handled by the template
    }
  }

  /**
   * Update the editor when filename changes (re-initialize with new syntax)
   * @param {string} newFilename - The new filename
   */
  function updateEditorForFilename(newFilename) {
    if (!editorView) return;

    try {
      const currentContent = editorView.state.doc.toString();
      const extensions = createExtensions(newFilename);

      const newState = EditorState.create({
        doc: currentContent,
        extensions,
      });

      editorView.setState(newState);
    } catch (error) {
      console.error('Failed to update editor for filename change:', error);
    }
  }

  /**
   * Update the editor content when value prop changes
   * @param {string} newValue - The new content value
   */
  function updateEditorContent(newValue) {
    if (!editorView) return;

    const currentContent = editorView.state.doc.toString();
    if (currentContent !== newValue) {
      try {
        editorView.dispatch({
          changes: {
            from: 0,
            to: currentContent.length,
            insert: newValue || '',
          },
        });
      } catch (error) {
        console.error('Failed to update editor content:', error);
      }
    }
  }

  /**
   * Lifecycle: Initialize the editor when component mounts
   */
  onMount(() => {
    initializeEditor();
  });

  /**
   * Lifecycle: Clean up editor instance when component is destroyed
   */
  onDestroy(() => {
    if (editorView) {
      editorView.destroy();
      editorView = null;
    }
  });

  /**
   * Handle content changes from the editor
   * @param {string} newContent - The updated content from the editor
   */
  function handleContentChange(newContent) {
    value = newContent;
    onchange?.(newContent);
  }

  /**
   * React to prop changes (filename or value)
   */
  $effect(() => {
	if (editorView && filename) {
    updateEditorForFilename(filename);
  }
});

  $effect(() => {
	if (editorView && value !== undefined) {
    updateEditorContent(value);
  }
});

  // Sync fallback value with prop value
  $effect(() => {
    if (useFallbackEditor) {
      fallbackValue = value;
    }
  });

</script>

<div class="code-editor-wrapper">
  {#if isLoading}
    <!-- Loading indicator -->
    <div class="loading-container w-full h-full flex items-center justify-center bg-gray-50 border border-gray-300 rounded-md">
      <div class="text-center">
        <div class="loading-spinner mb-2"></div>
        <p class="text-gray-600 text-sm">Loading editor...</p>
      </div>
    </div>
  {:else if useFallbackEditor}
    <!-- Fallback textarea if CodeMirror fails to load -->
    <div class="fallback-container w-full h-full">
      {#if loadError}
        <div class="error-message bg-yellow-50 border border-yellow-200 rounded-md p-2 mb-2">
          <p class="text-yellow-800 text-xs">
            <strong>Editor loading failed:</strong> {loadError}
          </p>
          <p class="text-yellow-700 text-xs mt-1">Using fallback text editor.</p>
        </div>
      {/if}
      <textarea
        bind:value={fallbackValue}
        oninput={() => handleContentChange(fallbackValue)}
        {readonly}
        class="w-full h-full border border-gray-300 rounded-md p-2 bg-gray-50 font-mono text-xs"
        placeholder="Enter your code here..."
      ></textarea>
    </div>
  {:else}
    <!-- Editor container with Tailwind CSS classes for consistent styling -->
    <div
      bind:this={editorContainer}
      class="code-editor-container w-full h-full border border-gray-300 rounded-md overflow-hidden bg-gray-50"
      class:readonly
    ></div>
  {/if}
</div>

<style>

  .code-editor-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .loading-container {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .loading-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #e5e7eb;
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .fallback-container {
    display: flex;
    flex-direction: column;
  }

  .fallback-container textarea {
    flex: 1;
    min-height: 0;
    resize: none;
  }

  .error-message {
    flex-shrink: 0;
  }

  .code-editor-container {
    flex: 1;
    min-height: 0;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
    font-size: 12px;
    line-height: 1.5;
  }

  .code-editor-container.readonly {
    background-color: #f9fafb;
    cursor: not-allowed;
  }

  /* Ensure the editor takes full height */
  .code-editor-container :global(.cm-editor) {
    height: 100%;
  }

  .code-editor-container :global(.cm-scroller) {
    overflow: auto;
  }

</style>