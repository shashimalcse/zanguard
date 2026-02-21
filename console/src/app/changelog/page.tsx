"use client";

import { useState } from "react";
import { ScrollText, Building2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useTenantStore } from "@/lib/stores/tenant-store";
import { useChangelog } from "@/lib/hooks/use-changelog";
import { ChangelogTable } from "@/components/changelog/changelog-table";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";

export default function ChangelogPage() {
  const { selectedTenantId } = useTenantStore();
  const [sinceSeq, setSinceSeq] = useState(0);
  const { data, isLoading } = useChangelog(selectedTenantId, sinceSeq, 50);

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to view the changelog."
      />
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">Changelog</h1>
          {data && (
            <Badge variant="secondary" className="text-xs">
              Latest seq: {data.latest_sequence}
            </Badge>
          )}
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : data ? (
        <>
          <ChangelogTable entries={data.entries ?? []} />
          {data.entries && data.entries.length >= 50 && (
            <div className="flex justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  setSinceSeq(data.entries[data.entries.length - 1].seq)
                }
              >
                Load More
              </Button>
            </div>
          )}
        </>
      ) : (
        <EmptyState
          icon={ScrollText}
          title="No changelog entries"
          description="Changelog entries will appear here after tuple operations are performed."
        />
      )}
    </div>
  );
}
