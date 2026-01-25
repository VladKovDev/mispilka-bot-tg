# Channel Support Migration Guide

## For Existing Users (Group Mode)

If you're currently using the bot with a private group, you need to update your `.env` file:

### Before:
```bash
PRIVATE_GROUP_ID=123456789
```

### After:
```bash
PRIVATE_RESOURCE_ID=123456789
PRIVATE_RESOURCE_TYPE=group
```

## For New Users (Channel Mode)

To use the bot with a private channel:

```bash
PRIVATE_RESOURCE_ID=123456789
PRIVATE_RESOURCE_TYPE=channel
```

## Key Differences

| Feature | Group Mode | Channel Mode |
|---------|------------|--------------|
| Join/leave tracking | ✅ Tracked via chat_member events | ❌ Not tracked |
| Invite link revocation | ✅ Explicit revoke after join | ❌ Not needed (member_limit=1) |
| JoinedGroup field | Set to true after join | Stays false |
| JoinedAt field | Set to join timestamp | nil |

## Testing

After updating your configuration:

1. Restart the bot
2. For group mode: verify join/leave still works
3. For channel mode: verify users can join via invite link
