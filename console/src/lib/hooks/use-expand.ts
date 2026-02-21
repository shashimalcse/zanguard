"use client";

import { useMutation } from "@tanstack/react-query";
import * as expandApi from "../api/expand";
import type { ExpandRequest } from "../api/types";

export function useExpand(tenantId: string) {
  return useMutation({
    mutationFn: (req: ExpandRequest) =>
      expandApi.expandRelation(tenantId, req),
  });
}
