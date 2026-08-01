import axios from 'axios';
import { useLoadingBridge } from './LoadingBridge';

export class EngineLoadingProgress {
  public progress = 0;
  public message = '';

  constructor(progress?: number, message?: string) {
    if (progress) {
      this.progress = progress;
    }
    if (message) {
      this.message = message;
    }
  }
}

class EngineResponseConfig {
  public loading?: EngineLoadingProgress;
}

export enum SubmitActions {
  restart = 'idefix_restart',
  generateToken = 'idefix_generate_token',
  update = 'idefix_update'
}

export interface EngineToken {
  sig?: string;
  ts?: number;
  cl?: string;
  n?: string;
}

export interface LoadingBridge {
  start: (msg?: string, progress?: number) => void;
  update: (msg?: string, progress?: number) => void;
  stop: () => void;
}

/**
 * Hex-encoded random identifier. Everything that names a terminal session or
 * its owner comes from here, so the choice of RNG lives in one place —
 * `crypto.getRandomValues` works over plain http too, unlike `crypto.subtle`,
 * and the router UI usually isn't on https.
 */
export const randomId = (bytes = 16): string => {
  const buf = new Uint8Array(bytes);
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(buf);
  } else {
    for (let i = 0; i < buf.length; i++) buf[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(buf, (b) => b.toString(16).padStart(2, '0')).join('');
};

class Engine {
  public token: EngineToken | undefined;
  private tokenRefresh: Promise<void> = Promise.resolve();
  private clientId: string | undefined;

  public getClientId(): string {
    this.clientId ??= randomId();
    return this.clientId;
  }

  public refreshToken(): Promise<EngineToken | undefined> {
    const minted = this.tokenRefresh.then(async () => {
      await this.submit(SubmitActions.generateToken, { client_token: this.getClientId() });
      return this.getServerToken();
    });

    this.tokenRefresh = minted.then(
      () => undefined,
      () => undefined
    );
    return minted;
  }

  private splitPayload(payload: string, chunkSize: number): string[] {
    const chunks: string[] = [];
    let index = 0;
    while (index < payload.length) {
      chunks.push(payload.slice(index, index + chunkSize));
      index += chunkSize;
    }
    return chunks;
  }
  public delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

  public setCookie(name: string, val: string) {
    const oneYear = 365 * 24 * 60 * 60 * 1000;
    const expires = new Date(Date.now() + oneYear).toUTCString();
    const safeVal = encodeURIComponent(val);

    document.cookie = `${encodeURIComponent(name)}=${safeVal}; ` + `expires=${expires}; path=/; SameSite=Lax`;
  }

  public getCookie = (name: string): string | undefined => {
    const value = '; ' + document.cookie;
    const parts = value.split('; ' + name + '=');

    if (parts.length === 2) {
      return parts.pop()?.split(';').shift();
    }
  };

  public deleteCookie = (name: string): void => {
    const date = new Date();
    date.setTime(date.getTime() + -1 * 24 * 60 * 60 * 1000);
    document.cookie = name + '=; expires=' + date.toUTCString() + '; path=/';
  };
  async getResponse(): Promise<EngineResponseConfig> {
    const response = await axios.get<EngineResponseConfig>(`/ext/idefix/response.json?_=${Date.now()}`, {
      headers: {
        'Cache-Control': 'no-cache',
        Pragma: 'no-cache',
        Expires: '0'
      }
    });
    let responseConfig = response.data;
    return responseConfig;
  }

  async getServerToken(): Promise<EngineToken | undefined> {
    const response = await axios.get<EngineToken>(`/ext/idefix/token.json?_=${Date.now()}`);

    this.token = response.data;

    return this.token;
  }

  public submit(action: string, payload: object | string | number | null | undefined = undefined, delay = 1000): Promise<void> {
    return new Promise((resolve) => {
      const iframeName = 'hidden_frame_' + Math.random().toString(36).substring(2, 9);
      const iframe = document.createElement('iframe');
      iframe.name = iframeName;
      iframe.style.display = 'none';

      document.body.appendChild(iframe);

      const form = document.createElement('form');
      form.method = 'post';
      form.action = '/start_apply.htm';
      form.target = iframeName;

      this.create_form_element(form, 'hidden', 'action_mode', 'apply');
      this.create_form_element(form, 'hidden', 'action_script', action);
      this.create_form_element(form, 'hidden', 'modified', '0');
      this.create_form_element(form, 'hidden', 'action_wait', '');

      const amngCustomInput = document.createElement('input');
      if (payload) {
        const chunkSize = 2048;
        const payloadString = JSON.stringify(payload);
        const chunks = this.splitPayload(payloadString, chunkSize);
        chunks.forEach((chunk: string, idx) => {
          window.idefix.custom_settings[`idefix_payload${idx}`] = chunk;
        });

        const customSettings = JSON.stringify(window.idefix.custom_settings);
        if (customSettings.length > 8 * 1024) {
          alert('Configuration is too large to submit via custom settings.');
          throw new Error('Configuration is too large to submit via custom settings.');
        }

        amngCustomInput.type = 'hidden';
        amngCustomInput.name = 'amng_custom';
        amngCustomInput.value = customSettings;
        form.appendChild(amngCustomInput);
      }

      document.body.appendChild(form);

      iframe.onload = () => {
        document.body.removeChild(form);
        document.body.removeChild(iframe);

        setTimeout(() => {
          resolve();
        }, delay);
      };
      form.submit();
      if (form.contains(amngCustomInput)) {
        form.removeChild(amngCustomInput);
      }
    });
  }

  create_form_element = (form: HTMLFormElement, type: string, name: string, value: string): HTMLInputElement => {
    const input = document.createElement('input');
    input.type = type;
    input.name = name;
    input.value = value;
    form.appendChild(input);
    return input;
  };

  async executeWithLoadingProgress(action: () => Promise<void>, bridge: LoadingBridge, windowReload = true): Promise<void> {
    bridge.start('Please, wait...', 0);

    await action();

    await this.checkLoadingProgress({
      onUpdate: bridge.update,
      onDone: bridge.stop
    });

    if (windowReload) {
      setTimeout(() => {
        window.location.href = window.location.pathname + '?updated=' + Date.now();
      }, 500);
    }
  }

  checkLoadingProgress(opts: { onUpdate: (msg?: string, progress?: number) => void; onDone: () => void }): Promise<void> {
    return new Promise((resolve) => {
      let seenProgress = false;
      const timer = setInterval(async () => {
        try {
          const r = await this.getResponse();

          if (r.loading) {
            seenProgress = true;
            opts.onUpdate(r.loading.message, r.loading.progress);

            if (typeof r.loading.progress === 'number' && r.loading.progress >= 100) {
              clearInterval(timer);
              opts.onDone();
              resolve();
            }
          } else if (seenProgress) {
            clearInterval(timer);
            opts.onDone();
            resolve();
          }
        } catch {}
      }, 500);
    });
  }
}

let engine = new Engine();
export default engine;
