---
id: google-drive
title: Google Drive–Style ACL
sidebar_position: 1
---

# Example: Google Drive–Style ACL

This example shows how to model a Google Drive-like permission system with folders, documents, group sharing, and inherited access — all using ZanGuard's included example schema.

## Schema

The schema is at `configs/examples/gdrive.zanguard.yaml`:

```yaml
version: "1.0"

types:
  user:
    attributes:
      clearance_level: int
      department: string
      region: string

  group:
    relations:
      member:
        types: [user]
      admin:
        types: [user]
    permissions:
      manage:
        resolve: admin

  folder:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view
      edit:
        resolve: owner

  document:
    attributes:
      classification: string
      department: string
    relations:
      owner:
        types: [user]
      editor:
        types: [user, group#member]
      viewer:
        types: [user, group#member]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - editor
          - owner
          - parent->view
      edit:
        union:
          - editor
          - owner
      delete:
        resolve: owner
      share:
        resolve: owner
```

## Scenario

We'll model this structure:

```
folder:company-docs
  └── document:q4-report
  └── document:engineering-spec

group:engineering
  └── member: user:alice
  └── member: user:bob

user:carol  (owner of company-docs)
user:dave   (direct viewer of q4-report)
```

## Full Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "zanguard/pkg/engine"
    "zanguard/pkg/model"
    "zanguard/pkg/schema"
    "zanguard/pkg/storage/memory"
    "zanguard/pkg/tenant"
)

func main() {
    ctx := context.Background()
    store := memory.New()

    // 1. Set up tenant
    mgr := tenant.NewManager(store)
    mgr.Create(ctx, "acme", "Acme Corp", model.SchemaOwn)
    mgr.Activate(ctx, "acme")
    tCtx, _ := tenant.BuildContext(ctx, store, "acme")

    // 2. Load gdrive schema
    data, err := os.ReadFile("configs/examples/gdrive.zanguard.yaml")
    if err != nil {
        log.Fatal(err)
    }
    raw, _ := schema.Parse(data)
    cs, _ := schema.Compile(raw, data)

    eng := engine.New(store, engine.DefaultConfig())
    eng.LoadSchema("acme", cs)

    // 3. Write tuples

    write := func(objType, objID, rel, subType, subID, subRel string) {
        t := &model.RelationTuple{
            ObjectType:      objType,
            ObjectID:        objID,
            Relation:        rel,
            SubjectType:     subType,
            SubjectID:       subID,
            SubjectRelation: subRel,
        }
        if err := store.WriteTuple(tCtx, t); err != nil {
            log.Printf("WriteTuple: %v", err)
        }
    }

    // carol owns the folder
    write("folder", "company-docs", "owner", "user", "carol", "")

    // documents are inside the folder
    write("document", "q4-report", "parent", "folder", "company-docs", "")
    write("document", "engineering-spec", "parent", "folder", "company-docs", "")

    // dave is a direct viewer of q4-report only
    write("document", "q4-report", "viewer", "user", "dave", "")

    // engineering group members can view engineering-spec
    write("document", "engineering-spec", "viewer", "group", "engineering", "member")

    // alice and bob are members of engineering
    write("group", "engineering", "member", "user", "alice", "")
    write("group", "engineering", "member", "user", "bob", "")

    // 4. Check permissions

    check := func(label, objType, objID, perm, subType, subID string) {
        result, err := eng.Check(tCtx, &engine.CheckRequest{
            ObjectType: objType, ObjectID: objID,
            Permission: perm,
            SubjectType: subType, SubjectID: subID,
        })
        if err != nil {
            log.Printf("Check error: %v", err)
            return
        }
        status := "✗ DENY"
        if result.Allowed {
            status = "✓ ALLOW"
        }
        fmt.Printf("%-55s %s\n", label, status)
    }

    fmt.Println("\n=== Permission Checks ===\n")

    // Direct ownership
    check("carol can view q4-report (via folder ownership)",
        "document", "q4-report", "view", "user", "carol")

    check("carol can delete q4-report (via folder ownership)",
        "document", "q4-report", "delete", "user", "carol")

    // Direct viewer
    check("dave can view q4-report (direct viewer)",
        "document", "q4-report", "view", "user", "dave")

    check("dave CANNOT delete q4-report (not owner)",
        "document", "q4-report", "delete", "user", "dave")

    // Group membership
    check("alice can view engineering-spec (via group)",
        "document", "engineering-spec", "view", "user", "alice")

    check("bob can view engineering-spec (via group)",
        "document", "engineering-spec", "view", "user", "bob")

    // Isolation: dave has no access to engineering-spec
    check("dave CANNOT view engineering-spec (not in group, not in folder viewer)",
        "document", "engineering-spec", "view", "user", "dave")

    // Parent folder inheritance
    check("carol can view company-docs folder",
        "folder", "company-docs", "view", "user", "carol")
}
```

## Expected Output

```
=== Permission Checks ===

carol can view q4-report (via folder ownership)         ✓ ALLOW
carol can delete q4-report (via folder ownership)       ✓ ALLOW
dave can view q4-report (direct viewer)                 ✓ ALLOW
dave CANNOT delete q4-report (not owner)                ✗ DENY
alice can view engineering-spec (via group)             ✓ ALLOW
bob can view engineering-spec (via group)               ✓ ALLOW
dave CANNOT view engineering-spec (not in group)        ✗ DENY
carol can view company-docs folder                      ✓ ALLOW
```

## Key Patterns Demonstrated

| Pattern | Example |
|---------|---------|
| Direct ownership | `folder:company-docs#owner@user:carol` |
| Arrow traversal | `document:q4-report#parent→folder:company-docs→view` |
| Group membership | `group:engineering#member@user:alice` |
| Userset relation | `document:spec#viewer@group:engineering#member` |
| Permission isolation | dave can't see engineering-spec |

## See Also

- [Group Membership](./group-membership) — deep dive into userset patterns
- [ABAC Clearance](./abac-clearance) — add attribute-based guards
