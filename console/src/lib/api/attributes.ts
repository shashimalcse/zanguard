import { apiClient } from "./client";
import type { AttributesResponse } from "./types";

export async function getObjectAttributes(
  tenantId: string,
  objectType: string,
  objectId: string
): Promise<AttributesResponse> {
  return apiClient
    .get(`api/v1/t/${tenantId}/attributes/objects/${objectType}/${objectId}`)
    .json();
}

export async function setObjectAttributes(
  tenantId: string,
  objectType: string,
  objectId: string,
  attributes: Record<string, unknown>
): Promise<AttributesResponse> {
  return apiClient
    .put(`api/v1/t/${tenantId}/attributes/objects/${objectType}/${objectId}`, {
      json: { attributes },
    })
    .json();
}

export async function getSubjectAttributes(
  tenantId: string,
  subjectType: string,
  subjectId: string
): Promise<AttributesResponse> {
  return apiClient
    .get(
      `api/v1/t/${tenantId}/attributes/subjects/${subjectType}/${subjectId}`
    )
    .json();
}

export async function setSubjectAttributes(
  tenantId: string,
  subjectType: string,
  subjectId: string,
  attributes: Record<string, unknown>
): Promise<AttributesResponse> {
  return apiClient
    .put(
      `api/v1/t/${tenantId}/attributes/subjects/${subjectType}/${subjectId}`,
      { json: { attributes } }
    )
    .json();
}
