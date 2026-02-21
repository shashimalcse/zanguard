import { apiClient } from "./client";
import type { ChangelogResponse } from "./types";

export async function readChangelog(
  tenantId: string,
  params?: { since_seq?: number; limit?: number }
): Promise<ChangelogResponse> {
  const searchParams: Record<string, string> = {};
  if (params?.since_seq !== undefined)
    searchParams.since_seq = String(params.since_seq);
  if (params?.limit !== undefined) searchParams.limit = String(params.limit);
  return apiClient
    .get(`api/v1/t/${tenantId}/changelog`, { searchParams })
    .json();
}
