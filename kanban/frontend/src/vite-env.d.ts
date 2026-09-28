/// <reference types="vite/client" />

interface Window {
  WebApp?: {
    ready?: () => void;
    initDataUnsafe?: {
      user?: {
        id?: number;
        user_id?: number;
        username?: string;
        first_name?: string;
        last_name?: string;
        name?: string;
      };
      chat?: {
        id: number;
      };
    };
    initData?: string;
    close?: () => void;
    expand?: () => void;
    MainButton: {
      setText: (text: string) => void;
      show: () => void;
      hide: () => void;
      onClick: (callback: () => void) => void;
    };
  };
}
