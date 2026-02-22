"use client";

import { useMemo } from "react";
import dagre from "dagre";
import {
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  MarkerType,
  ReactFlow,
  Handle,
  Position,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

// ---- Types ----

interface ParsedRelation {
  relation: string;
  targets: string[];
}

interface ParsedType {
  name: string;
  relations: ParsedRelation[];
}

interface TypeNodeData {
  label: string;
  isExternal: boolean;
  relCount: number;
  [key: string]: unknown;
}

// ---- Custom Node ----

function TypeNode({ data }: NodeProps) {
  const d = data as TypeNodeData;
  return (
    <div
      style={{
        background: d.isExternal ? "#f8fafc" : "#ffffff",
        border: d.isExternal ? "1.5px dashed #94a3b8" : "1.5px solid #e2e8f0",
        borderRadius: 10,
        minWidth: 140,
        boxShadow: d.isExternal ? "none" : "0 1px 3px 0 rgba(0,0,0,0.08)",
      }}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{ opacity: 0, pointerEvents: "none" }}
      />
      <div style={{ padding: "8px 14px" }}>
        <div
          style={{
            fontFamily: "ui-monospace, monospace",
            fontSize: 13,
            fontWeight: 600,
            color: d.isExternal ? "#64748b" : "#0f172a",
            whiteSpace: "nowrap",
          }}
        >
          {d.label}
        </div>
        <div style={{ fontSize: 11, color: "#94a3b8", marginTop: 2 }}>
          {d.isExternal
            ? "external"
            : d.relCount > 0
            ? `${d.relCount} relation${d.relCount !== 1 ? "s" : ""}`
            : "no relations"}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Right}
        style={{ opacity: 0, pointerEvents: "none" }}
      />
    </div>
  );
}

const nodeTypes = { typeNode: TypeNode };

// ---- Parser ----

