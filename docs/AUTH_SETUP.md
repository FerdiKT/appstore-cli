# Direct App Store API Auth Setup

This CLI uses direct App Store API requests with **only** an access token.

No worker handshake, signature, API key, or private token is required.

---

## 1. Required credential

You only need:

- `ACCESS_TOKEN` (Bearer token)

---

## 2. Add profile

```bash
./appstore auth add prod --access-token '<ACCESS_TOKEN>'
```

Set as active:

```bash
./appstore auth use prod
```

Verify profile:

```bash
./appstore auth show
```

---

## 3. Test endpoints

```bash
./appstore search --keyword notes --storefront us --platform iphone
./appstore hints --term photo
./appstore app-details --app-id 1234567890 --storefront us --language en-GB --platform iphone
```

---

## 4. Multi-profile workflow

```bash
./appstore auth add personal --access-token '<TOKEN_1>'
./appstore auth add agency   --access-token '<TOKEN_2>'

./appstore auth use agency
./appstore search --profile personal --keyword calendar --storefront us
```

---

## 5. Config path

Profiles are stored in:

- `~/.config/appstore/config.json`

---

## 6. Troubleshooting

### `request failed with status 401`

1. Token expired or invalid
2. Wrong account/token scope
3. Profile token missing (`./appstore auth show`)

### `request failed with status 403`

Token is valid but not authorized for endpoint/account context.

### `request failed with status 429`

Rate limit on Apple side; retry later or reduce request rate.
