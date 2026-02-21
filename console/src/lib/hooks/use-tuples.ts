"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as tuplesApi from "../api/tuples";
import type { TupleFilter, TupleRequest, BatchTuplesRequest } from "../api/types";

export const tupleKeys = {
  all: ["tuples"] as const,
  lists: () => [...tupleKeys.all, "list"] as const,
  list: (tenantId: string, filter: TupleFilter) =>
    [...tupleKeys.lists(), tenantId, filter] as const,
};

export function useTuples(tenantId: string | null, filter?: TupleFilter) {
  return useQuery({
    queryKey: tupleKeys.list(tenantId ?? "", filter ?? {}),
    queryFn: () => tuplesApi.readTuples(tenantId!, filter),
    enabled: !!tenantId,
  });
}

export function useWriteTuple(tenantId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tuple: TupleRequest) =>
      tuplesApi.writeTuple(tenantId, tuple),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tupleKeys.lists() });
    },
  });
}

export function useWriteTuplesBatch(tenantId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (batch: BatchTuplesRequest) =>
      tuplesApi.writeTuplesBatch(tenantId, batch),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tupleKeys.lists() });
    },
  });
}

export function useDeleteTuple(tenantId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tuple: TupleRequest) =>
      tuplesApi.deleteTuple(tenantId, tuple),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tupleKeys.lists() });
    },
  });
}
