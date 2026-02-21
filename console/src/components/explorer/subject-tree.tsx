"use client";

import { useMemo, useCallback } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import dagre from "dagre";
import { TreeNode } from "./tree-node";
import type { SubjectTree as SubjectTreeType } from "@/lib/api/types";
import "@xyflow/react/dist/style.css";

const nodeTypes = { subjectNode: TreeNode };

function treeToFlow(
  tree: SubjectTreeType,
  parentId?: string,
  depth = 0
): { nodes: Node[]; edges: Edge[] } {
  const nodeId = `${tree.Subject.Type}:${tree.Subject.ID}${tree.Subject.Relation ? "#" + tree.Subject.Relation : ""}-${depth}`;
  const nodes: Node[] = [
    {
      id: nodeId,
      type: "subjectNode",
      data: { subject: tree.Subject, isRoot: depth === 0 },
      position: { x: 0, y: 0 },
    },
  ];
  const edges: Edge[] = parentId
    ? [
        {
          id: `${parentId}->${nodeId}`,
          source: parentId,
          target: nodeId,
          animated: true,
          style: { stroke: "#94a3b8" },
        },
      ]
    : [];

  if (tree.Children) {
    for (const child of tree.Children) {
      const result = treeToFlow(child, nodeId, depth + 1);
      nodes.push(...result.nodes);
      edges.push(...result.edges);
    }
  }

  return { nodes, edges };
}

function layoutTree(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "TB", nodesep: 60, ranksep: 80 });
  g.setDefaultEdgeLabel(() => ({}));

  nodes.forEach((n) => g.setNode(n.id, { width: 180, height: 50 }));
  edges.forEach((e) => g.setEdge(e.source, e.target));
  dagre.layout(g);

  return nodes.map((n) => {
    const pos = g.node(n.id);
    return { ...n, position: { x: pos.x - 90, y: pos.y - 25 } };
  });
}

interface SubjectTreeProps {
  tree: SubjectTreeType;
}

export function SubjectTreeView({ tree }: SubjectTreeProps) {
  const { nodes: rawNodes, edges: rawEdges } = useMemo(
    () => treeToFlow(tree),
    [tree]
  );

  const layoutNodes = useMemo(
    () => layoutTree(rawNodes, rawEdges),
    [rawNodes, rawEdges]
  );

  const [nodes, , onNodesChange] = useNodesState(layoutNodes);
  const [edges, , onEdgesChange] = useEdgesState(rawEdges);

  return (
    <div className="h-[500px] rounded-md border bg-gray-50/50">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
