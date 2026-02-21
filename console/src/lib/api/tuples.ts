import { apiClient } from "./client";
import type {
  TupleRequest,
  BatchTuplesRequest,
  TuplesResponse,
  TupleFilter,
} from "./types";

export async function readTuples(
  tenantId: string,
  filter?: TupleFilter
): Promise<TuplesResponse> {
  const searchParams: Record<string, string> = {};
  if (filter?.object_type) searchParams.object_type = filter.object_type;
  if (filter?.object_id) searchParams.object_id = filter.object_id;
  if (filter?.relation) searchParams.relation = filter.relation;
  if (filter?.subject_type) searchParams.subject_type = filter.subject_type;
  if (filter?.subject_id) searchParams.subject_id = filter.subject_id;
  if (filter?.subject_relation)
    searchParams.subject_relation = filter.subject_relation;
  return apiClient
    .get(`api/v1/t/${tenantId}/tuples`, { searchParams })
    .json();
}

export async function writeTuple(
  tenantId: string,
  tuple: TupleRequest
): Promise<void> {
  await apiClient.post(`api/v1/t/${tenantId}/tuples`, { json: tuple });
}

export async function writeTuplesBatch(
  tenantId: string,
  batch: BatchTuplesRequest
): Promise<{ status: string; count: number }> {
  return apiClient
    .post(`api/v1/t/${tenantId}/tuples/batch`, { json: batch })
    .json();
}

export async function deleteTuple(
  tenantId: string,
  tuple: TupleRequest
): Promise<void> {
  await apiClient.delete(`api/v1/t/${tenantId}/tuples`, { json: tuple });
}
