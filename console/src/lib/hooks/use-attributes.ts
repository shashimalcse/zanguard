"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as attributesApi from "../api/attributes";

export const attributeKeys = {
  all: ["attributes"] as const,
  object: (tenantId: string, type: string, id: string) =>
    [...attributeKeys.all, "object", tenantId, type, id] as const,
  subject: (tenantId: string, type: string, id: string) =>
    [...attributeKeys.all, "subject", tenantId, type, id] as const,
};

export function useObjectAttributes(
  tenantId: string | null,
  objectType: string,
  objectId: string
) {
  return useQuery({
    queryKey: attributeKeys.object(tenantId ?? "", objectType, objectId),
    queryFn: () =>
      attributesApi.getObjectAttributes(tenantId!, objectType, objectId),
    enabled: !!tenantId && !!objectType && !!objectId,
    retry: false,
  });
}

export function useSubjectAttributes(
  tenantId: string | null,
  subjectType: string,
  subjectId: string
) {
  return useQuery({
    queryKey: attributeKeys.subject(tenantId ?? "", subjectType, subjectId),
    queryFn: () =>
      attributesApi.getSubjectAttributes(tenantId!, subjectType, subjectId),
    enabled: !!tenantId && !!subjectType && !!subjectId,
    retry: false,
  });
}

export function useSaveObjectAttributes(
  tenantId: string,
  objectType: string,
  objectId: string
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (attributes: Record<string, unknown>) =>
      attributesApi.setObjectAttributes(
        tenantId,
        objectType,
        objectId,
        attributes
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: attributeKeys.object(tenantId, objectType, objectId),
      });
    },
  });
}

export function useSaveSubjectAttributes(
  tenantId: string,
  subjectType: string,
  subjectId: string
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (attributes: Record<string, unknown>) =>
      attributesApi.setSubjectAttributes(
        tenantId,
        subjectType,
        subjectId,
        attributes
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: attributeKeys.subject(tenantId, subjectType, subjectId),
      });
    },
  });
}
