"use client";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { TupleDisplay } from "@/components/tuples/tuple-display";
import type { ChangelogEntry } from "@/lib/api/types";

const opVariants: Record<string, "default" | "destructive" | "secondary"> = {
  INSERT: "default",
  DELETE: "destructive",
  UPDATE: "secondary",
};

interface ChangelogTableProps {
  entries: ChangelogEntry[];
}

export function ChangelogTable({ entries }: ChangelogTableProps) {
  if (entries.length === 0) {
    return (
      <div className="text-center py-8 text-sm text-muted-foreground">
        No changelog entries found.
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[70px]">Seq</TableHead>
          <TableHead className="w-[80px]">Op</TableHead>
          <TableHead>Tuple</TableHead>
          <TableHead className="w-[160px]">Timestamp</TableHead>
          <TableHead className="w-[80px]">Source</TableHead>
          <TableHead className="w-[100px]">Actor</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((entry) => (
          <TableRow key={entry.seq}>
            <TableCell className="font-mono text-xs">{entry.seq}</TableCell>
            <TableCell>
              <Badge variant={opVariants[entry.op] ?? "secondary"}>
                {entry.op}
              </Badge>
            </TableCell>
            <TableCell>
              {entry.tuple && (
                <TupleDisplay
                  objectType={entry.tuple.object_type}
                  objectId={entry.tuple.object_id}
                  relation={entry.tuple.relation}
                  subjectType={entry.tuple.subject_type}
                  subjectId={entry.tuple.subject_id}
                  subjectRelation={entry.tuple.subject_relation}
                />
              )}
            </TableCell>
            <TableCell className="text-xs text-muted-foreground">
              {new Date(entry.ts).toLocaleString()}
            </TableCell>
            <TableCell className="text-xs">{entry.source}</TableCell>
            <TableCell className="text-xs text-muted-foreground truncate max-w-[100px]">
              {entry.actor || "-"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
