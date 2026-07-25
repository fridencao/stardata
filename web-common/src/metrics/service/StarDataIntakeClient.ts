import type { MetricsEvent } from "./MetricsTypes";

const StarDataIntakeUser = import.meta.env.STARDATA_UI_PUBLIC_INTAKE_USER;
const StarDataIntakePassword = import.meta.env.STARDATA_UI_PUBLIC_INTAKE_PASSWORD;

export interface TelemetryClient {
  fireEvent(event: MetricsEvent): Promise<void>;
}

export class StarDataIntakeClient implements TelemetryClient {
  private readonly authHeader: string;
  private readonly host: string;

  public constructor(host: string) {
    this.host = host;
    // this is the format rill-intake expects.
    this.authHeader =
      "Basic " + btoa(`${StarDataIntakeUser}:${StarDataIntakePassword}`);
  }

  public async fireEvent(event: MetricsEvent) {
    if (!StarDataIntakeUser || !StarDataIntakePassword) return;

    try {
      const resp = await fetch(`${this.host}/local/track`, {
        method: "POST",
        body: JSON.stringify(event),
        headers: {
          Authorization: this.authHeader,
        },
      });
      if (!resp.ok)
        console.error(`Failed to send ${event.event_type}. ${resp.statusText}`);
    } catch (err) {
      console.error(`Failed to send ${event.event_type}. ${err.message}`);
    }
  }
}
