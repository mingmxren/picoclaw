# Session Management + Command Stack Architecture Change (#959/#960/#961)

## Scope

This document separates architecture changes into two concerns:

- Commands path: channel adapters, agent command entry, and command package execution model.
- Session path: scope-aware indexing, active-pointer lifecycle, and persistence model.

## 1) Commands Architecture Change

### Before (upstream main)

```mermaid
flowchart LR
  U["User Input"] --> TG["Telegram Adapter"]
  U --> OTH["Other Channel Adapters"]

  TG --> TGLOCAL["telegram_commands.go\nlocal /help /show /list"]
  OTH --> AGCMD["AgentLoop.handleCommand\npartial command set"]
  TG --> AGCMD

  AGCMD --> AGSTATE["AgentLoop mutable state"]

  class TG,TGLOCAL p_before;
  class OTH,AGCMD,AGSTATE p_before;

  classDef p_before fill:#F5F5F5,stroke:#8C8C8C,stroke-width:1.5px,color:#1F1F1F;
```

### After (stacked PRs)

```mermaid
flowchart LR
  U2["User Input"] --> CH["Channel Adapters"]
  CH --> AGENTRY["AgentLoop command entry"]

  AGENTRY --> EX["commands.Executor\nhandled/passthrough"]
  EX --> REG["commands.Registry\ncanonical definitions"]
  EX --> HANDLERS["Builtin handlers\n/show /list /session"]

  CH --> TGREG["Telegram Start()\nasync command registration"]
  TGREG --> REG

  class CH,TGREG,REG p959;
  class EX,AGENTRY,HANDLERS p961;

  classDef p959 fill:#E6F4FF,stroke:#1677FF,stroke-width:2px,color:#0B2A4A;
  classDef p961 fill:#F6FFED,stroke:#52C41A,stroke-width:2px,color:#17380A;
```

### Command Impact

- Command definitions are globally visible and shared by all channels.
- Channel-specific support filtering is removed from `pkg/commands`; execution is now command-name driven.
- `/show channel` and `/list channels` remain user-visible features handled by builtin handlers.
- Telegram command menu sync still exists, but it now consumes the same canonical definitions.

## 2) Session Architecture Change

### Before (upstream main)

```mermaid
flowchart LR
  ROUTE0["Routing result / inbound session key"] --> SM0["SessionManager\nflat map by sessionKey"]
  SM0 --> FILES0["One JSON file per session"]
  SM0 --> ACTIVE0["Active session implicit\ncaller-managed"]

  class ROUTE0,SM0,FILES0,ACTIVE0 p_before;

  classDef p_before fill:#F5F5F5,stroke:#8C8C8C,stroke-width:1.5px,color:#1F1F1F;
```

### After (stacked PRs)

```mermaid
flowchart LR
  ROUTE1["Resolved scopeKey\n(dm/group/route)"] --> RT960["commands.Runtime\nScopeKey + SessionOps"]
  RT960 --> SM1["SessionManager\nscope-aware core"]

  SM1 --> IDX["index.json\nscopes.active + ordered list\npending deletes"]
  SM1 --> SFILES["Session JSON payloads"]

  SM1 --> OPS["ResolveActive / StartNew\nList / Resume / Prune"]
  OPS --> AGHANDLER["Agent command handlers\n/new /session ..."]

  class RT960,SM1,IDX,SFILES,OPS p960;
  class AGHANDLER p961;

  classDef p960 fill:#FFF7E6,stroke:#FA8C16,stroke-width:2px,color:#4A2A0B;
  classDef p961 fill:#F6FFED,stroke:#52C41A,stroke-width:2px,color:#17380A;
```

### Session Impact

- Session lifecycle is explicitly scope-aware instead of relying on a flat key convention.
- Active session pointer and ordered history are persisted in index metadata, enabling deterministic `list/resume` behavior.
- New-session rotation and prune are first-class operations with rollback/deferred-delete safeguards.
- Agent command runtime now consumes session operations through a narrow interface, reducing coupling.

## PR Layer Mapping

- #959: shared command registry model, channel integration points, and Telegram async registration baseline.
- #960: scope-aware `SessionManager`, persistent scope index, and lifecycle operations.
- #961: centralized command execution via runtime-backed executor and agent integration.
