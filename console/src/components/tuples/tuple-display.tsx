"use client";

interface TupleDisplayProps {
  objectType: string;
  objectId: string;
  relation: string;
  subjectType: string;
  subjectId: string;
  subjectRelation?: string;
}

export function TupleDisplay({
  objectType,
  objectId,
  relation,
  subjectType,
  subjectId,
  subjectRelation,
}: TupleDisplayProps) {
  return (
    <code className="text-xs font-mono">
      <span className="text-blue-600">{objectType}</span>
      <span className="text-muted-foreground">:</span>
      <span className="text-slate-700">{objectId}</span>
      <span className="text-muted-foreground">#</span>
      <span className="text-purple-600">{relation}</span>
      <span className="text-muted-foreground">@</span>
      <span className="text-green-600">{subjectType}</span>
      <span className="text-muted-foreground">:</span>
      <span className="text-slate-700">{subjectId}</span>
      {subjectRelation && (
        <>
          <span className="text-muted-foreground">#</span>
          <span className="text-orange-600">{subjectRelation}</span>
        </>
      )}
    </code>
  );
}
