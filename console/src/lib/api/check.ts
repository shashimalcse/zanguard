import { apiClient } from "./client";

export interface CheckRequest {
  subject_type: string;
  subject_id: string;
  resource_type: string;
  resource_id: string;
  action: string;
}

export interface CheckResponse {
  decision: boolean;
}

export async function checkPermission(
  tenantId: string,
  req: CheckRequest
): Promise<CheckResponse> {
  return apiClient
    .post("access/v1/evaluation", {
      headers: { "X-Tenant-ID": tenantId },
      json: {
        subject: { type: req.subject_type, id: req.subject_id },
        resource: { type: req.resource_type, id: req.resource_id },
        action: { name: req.action },
      },
    })
    .json();
}
