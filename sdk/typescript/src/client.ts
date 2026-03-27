import { WAIDError } from "./errors.js";
import type {
  Contact,
  CreateContactInput,
  UpdateContactInput,
  ListOptions,
  PaginatedResponse,
  IdentityResult,
  InboundEvent,
  ImportReport,
  WebhookTarget,
  CreateWebhookInput,
  HealthStatus,
} from "./types.js";

export interface WAIDClientOptions {
  baseURL: string;
  apiKey?: string;
}

export class WAIDClient {
  private readonly baseURL: string;
  private readonly apiKey?: string;

  constructor({ baseURL, apiKey }: WAIDClientOptions) {
    this.baseURL = baseURL.replace(/\/$/, "");
    this.apiKey = apiKey;
  }

  private async request<T>(
    method: string,
    path: string,
    options: { body?: unknown; query?: Record<string, string | number | undefined>; formData?: FormData } = {}
  ): Promise<T> {
    const { body, query, formData } = options;

    let url = `${this.baseURL}${path}`;
    if (query) {
      const params = new URLSearchParams();
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) {
          params.set(key, String(value));
        }
      }
      const qs = params.toString();
      if (qs) url += `?${qs}`;
    }

    const headers: Record<string, string> = {};
    if (this.apiKey) {
      headers["X-API-Key"] = this.apiKey;
    }

    let fetchBody: BodyInit | undefined;
    if (formData) {
      fetchBody = formData;
    } else if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      fetchBody = JSON.stringify(body);
    }

    const response = await fetch(url, {
      method,
      headers,
      body: fetchBody,
    });

    if (response.status === 204) {
      return undefined as T;
    }

    const contentType = response.headers.get("content-type") ?? "";
    let responseBody: unknown;

    if (contentType.includes("application/json")) {
      responseBody = await response.json();
    } else {
      responseBody = await response.text();
    }

    if (!response.ok) {
      const message =
        typeof responseBody === "object" &&
        responseBody !== null &&
        "error" in responseBody
          ? String((responseBody as { error: unknown }).error)
          : `HTTP ${response.status}`;
      throw new WAIDError(message, response.status, responseBody);
    }

    return responseBody as T;
  }

  async health(): Promise<HealthStatus> {
    return this.request<HealthStatus>("GET", "/health");
  }

  async resolve(phoneOrId: string): Promise<IdentityResult> {
    return this.request<IdentityResult>(
      "GET",
      `/resolve/${encodeURIComponent(phoneOrId)}`
    );
  }

  async createContact(input: CreateContactInput): Promise<Contact> {
    return this.request<Contact>("POST", "/contacts", { body: input });
  }

  async listContacts(
    options: ListOptions = {}
  ): Promise<PaginatedResponse<Contact>> {
    return this.request<PaginatedResponse<Contact>>("GET", "/contacts", {
      query: {
        page: options.page,
        per_page: options.per_page,
        q: options.q,
      },
    });
  }

  async getContact(id: string): Promise<Contact> {
    return this.request<Contact>("GET", `/contacts/${encodeURIComponent(id)}`);
  }

  async updateContact(id: string, input: UpdateContactInput): Promise<Contact> {
    return this.request<Contact>(
      "PUT",
      `/contacts/${encodeURIComponent(id)}`,
      { body: input }
    );
  }

  async deleteContact(id: string): Promise<void> {
    return this.request<void>(
      "DELETE",
      `/contacts/${encodeURIComponent(id)}`
    );
  }

  async importContacts(file: Blob, filename?: string): Promise<ImportReport> {
    const formData = new FormData();
    formData.append("file", file, filename ?? "contacts");
    return this.request<ImportReport>("POST", "/import", { formData });
  }

  async createWebhook(input: CreateWebhookInput): Promise<WebhookTarget> {
    return this.request<WebhookTarget>("POST", "/webhooks", { body: input });
  }

  async listWebhooks(): Promise<WebhookTarget[]> {
    return this.request<WebhookTarget[]>("GET", "/webhooks");
  }

  async deleteWebhook(id: string): Promise<void> {
    return this.request<void>(
      "DELETE",
      `/webhooks/${encodeURIComponent(id)}`
    );
  }

  async inbound(
    source: "waha" | "evolution" | "meta" | "generic",
    payload: Record<string, unknown>
  ): Promise<InboundEvent> {
    return this.request<InboundEvent>(
      "POST",
      `/inbound/${encodeURIComponent(source)}`,
      { body: payload }
    );
  }
}