function stripQuotes(value: string): string {
  return value.replace(/^['\"]|['\"]$/g, "");
}

function parseInlineList(value: string): string[] {
  const normalized = value.trim();
  if (!normalized) return [];
  if (normalized.startsWith("[") && normalized.endsWith("]")) {
    return normalized
      .slice(1, -1)
      .split(",")
      .map((item) => stripQuotes(item.trim()))
      .filter(Boolean);
  }
  return [stripQuotes(normalized)];
}

function parseTypes(source: string): ParsedType[] {
  const lines = source.split("\n");
  const parsed = new Map<string, ParsedType>();

  let inTypesSection = false;
  let currentType: ParsedType | null = null;
  let currentSection: "relations" | null = null;
  let currentRelation: ParsedRelation | null = null;
  let collectingTypesList = false;

  for (const rawLine of lines) {
    if (!rawLine.trim() || rawLine.trimStart().startsWith("#")) continue;

    const indent = rawLine.length - rawLine.trimStart().length;
    const line = rawLine.trim();

    if (indent === 0) {
      inTypesSection = line === "types:";
      currentType = null;
      currentSection = null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (!inTypesSection) continue;

    if (indent === 2 && line.endsWith(":")) {
      const typeName = line.slice(0, -1).trim();
      if (!typeName) continue;
      currentType = { name: typeName, relations: [] };
      parsed.set(typeName, currentType);
      currentSection = null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (!currentType) continue;

    if (indent === 4 && line.endsWith(":")) {
      currentSection =
        line.slice(0, -1).trim() === "relations" ? "relations" : null;
      currentRelation = null;
      collectingTypesList = false;
      continue;
    }

    if (currentSection !== "relations") continue;

    if (indent === 6 && line.endsWith(":")) {
      currentRelation = { relation: line.slice(0, -1).trim(), targets: [] };
      currentType.relations.push(currentRelation);
      collectingTypesList = false;
      continue;
    }

    if (!currentRelation) continue;

    if (indent === 8 && line.startsWith("types:")) {
      const rawTypes = line.slice("types:".length).trim();
      if (rawTypes) {
        currentRelation.targets.push(...parseInlineList(rawTypes));
        collectingTypesList = false;
      } else {
        collectingTypesList = true;
      }
      continue;
    }

    if (collectingTypesList && indent >= 10 && line.startsWith("-")) {
      const item = line.slice(1).trim();
      if (item) currentRelation.targets.push(stripQuotes(item));
      continue;
    }

    if (indent <= 8) collectingTypesList = false;
  }

  return Array.from(parsed.values());
}

// ---- Graph builder ----

function buildGraph(source: string): { nodes: Node[]; edges: Edge[] } {
  const parsedTypes = parseTypes(source);
  const typesByName = new Map(parsedTypes.map((t) => [t.name, t]));

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const addedTypes = new Set<string>();

  // Aggregate relation names per source→target pair
  const edgeRelations = new Map<string, string[]>();

  for (const typeDef of parsedTypes) {
    if (!addedTypes.has(typeDef.name)) {
      addedTypes.add(typeDef.name);
      nodes.push({
        id: `type:${typeDef.name}`,
        type: "typeNode",
        width: 160,
        height: 56,
        data: {
          label: typeDef.name,
          isExternal: false,
          relCount: typeDef.relations.length,
        } as TypeNodeData,
        position: { x: 0, y: 0 },
      });
    }

    for (const relation of typeDef.relations) {
      for (const target of relation.targets) {
        const [targetType] = target.split("#");
        if (!targetType) continue;

        if (!addedTypes.has(targetType)) {
          addedTypes.add(targetType);
          const targetDef = typesByName.get(targetType);
          nodes.push({
            id: `type:${targetType}`,
            type: "typeNode",
            width: 160,
            height: 56,
            data: {
              label: targetType,
              isExternal: !targetDef,
              relCount: targetDef?.relations.length ?? 0,
            } as TypeNodeData,
            position: { x: 0, y: 0 },
          });
        }

        const edgeKey = `${typeDef.name}\u2192${targetType}`;
        if (!edgeRelations.has(edgeKey)) edgeRelations.set(edgeKey, []);
        edgeRelations.get(edgeKey)!.push(relation.relation);
      }
    }
  }

  for (const [key, relNames] of edgeRelations.entries()) {
    const arrowIdx = key.indexOf("\u2192");
    const sourceType = key.slice(0, arrowIdx);
    const targetType = key.slice(arrowIdx + 1);
    edges.push({
      id: `edge:${key}`,
      source: `type:${sourceType}`,
      target: `type:${targetType}`,
      label: relNames.join(", "),
      type: "smoothstep",
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 14,
        height: 14,
        color: "#94a3b8",
      },
      style: { stroke: "#94a3b8", strokeWidth: 1.5 },
      labelStyle: {
        fill: "#475569",
        fontSize: 11,
        fontFamily: "ui-monospace, monospace",
      },
      labelBgStyle: { fill: "#ffffff", fillOpacity: 0.95 },
      labelBgPadding: [4, 8] as [number, number],
      labelBgBorderRadius: 4,
    });
  }

  // Dagre layout
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "LR", ranksep: 130, nodesep: 70 });
  g.setDefaultEdgeLabel(() => ({}));

  for (const node of nodes) {
    g.setNode(node.id, { width: 160, height: 56 });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return {
    nodes: nodes.map((node) => {
      const pos = g.node(node.id);
      return { ...node, position: { x: pos.x - 80, y: pos.y - 28 } };
    }),
    edges,
  };
}

// ---- Component ----

interface SchemaGraphProps {
  source: string;
}

export function SchemaGraph({ source }: SchemaGraphProps) {
  const { nodes, edges } = useMemo(() => buildGraph(source), [source]);

  if (nodes.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Unable to parse schema relations for graph visualization.
      </p>
    );
  }

  return (
    <div className="h-[580px] rounded-lg border overflow-hidden bg-slate-50/40">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.3 }}
        nodesDraggable={true}
        nodesConnectable={false}
        elementsSelectable={true}
        proOptions={{ hideAttribution: true }}
        defaultEdgeOptions={{ type: "smoothstep" }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="#cbd5e1"
        />
        <Controls showInteractive={false} />
        <MiniMap
          nodeColor={(node) =>
            (node.data as TypeNodeData).isExternal ? "#cbd5e1" : "#93c5fd"
          }
          nodeStrokeColor={(node) =>
            (node.data as TypeNodeData).isExternal ? "#94a3b8" : "#3b82f6"
          }
          nodeStrokeWidth={2}
          maskColor="rgba(241,245,249,0.85)"
          style={{ border: "1px solid #e2e8f0", borderRadius: 8 }}
          pannable
          zoomable
        />
      </ReactFlow>
    </div>
  );
}
