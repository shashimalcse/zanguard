"use client";

import { useQuery } from "@tanstack/react-query";
import * as changelogApi from "../api/changelog";

export const changelogKeys = {
  all: ["changelog"] as const,
  list: (tenantId: string, sinceSeq: number, limit: number) =>
    [...changelogKeys.all, tenantId, sinceSeq, limit] as const,
};

export function useChangelog(
  tenantId: string | null,
  sinceSeq?: number,
  limit?: number
) {
  return useQuery({
    queryKey: changelogKeys.list(tenantId ?? "", sinceSeq ?? 0, limit ?? 100),
    queryFn: () =>
      changelogApi.readChangelog(tenantId!, {
        since_seq: sinceSeq,
        limit: limit ?? 100,
      }),
    enabled: !!tenantId,
  });
}
