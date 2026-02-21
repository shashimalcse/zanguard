import { apiClient } from "./client";
import type { Tenant, ListTenantsResponse } from "./types";

export async function listTenants(params?: {
  status?: string;
  parent_id?: string;
  limit?: number;
  offset?: number;
}): Promise<ListTenantsResponse> {
  const searchParams: Record<string, string> = {};
  if (params?.status) searchParams.status = params.status;
  if (params?.parent_id) searchParams.parent_id = params.parent_id;
  if (params?.limit) searchParams.limit = String(params.limit);
  if (params?.offset) searchParams.offset = String(params.offset);
  return apiClient.get("api/v1/tenants", { searchParams }).json();
}

export async function getTenant(tenantId: string): Promise<Tenant> {
  return apiClient.get(`api/v1/tenants/${tenantId}`).json();
}
