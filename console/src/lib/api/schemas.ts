import { apiClient } from "./client";
import type { SchemaResponse } from "./types";

export async function getSchema(tenantId: string): Promise<SchemaResponse> {
  return apiClient.get(`api/v1/tenants/${tenantId}/schema`).json();
}

export async function loadSchema(
  tenantId: string,
  yaml: string
): Promise<SchemaResponse> {
  return apiClient
    .put(`api/v1/tenants/${tenantId}/schema`, {
      body: yaml,
      headers: { "Content-Type": "application/yaml" },
    })
    .json();
}
