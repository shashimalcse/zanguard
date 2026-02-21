import { apiClient } from "./client";
import type { ExpandRequest, SubjectTree } from "./types";

export async function expandRelation(
  tenantId: string,
  req: ExpandRequest
): Promise<SubjectTree> {
  return apiClient
    .post(`api/v1/t/${tenantId}/expand`, { json: req })
    .json();
}
