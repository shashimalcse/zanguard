"use client";

import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface ParsedType {
  name: string;
  attributes?: Record<string, string>;
  relations?: Record<string, { types: string[] }>;
  permissions?: Record<string, Record<string, unknown>>;
}

interface SchemaViewerProps {
  source: string;
}

function parseYamlSimple(source: string): ParsedType[] {
  // Simple YAML-like parser for display purposes
  // We parse the structure enough to render the visual view
  try {
    const types: ParsedType[] = [];
    const lines = source.split("\n");
    let currentType: ParsedType | null = null;
    let currentSection: "attributes" | "relations" | "permissions" | null =
      null;
    let currentRelation: string | null = null;
    let currentPermission: string | null = null;
    let permissionBody: string[] = [];

    for (const line of lines) {
      const trimmed = line.trimEnd();
      const indent = line.length - line.trimStart().length;

      if (indent === 2 && !trimmed.startsWith("-") && trimmed.endsWith(":")) {
        // Type name
        if (currentType) types.push(currentType);
        currentType = { name: trimmed.slice(2, -1) };
        currentSection = null;
      } else if (indent === 4 && trimmed.trim().endsWith(":")) {
        const section = trimmed.trim().slice(0, -1);
        if (
          section === "attributes" ||
          section === "relations" ||
          section === "permissions"
        ) {
          currentSection = section;
          currentRelation = null;
          currentPermission = null;
        }
      } else if (indent === 6 && currentType && currentSection) {
        const kv = trimmed.trim();
        if (currentSection === "attributes" && kv.includes(":")) {
          const [key, val] = kv.split(":").map((s) => s.trim());
          if (!currentType.attributes) currentType.attributes = {};
          currentType.attributes[key] = val;
        } else if (currentSection === "relations" && kv.endsWith(":")) {
          currentRelation = kv.slice(0, -1);
          if (!currentType.relations) currentType.relations = {};
          currentType.relations[currentRelation] = { types: [] };
        } else if (currentSection === "permissions" && kv.endsWith(":")) {
          if (currentPermission && permissionBody.length > 0) {
            if (!currentType.permissions) currentType.permissions = {};
            currentType.permissions[currentPermission] = {
              raw: permissionBody.join("\n"),
            };
          }
          currentPermission = kv.slice(0, -1);
          permissionBody = [];
        }
      } else if (indent === 8 && currentType) {
        if (
          currentSection === "relations" &&
          currentRelation &&
          trimmed.trim().startsWith("types:")
        ) {
          const typesStr = trimmed.trim().slice(6).trim();
          const types = typesStr
            .replace(/[\[\]]/g, "")
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean);
          if (currentType.relations?.[currentRelation]) {
            currentType.relations[currentRelation].types = types;
          }
        } else if (currentSection === "permissions" && currentPermission) {
          permissionBody.push(trimmed.trim());
        }
      } else if (indent === 10 && currentSection === "permissions") {
        permissionBody.push(trimmed.trim());
      }
    }
    if (currentType) {
      if (currentPermission && permissionBody.length > 0) {
        if (!currentType.permissions) currentType.permissions = {};
        currentType.permissions[currentPermission] = {
          raw: permissionBody.join("\n"),
        };
      }
      types.push(currentType);
    }
    return types;
  } catch {
    return [];
  }
}

export function SchemaViewer({ source }: SchemaViewerProps) {
  const types = useMemo(() => parseYamlSimple(source), [source]);

  if (types.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Unable to parse schema for visual display.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {types.map((type) => (
        <Card key={type.name}>
          <CardHeader className="pb-3">
            <CardTitle className="text-base font-mono">{type.name}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {type.attributes && Object.keys(type.attributes).length > 0 && (
              <div>
                <h4 className="text-sm font-medium mb-2">Attributes</h4>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[200px]">Name</TableHead>
                      <TableHead>Type</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(type.attributes).map(([name, attrType]) => (
                      <TableRow key={name}>
                        <TableCell className="font-mono text-sm">
                          {name}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{attrType}</Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {type.relations && Object.keys(type.relations).length > 0 && (
              <div>
                <h4 className="text-sm font-medium mb-2">Relations</h4>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[200px]">Name</TableHead>
                      <TableHead>Allowed Types</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(type.relations).map(([name, rel]) => (
                      <TableRow key={name}>
                        <TableCell className="font-mono text-sm">
                          {name}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1 flex-wrap">
                            {rel.types.map((t) => (
                              <Badge key={t} variant="secondary">
                                {t}
                              </Badge>
                            ))}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {type.permissions && Object.keys(type.permissions).length > 0 && (
              <div>
                <h4 className="text-sm font-medium mb-2">Permissions</h4>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[200px]">Name</TableHead>
                      <TableHead>Definition</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(type.permissions).map(([name, def]) => (
                      <TableRow key={name}>
                        <TableCell className="font-mono text-sm">
                          {name}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {(def as { raw?: string }).raw || JSON.stringify(def)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
