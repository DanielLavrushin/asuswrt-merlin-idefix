/* eslint-disable no-unused-vars */
import { EngineLoadingProgress } from './modules/Engine';

export {};

interface AddonUiGlobal {
  custom_settings: Record<string, string>;
}

declare global {
  interface Window {
    idefix: AddonUiGlobal;
    confirm: (message?: string) => boolean;
    hint: (message: string) => void;
    overlib: (message: string) => void;
    show_menu: () => void;
    showLoading: (delay?: number | null, flag?: string | EngineLoadingProgress) => void;
    updateLoadingProgress: (progress?: EngineLoadingProgress) => void;
    hideLoading: () => void;
    LoadingTime: (seconds: number, flag?: string) => void;
    showtext: (element: HTMLElement | null, text: string) => void;
    y: number;
    progress: number;
  }
}
