# Migration Scripts Guide

## Overview

Two migration scripts are provided to migrate data from old project formats to the new multi-scenario architecture:

1. **migrate_legacy** - For old format without template files (like `data/test`)
2. **migrate_with_templates** - For new format with template files and assets (like `data/test2`)

## Building the Scripts

```bash
# Build migrate_legacy script
go build -o .bin/migrate_legacy scripts/migrate_legacy/main.go

# Build migrate_with_templates script
go build -o .bin/migrate_with_templates scripts/migrate_with_templates/main.go
```

## Using the Scripts

### Migrate from Old Format (without templates)

```bash
./.bin/migrate_legacy data/test
```

### Migrate from New Format (with templates and assets)

```bash
./.bin/migrate_with_templates data/test2
```

## What Gets Migrated

Both scripts migrate:

1. **Users** - All user data with payment history and scenario state
2. **Messages** - Message configurations and templates
3. **Schedule Backup** - Message scheduling data
4. **Scenario Config** - Configuration with values from `.env` file
5. **Assets** - Images and other assets (only in migrate_with_templates)
6. **Commands** - Command configurations (only in migrate_legacy)

## Configuration

The scripts read configuration from the `.env` file in the project root:

- `PRODAMUS_PRODUCT_NAME` - Product name for the scenario
- `PRODAMUS_PRODUCT_PRICE` - Product price
- `PRODAMUS_PRODUCT_PAID_CONTENT` - Paid content message
- `PRIVATE_GROUP_ID` - Private Telegram group ID

## Output Structure

After migration, the data will be organized as:

```
data/
├── scenarios/
│   ├── registry.json          # Scenario registry
│   └── default/               # Default scenario
│       ├── config.json        # Scenario configuration
│       └── messages.json      # Message configurations
│           └── messages/      # Message templates (.md files)
├── assets/                    # Migrated assets (if any)
├── buttons/
│   └── registry.json          # Button registry
├── templates/
│   └── bot_globals.json       # Bot global variables
├── schedules/
│   └── backup.json            # Schedule backup
├── users.json                 # Migrated users
└── commands.json              # Migrated commands (if any)
```

## Backup

Both scripts automatically create backups in the source directory:

```
data/test/
└── migration_backup/          # Backup of all migrated files
```

## Verification

After migration, the scripts verify that all required files exist:

- `data/scenarios/registry.json`
- `data/scenarios/default/config.json`
- `data/scenarios/default/messages.json`
- `data/buttons/registry.json`
- `data/templates/bot_globals.json`
- `data/users.json`

## Running the Bot After Migration

After successful migration, you can run the bot:

```bash
make run-dev
# or
./.bin/bot
```

## Troubleshooting

### Migration fails with "failed to load .env file"

Ensure the `.env` file exists in the project root with all required variables.

### Users are not migrated correctly

Check that the source `users.json` file is valid JSON and contains user data.

### Messages are not migrated correctly

- For old format: Ensure `messages.json` has the correct structure with `messages_list` and `messages`
- For new format: Ensure `messages.json` has `timing` as objects (`{hours, minutes}`) and `inline_keyboard` structure
