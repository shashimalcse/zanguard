"use client";

import { useState } from "react";
import { Check, ChevronsUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Badge } from "@/components/ui/badge";
import { useTenants } from "@/lib/hooks/use-tenants";
import { useTenantStore } from "@/lib/stores/tenant-store";

export function TenantSelector() {
  const [open, setOpen] = useState(false);
  const { selectedTenantId, setSelectedTenantId } = useTenantStore();
  const { data, isLoading } = useTenants();

  const tenants = data?.tenants ?? [];
  const selected = tenants.find((t) => t.id === selectedTenantId);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-[260px] justify-between"
        >
          {selected ? (
            <span className="flex items-center gap-2 truncate">
              <span className="truncate">{selected.display_name || selected.id}</span>
              <Badge
                variant={selected.status === "active" ? "default" : "secondary"}
                className="text-[10px] px-1.5 py-0"
              >
                {selected.status}
              </Badge>
            </span>
          ) : (
            <span className="text-muted-foreground">
              {isLoading ? "Loading..." : "Select tenant..."}
            </span>
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[260px] p-0">
        <Command>
          <CommandInput placeholder="Search tenants..." />
          <CommandList>
            <CommandEmpty>No tenants found.</CommandEmpty>
            <CommandGroup>
              {tenants.map((tenant) => (
                <CommandItem
                  key={tenant.id}
                  value={tenant.id}
                  onSelect={(value) => {
                    setSelectedTenantId(
                      value === selectedTenantId ? null : value
                    );
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      selectedTenantId === tenant.id
                        ? "opacity-100"
                        : "opacity-0"
                    )}
                  />
                  <span className="truncate">
                    {tenant.display_name || tenant.id}
                  </span>
                  <Badge
                    variant={
                      tenant.status === "active" ? "default" : "secondary"
                    }
                    className="ml-auto text-[10px] px-1.5 py-0"
                  >
                    {tenant.status}
                  </Badge>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
