"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Copy, Check } from "lucide-react";
import { useState } from "react";

interface SchemaMetaProps {
  hash: string;
  version: string;
  compiledAt: string;
}

export function SchemaMeta({ hash, version, compiledAt }: SchemaMetaProps) {
  const [copied, setCopied] = useState(false);
  const shortHash = hash.slice(0, 12);

  const copyHash = () => {
    navigator.clipboard.writeText(hash);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-center gap-4 text-sm text-muted-foreground">
      <div className="flex items-center gap-1.5">
        <span>Version:</span>
        <Badge variant="outline">{version}</Badge>
      </div>
      <div className="flex items-center gap-1.5">
        <span>Hash:</span>
        <code className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
          {shortHash}
        </code>
        <Button
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0"
          onClick={copyHash}
        >
          {copied ? (
            <Check className="h-3 w-3" />
          ) : (
            <Copy className="h-3 w-3" />
          )}
        </Button>
      </div>
      <div className="flex items-center gap-1.5">
        <span>Compiled:</span>
        <span>{new Date(compiledAt).toLocaleString()}</span>
      </div>
    </div>
  );
}
