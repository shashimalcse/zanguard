"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as schemasApi from "../api/schemas";

export const schemaKeys = {
  all: ["schema"] as const,
  detail: (tenantId: string) => [...schemaKeys.all, tenantId] as const,
};

export function useSchema(tenantId: string | null) {
  return useQuery({
    queryKey: schemaKeys.detail(tenantId ?? ""),
    queryFn: () => schemasApi.getSchema(tenantId!),
    enabled: !!tenantId,
    retry: false,
  });
}

export function useSaveSchema(tenantId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (yaml: string) => schemasApi.loadSchema(tenantId, yaml),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: schemaKeys.detail(tenantId),
      });
    },
  });
}
