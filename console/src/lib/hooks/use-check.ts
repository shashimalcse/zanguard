"use client";

import { useMutation } from "@tanstack/react-query";
import * as checkApi from "../api/check";

export function useCheck(tenantId: string) {
  return useMutation({
    mutationFn: (req: checkApi.CheckRequest) =>
      checkApi.checkPermission(tenantId, req),
  });
}
