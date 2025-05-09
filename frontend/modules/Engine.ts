/* eslint-disable @typescript-eslint/no-unsafe-call */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
/* eslint-disable security/detect-object-injection */
/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unsafe-return */
/* eslint-disable @typescript-eslint/no-misused-promises */
import axios from 'axios';

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

/* eslint-disable @typescript-eslint/no-unused-vars */
export enum SubmitActions {
  restart = 'idefix_restart',
  generateToken = 'idefix_generate_token',
  update = 'idefix_update'
}

export interface EngineToken {
  sig?: string;
  ts?: number;
  cl?: string;
}
class Engine {
  public token: EngineToken | undefined;
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

  async executeWithLoadingProgress(action: () => Promise<void>, windowReload = true): Promise<void> {
    let loadingProgress = new EngineLoadingProgress(0, 'Please, wait');
    window.showLoading(null, loadingProgress);

    await action();
    await this.checkLoadingProgress(loadingProgress, windowReload);
  }

  async checkLoadingProgress(loadingProgress: EngineLoadingProgress, windowReload = true): Promise<void> {
    return new Promise((resolve, reject) => {
      const checkProgressInterval = setInterval(async () => {
        try {
          const response = await this.getResponse();
          if (response.loading) {
            loadingProgress = response.loading;
            window.updateLoadingProgress(loadingProgress);
          } else {
            clearInterval(checkProgressInterval);
            window.hideLoading();
            resolve();
            if (windowReload) {
              setTimeout(() => {
                window.location.reload();
              }, 1000);
            }
          }
        } catch (error) {
          clearInterval(checkProgressInterval);
          window.hideLoading();
          reject(new Error('Error while checking loading progress'));
        }
      }, 1000);
    });
  }
}

let engine = new Engine();
export default engine;
