"use client";

import { TenantSelector } from "./tenant-selector";

export function Header() {
  return (
    <header className="flex h-14 items-center justify-between border-b px-6">
      <div className="text-sm font-medium text-muted-foreground">
        Admin Console
      </div>
      <TenantSelector />
    </header>
  );
}
