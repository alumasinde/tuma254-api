# Tuma254 API Tests

## Bruno

Open this directory as the Bruno collection:

`tests/bruno`

Select the local environment from:

`environments/Local.bru`

Run the API first:

```powershell
go run ./cmd/api
```

Then test Infrastructure before Identity:

1. Health
2. Ready
3. API Info
4. Register
5. Login
6. Refresh
7. Logout

Never commit production credentials, tokens, database passwords, JWT secrets, M-Pesa credentials, or real production passwords.
