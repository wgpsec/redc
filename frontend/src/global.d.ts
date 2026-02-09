/// <reference types="svelte" />

// Svelte 5 Runes type definitions
declare module 'svelte' {
  export function mount<Props extends Record<string, any>>(
    component: any,
    options: {
      target: Element | Document | ShadowRoot;
      anchor?: Node;
      props?: Props;
      context?: Map<any, any>;
      intro?: boolean;
    }
  ): {
    $set: (props: Partial<Props>) => void;
    $on: (event: string, callback: (event: CustomEvent) => void) => () => void;
    $destroy: () => void;
  };

  export function unmount(component: ReturnType<typeof mount>): void;
}

// Global Svelte 5 Runes - these are compiler-transformed
declare global {
  // $state rune
  function $state<T>(initial: T): T;
  function $state<T>(): T | undefined;

  // $derived rune
  function $derived<T>(expression: T): T;
  
  // $effect rune
  function $effect(fn: () => void | (() => void)): void;
  
  // $props rune
  function $props<T extends Record<string, any>>(): T;
  
  // $bindable rune
  function $bindable<T>(initial?: T): T;
  
  // $inspect rune
  function $inspect(...values: any[]): void;
}

export {};
