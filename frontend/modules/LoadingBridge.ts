// LoadingBridge.ts   – optional convenience layer
import { useLoading } from './../LoadingProvider';

export function useLoadingBridge() {
  const loading = useLoading();

  return {
    start: (msg = 'Please wait...', progress = 0) => loading.show(msg, progress),
    update: (msg?: string, progress?: number) => loading.update(msg, progress),
    stop: () => loading.hide()
  };
}
