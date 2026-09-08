# Tuma254 API smoke tests

This collection is test client infrastructure only. It does not change API authentication, OTP generation, or database logic.

## Chosen approach

The collection uses the existing SMS webhook implementation and a protected local SMS sink. This avoids logging OTPs or adding an OTP retrieval endpoint to the API.

- production OTP behavior remains unchanged;
- OTPs are not written into application logs;
- the API is tested through its real webhook integration path;
- the local sink binds only to 127.0.0.1;
- writing and reading messages require the local sink credential.

## Local setup

Create one random local sink credential and use the same value in the API and Bruno environment.

Start the sink:

    set SMS_SINK_TOKEN=your-local-random-value
    go run ./cmd/sms-sink

Configure the API .env:

    SMS_PROVIDER=webhook
    SMS_WEBHOOK_URL=http://127.0.0.1:8090/messages
    SMS_WEBHOOK_TOKEN=your-local-random-value

Start the API in another terminal:

    go run ./cmd/api

Copy environments/local.example.bru to environments/local.bru and set the same sink credential plus unique test registration data.

Open api-tests as a Bruno collection and run:

1. Health
2. Register
3. Latest SMS
4. Verify Phone
5. Login
6. Current User
7. Refresh Session
8. Old Refresh Rejected
9. Logout
10. Logged Out Refresh Rejected

local.bru is ignored by Git because it can contain runtime tokens and the local sink credential.
