"use client";

import { useQuery } from "@tanstack/react-query";
import * as tenantsApi from "../api/tenants";

export const tenantKeys = {
  all: ["tenants"] as const,
  lists: () => [...tenantKeys.all, "list"] as const,
  list: (filters: Record<string, unknown>) =>
    [...tenantKeys.lists(), filters] as const,
  details: () => [...tenantKeys.all, "detail"] as const,
  detail: (id: string) => [...tenantKeys.details(), id] as const,
};

export function useTenants(filters?: { status?: string }) {
  return useQuery({
    queryKey: tenantKeys.list(filters ?? {}),
    queryFn: () => tenantsApi.listTenants(filters),
  });
}

export function useTenant(tenantId: string | null) {
  return useQuery({
    queryKey: tenantKeys.detail(tenantId ?? ""),
    queryFn: () => tenantsApi.getTenant(tenantId!),
    enabled: !!tenantId,
  });
}
