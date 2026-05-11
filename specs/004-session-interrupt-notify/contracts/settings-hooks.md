# Contract: ~/.claude/settings.json Hook Configuration

`cs setup` merges the following entries into `~/.claude/settings.json`.
Existing keys outside `hooks` are preserved verbatim.

## Installed Hook Shape

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "permission_prompt|idle_prompt|elicitation_dialog",
        "hooks": [
          {
            "type": "command",
            "command": "~/.local/share/cs/notify.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "~/.local/share/cs/notify.sh"
          }
        ]
      }
    ]
  }
}
```

## Merge Rules

- If a `Notification` or `Stop` key already exists in `hooks`, the cs entry is appended to the
  existing array (not replaced).
- If an identical `command` entry already exists for the same event type, no duplicate is added
  (idempotent).
- All other top-level keys in `settings.json` are written back unchanged.

## Removal Rules

`cs setup` remove path strips any `hooks` entry whose `command` field equals
`~/.local/share/cs/notify.sh`. If the resulting array is empty, the event type key is removed.
If the `hooks` object becomes empty, the `hooks` key itself is removed.

## Atomic Write

Settings are updated via: write to temp file → `os.Rename` to target path.
This prevents partial writes on crash or signal.
