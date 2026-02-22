"use client";

import { useState } from "react";
import { Tags, Building2, Search, Loader2, ChevronRight, Plus, X } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useTenantStore } from "@/lib/stores/tenant-store";
import {
  useObjectAttributes,
  useSubjectAttributes,
  useSaveObjectAttributes,
  useSaveSubjectAttributes,
  useListObjectAttributes,
  useListSubjectAttributes,
} from "@/lib/hooks/use-attributes";
import { AttributeEditor } from "@/components/attributes/attribute-editor";
import { EmptyState } from "@/components/shared/empty-state";
import { toast } from "sonner";
import type { ObjectAttributesItem, SubjectAttributesItem } from "@/lib/api/types";

interface SelectedEntity {
  type: string;
  id: string;
}

function ObjectAttributesTab({ tenantId }: { tenantId: string }) {
  const [selected, setSelected] = useState<SelectedEntity | null>(null);
  const [showNewForm, setShowNewForm] = useState(false);
  const [newType, setNewType] = useState("");
  const [newId, setNewId] = useState("");
  const [filterType, setFilterType] = useState("");

  const { data: listData, isLoading: isListLoading } = useListObjectAttributes(
    tenantId,
    filterType || undefined
  );

  const { data, isLoading } = useObjectAttributes(
    tenantId,
    selected?.type ?? "",
    selected?.id ?? ""
  );
  const saveAttrs = useSaveObjectAttributes(
    tenantId,
    selected?.type ?? "",
    selected?.id ?? ""
  );

  const handleSelect = (item: ObjectAttributesItem) => {
    setSelected({ type: item.object_type, id: item.object_id });
    setShowNewForm(false);
  };

  const handleLookupNew = () => {
    if (!newType || !newId) return;
    setSelected({ type: newType, id: newId });
    setShowNewForm(false);
    setNewType("");
    setNewId("");
  };

  const handleSave = async (attributes: Record<string, unknown>) => {
    try {
      await saveAttrs.mutateAsync(attributes);
      toast.success("Object attributes saved");
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to save attributes");
    }
  };

  const objects = listData?.objects ?? [];

  return (
    <div className="flex gap-4 h-full">
      {/* Left panel: list of objects */}
      <div className="w-72 flex-shrink-0 space-y-2">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder="Filter by type..."
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="h-8 pl-7 text-sm"
            />
            {filterType && (
              <button
                onClick={() => setFilterType("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </div>
          <Button
            size="sm"
            variant="outline"
            className="h-8 px-2"
            onClick={() => {
              setShowNewForm(true);
              setSelected(null);
            }}
            title="Add new object"
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        </div>

        <div className="border rounded-md overflow-hidden">
          {isListLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground text-sm gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading...
            </div>
          ) : objects.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground text-sm">
              No objects with attributes
            </div>
          ) : (
            <div className="divide-y max-h-[500px] overflow-y-auto">
              {objects.map((item) => {
                const isActive =
                  selected?.type === item.object_type &&
                  selected?.id === item.object_id;
                return (
                  <button
                    key={`${item.object_type}:${item.object_id}`}
                    onClick={() => handleSelect(item)}
                    className={`w-full flex items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted/50 transition-colors ${
                      isActive ? "bg-muted" : ""
                    }`}
                  >
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <Badge variant="outline" className="text-xs px-1 py-0 font-normal">
                          {item.object_type}
                        </Badge>
                      </div>
                      <div className="font-mono text-xs text-muted-foreground truncate mt-0.5">
                        {item.object_id}
                      </div>
                    </div>
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0 ml-2" />
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Right panel: editor */}
      <div className="flex-1 min-w-0">
        {showNewForm ? (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Lookup Object</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-end gap-2 mb-4">
                <div className="space-y-1">
                  <Label className="text-xs">Object Type</Label>
                  <Input
                    placeholder="document"
                    value={newType}
                    onChange={(e) => setNewType(e.target.value)}
                    className="h-8 w-36 text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleLookupNew()}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Object ID</Label>
                  <Input
                    placeholder="doc_1"
                    value={newId}
                    onChange={(e) => setNewId(e.target.value)}
                    className="h-8 w-36 text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleLookupNew()}
                  />
                </div>
                <Button
                  size="sm"
                  className="h-8"
                  onClick={handleLookupNew}
                  disabled={!newType || !newId}
                >
                  <Search className="mr-2 h-3.5 w-3.5" />
                  Open
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-8"
                  onClick={() => setShowNewForm(false)}
                >
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : selected ? (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-mono flex items-center gap-2">
                <Badge variant="outline" className="font-normal">
                  {selected.type}
                </Badge>
                <span>{selected.id}</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="flex items-center gap-2 py-8 justify-center text-muted-foreground text-sm">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading...
                </div>
              ) : (
                <AttributeEditor
                  attributes={data?.attributes ?? {}}
                  onSave={handleSave}
                  isSaving={saveAttrs.isPending}
                />
              )}
            </CardContent>
          </Card>
        ) : (
          <div className="flex items-center justify-center h-48 border rounded-md border-dashed text-muted-foreground text-sm">
            Select an object from the list or click + to look up a new one
          </div>
        )}
      </div>
    </div>
  );
}

function SubjectAttributesTab({ tenantId }: { tenantId: string }) {
  const [selected, setSelected] = useState<SelectedEntity | null>(null);
  const [showNewForm, setShowNewForm] = useState(false);
  const [newType, setNewType] = useState("");
  const [newId, setNewId] = useState("");
  const [filterType, setFilterType] = useState("");

  const { data: listData, isLoading: isListLoading } = useListSubjectAttributes(
    tenantId,
    filterType || undefined
  );

  const { data, isLoading } = useSubjectAttributes(
    tenantId,
    selected?.type ?? "",
    selected?.id ?? ""
  );
  const saveAttrs = useSaveSubjectAttributes(
    tenantId,
    selected?.type ?? "",
    selected?.id ?? ""
  );

  const handleSelect = (item: SubjectAttributesItem) => {
    setSelected({ type: item.subject_type, id: item.subject_id });
    setShowNewForm(false);
  };

  const handleLookupNew = () => {
    if (!newType || !newId) return;
    setSelected({ type: newType, id: newId });
    setShowNewForm(false);
    setNewType("");
    setNewId("");
  };

  const handleSave = async (attributes: Record<string, unknown>) => {
    try {
      await saveAttrs.mutateAsync(attributes);
      toast.success("Subject attributes saved");
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to save attributes");
    }
  };

  const subjects = listData?.subjects ?? [];

  return (
    <div className="flex gap-4 h-full">
      {/* Left panel: list of subjects */}
      <div className="w-72 flex-shrink-0 space-y-2">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder="Filter by type..."
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="h-8 pl-7 text-sm"
            />
            {filterType && (
              <button
                onClick={() => setFilterType("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </div>
          <Button
            size="sm"
            variant="outline"
            className="h-8 px-2"
            onClick={() => {
              setShowNewForm(true);
              setSelected(null);
            }}
            title="Add new subject"
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        </div>

        <div className="border rounded-md overflow-hidden">
          {isListLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground text-sm gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading...
            </div>
          ) : subjects.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground text-sm">
              No subjects with attributes
            </div>
          ) : (
            <div className="divide-y max-h-[500px] overflow-y-auto">
              {subjects.map((item) => {
                const isActive =
                  selected?.type === item.subject_type &&
                  selected?.id === item.subject_id;
                return (
                  <button
                    key={`${item.subject_type}:${item.subject_id}`}
                    onClick={() => handleSelect(item)}
                    className={`w-full flex items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted/50 transition-colors ${
                      isActive ? "bg-muted" : ""
                    }`}
                  >
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <Badge variant="outline" className="text-xs px-1 py-0 font-normal">
                          {item.subject_type}
                        </Badge>
                      </div>
                      <div className="font-mono text-xs text-muted-foreground truncate mt-0.5">
                        {item.subject_id}
                      </div>
                    </div>
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0 ml-2" />
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Right panel: editor */}
      <div className="flex-1 min-w-0">
        {showNewForm ? (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">Lookup Subject</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-end gap-2 mb-4">
                <div className="space-y-1">
                  <Label className="text-xs">Subject Type</Label>
                  <Input
                    placeholder="user"
                    value={newType}
                    onChange={(e) => setNewType(e.target.value)}
                    className="h-8 w-36 text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleLookupNew()}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Subject ID</Label>
                  <Input
                    placeholder="alice"
                    value={newId}
                    onChange={(e) => setNewId(e.target.value)}
                    className="h-8 w-36 text-sm"
                    onKeyDown={(e) => e.key === "Enter" && handleLookupNew()}
                  />
                </div>
                <Button
                  size="sm"
                  className="h-8"
                  onClick={handleLookupNew}
                  disabled={!newType || !newId}
                >
                  <Search className="mr-2 h-3.5 w-3.5" />
                  Open
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-8"
                  onClick={() => setShowNewForm(false)}
                >
                  Cancel
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : selected ? (
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-mono flex items-center gap-2">
                <Badge variant="outline" className="font-normal">
                  {selected.type}
                </Badge>
                <span>{selected.id}</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="flex items-center gap-2 py-8 justify-center text-muted-foreground text-sm">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading...
                </div>
              ) : (
                <AttributeEditor
                  attributes={data?.attributes ?? {}}
                  onSave={handleSave}
                  isSaving={saveAttrs.isPending}
                />
              )}
            </CardContent>
          </Card>
        ) : (
          <div className="flex items-center justify-center h-48 border rounded-md border-dashed text-muted-foreground text-sm">
            Select a subject from the list or click + to look up a new one
          </div>
        )}
      </div>
    </div>
  );
}

export default function AttributesPage() {
  const { selectedTenantId } = useTenantStore();

  if (!selectedTenantId) {
    return (
      <EmptyState
        icon={Building2}
        title="No tenant selected"
        description="Select a tenant from the dropdown above to manage attributes."
      />
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Attributes</h1>

      <Tabs defaultValue="objects">
        <TabsList>
          <TabsTrigger value="objects">Object Attributes</TabsTrigger>
          <TabsTrigger value="subjects">Subject Attributes</TabsTrigger>
        </TabsList>

        <TabsContent value="objects" className="mt-4">
          <ObjectAttributesTab tenantId={selectedTenantId} />
        </TabsContent>

        <TabsContent value="subjects" className="mt-4">
          <SubjectAttributesTab tenantId={selectedTenantId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
