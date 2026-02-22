import { apiClient } from "./client";
import type {
  AttributesResponse,
  ListObjectAttributesResponse,
  ListSubjectAttributesResponse,
} from "./types";

export async function listObjectAttributes(
  tenantId: string,
  objectType?: string
): Promise<ListObjectAttributesResponse> {
  const searchParams: Record<string, string> = {};
  if (objectType) searchParams.type = objectType;
  return apiClient
    .get(`api/v1/t/${tenantId}/attributes/objects`, { searchParams })
    .json();
}

export async function listSubjectAttributes(
  tenantId: string,
  subjectType?: string
): Promise<ListSubjectAttributesResponse> {
  const searchParams: Record<string, string> = {};
  if (subjectType) searchParams.type = subjectType;
  return apiClient
    .get(`api/v1/t/${tenantId}/attributes/subjects`, { searchParams })
    .json();
}

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
