/**
 * Pluggable submitter for chat data requests (StarData).
 *
 * By default, RequestDialog writes /requests.yaml through the runtime (web-local
 * draft repo). Cloud apps (web-admin) register a custom submitter here that goes
 * through the admin service instead, since viewers have no runtime repo permissions.
 */
export type RequestSubmitter = (question: string, note?: string) => Promise<void>;

let submitter: RequestSubmitter | null = null;

export function setRequestSubmitter(fn: RequestSubmitter | null): void {
  submitter = fn;
}

export function getRequestSubmitter(): RequestSubmitter | null {
  return submitter;
}
