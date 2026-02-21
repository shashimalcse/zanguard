"use client";

import { Handle, Position, type NodeProps } from "@xyflow/react";
import { User, Users, Box } from "lucide-react";
import type { SubjectRef } from "@/lib/api/types";

interface TreeNodeData {
  subject: SubjectRef;
  isRoot?: boolean;
  [key: string]: unknown;
}

const typeIcons: Record<string, typeof User> = {
  user: User,
  group: Users,
};

const typeColors: Record<string, string> = {
  user: "bg-green-50 border-green-200",
  group: "bg-purple-50 border-purple-200",
};

export function TreeNode({ data }: NodeProps) {
  const nodeData = data as TreeNodeData;
  const subject = nodeData.subject;
  const Icon = typeIcons[subject.Type] ?? Box;
  const colorClass = nodeData.isRoot
    ? "bg-blue-50 border-blue-300"
    : typeColors[subject.Type] ?? "bg-gray-50 border-gray-200";

  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-gray-400" />
      <div
        className={`flex items-center gap-2 rounded-lg border-2 px-3 py-2 shadow-sm ${colorClass}`}
      >
        <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0">
          <div className="text-xs font-mono font-medium truncate">
            {subject.Type}:{subject.ID}
          </div>
          {subject.Relation && (
            <div className="text-[10px] text-muted-foreground font-mono">
              #{subject.Relation}
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-gray-400"
      />
    </>
  );
}
