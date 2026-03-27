export interface Contact {
  id: string;
  phone: string;
  bsuid: string | null;
  external_id: string | null;
  name: string;
  metadata: Record<string, unknown>;
  status: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface CreateContactInput {
  phone: string;
  name: string;
  external_id?: string | null;
  metadata?: Record<string, unknown>;
}

export interface UpdateContactInput {
  name?: string;
  external_id?: string | null;
  status?: string;
  metadata?: Record<string, unknown>;
}

export interface ListOptions {
  page?: number;
  per_page?: number;
  q?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
}

export interface IdentityResult {
  contact: Contact | null;
  match_type: "phone" | "bsuid" | "created" | "not_found";
  confidence: number;
  resolved_at: string;
}

export interface InboundEvent {
  source_id: string;
  phone: string;
  bsuid: string | null;
  display_name: string;
  source: "waha" | "evolution" | "meta" | "generic";
  raw_payload: Record<string, unknown>;
  timestamp: string;
}

export interface ImportError {
  row: number;
  phone: string;
  reason: string;
}

export interface ImportReport {
  total: number;
  created: number;
  updated: number;
  errors: number;
  details?: ImportError[];
}

export interface WebhookTarget {
  id: string;
  url: string;
  events: Array<
    | "contact.resolved"
    | "contact.created"
    | "contact.updated"
    | "contact.not_found"
  >;
  secret?: string;
  active: boolean;
  created_at: string;
}

export interface CreateWebhookInput {
  url: string;
  events?: Array<
    | "contact.resolved"
    | "contact.created"
    | "contact.updated"
    | "contact.not_found"
  >;
  secret?: string;
}

export interface HealthStatus {
  status: string;
  database: "ok" | "error";
  version: string;
}
