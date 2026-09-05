# Tuma254 API Tests

## Bruno

Open this directory as the Bruno collection:

`tests/bruno`

Bruno discovers the runnable local environment from:

`tests/bruno/environments/Local.bru`

The repository root also keeps:

`environments/Local.bru`

as the shared environment reference. Both files use the same local values.

Run the API first:

```powershell
go run ./cmd/api
```

Then select **Local** in Bruno and test:

1. Health
2. Ready
3. API Info
4. Register
5. Login
6. Refresh
7. Logout

Never commit production credentials, tokens, database passwords, JWT secrets, M-Pesa credentials, or real production passwords.
