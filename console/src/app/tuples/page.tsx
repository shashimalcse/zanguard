"use client";

import { useState } from "react";
import { Link2, Building2 } from "lucide-react";
import { useTuples } from "@/lib/hooks/use-tuples";
import { useTenantStore } from "@/lib/stores/tenant-store";
import { TupleTable } from "@/components/tuples/tuple-table";
import { TupleFilters } from "@/components/tuples/tuple-filters";
import { CreateTupleDialog } from "@/components/tuples/create-tuple-dialog";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import type { TupleFilter } from "@/lib/api/types";

export default function TuplesPage() {
  const { selectedTenantId } = useTenantStore();
  const [filter, setFilter] = useState<TupleFilter>({});
  const { data, isLoading } = useTuples(selectedTenantId, filter);

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to manage relation tuples."
      />
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">Tuples</h1>
          {data && (
            <Badge variant="secondary" className="text-xs">
              {data.count} total
            </Badge>
          )}
        </div>
        <CreateTupleDialog tenantId={selectedTenantId} />
      </div>

      <TupleFilters
        filter={filter}
        onChange={setFilter}
        onClear={() => setFilter({})}
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : data ? (
        <TupleTable tenantId={selectedTenantId} tuples={data.tuples ?? []} />
      ) : (
        <EmptyState
          icon={Link2}
          title="No tuples"
          description="Create a relation tuple to define authorization relationships."
        />
      )}
    </div>
  );
}
