# CAPTCHA & Bot Verification Skill Guide

## Overview

CAPTCHA systems distinguish humans from bots at form submission, login, and high-value action endpoints. Four major providers: Google reCAPTCHA v2/v3, hCaptcha, and Cloudflare Turnstile. All follow the same pattern: client widget generates a token → server verifies token with provider API → allow or reject.

---

## reCAPTCHA v2

### Checkbox Widget

```html
<!-- Load script -->
<script src="https://www.google.com/recaptcha/api.js" async defer></script>

<!-- Render widget -->
<div class="g-recaptcha" data-sitekey="YOUR_SITE_KEY"></div>
<button type="submit">Submit</button>
```

### Invisible reCAPTCHA v2

```html
<script src="https://www.google.com/recaptcha/api.js" async defer></script>
<button class="g-recaptcha"
        data-sitekey="YOUR_SITE_KEY"
        data-callback="onSubmit"
        data-action="submit">Submit</button>

<script>
function onSubmit(token) {
  document.getElementById("myForm").submit();
}
</script>
```

### Server Verification

POST to `https://www.google.com/recaptcha/api/siteverify`:

```
secret=YOUR_SECRET_KEY&response=TOKEN_FROM_CLIENT&remoteip=USER_IP
```

Response:
```json
{
  "success": true,
  "challenge_ts": "2024-01-01T12:00:00Z",
  "hostname": "yoursite.com",
  "error-codes": []
}
```

Go verification example:
```go
func verifyRecaptchaV2(token, secret, userIP string) (bool, error) {
    resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
        "secret":   {secret},
        "response": {token},
        "remoteip": {userIP},
    })
    if err != nil {
        return false, fmt.Errorf("captcha request: %w", err)
    }
    defer resp.Body.Close()
    var result struct {
        Success bool `json:"success"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, fmt.Errorf("captcha decode: %w", err)
    }
    return result.Success, nil
}
```

---

## reCAPTCHA v3

### Score-Based Verification (No User Interaction)

```html
<script src="https://www.google.com/recaptcha/api.js?render=YOUR_SITE_KEY"></script>
<script>
grecaptcha.ready(function() {
    grecaptcha.execute('YOUR_SITE_KEY', {action: 'submit'}).then(function(token) {
        document.getElementById('g-recaptcha-response').value = token;
    });
});
</script>
```

### Server Verification with Score

```go
type RecaptchaV3Response struct {
    Success     bool     `json:"success"`
    Score       float64  `json:"score"`       // 0.0 (bot) to 1.0 (human)
    Action      string   `json:"action"`
    ChallengeTS string   `json:"challenge_ts"`
    Hostname    string   `json:"hostname"`
    ErrorCodes  []string `json:"error-codes"`
}

func verifyRecaptchaV3(token, secret, expectedAction string) (bool, error) {
    resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
        "secret":   {secret},
        "response": {token},
    })
    if err != nil {
        return false, fmt.Errorf("captcha request: %w", err)
    }
    defer resp.Body.Close()

    var result RecaptchaV3Response
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, fmt.Errorf("captcha decode: %w", err)
    }

    const scoreThreshold = 0.5
    if !result.Success || result.Score < scoreThreshold {
        return false, nil
    }
    if result.Action != expectedAction {
        return false, nil  // Action mismatch — possible token replay
    }
    return true, nil
}
```

**Score thresholds:**
- `>= 0.7` — High confidence human; allow
- `0.5 – 0.7` — Borderline; require v2 challenge as fallback
- `< 0.5` — Likely bot; block or add friction

**Adaptive blocking:** step up to v2 challenge when score is 0.3–0.5 rather than hard-blocking.

---

## hCaptcha

### Widget

```html
<script src="https://js.hcaptcha.com/1/api.js" async defer></script>
<div class="h-captcha" data-sitekey="YOUR_SITE_KEY"></div>
```

### Server Verification

POST to `https://api.hcaptcha.com/siteverify`:
```
secret=YOUR_SECRET&response=TOKEN&remoteip=USER_IP
```

```go
func verifyHCaptcha(token, secret, userIP string) (bool, error) {
    resp, err := http.PostForm("https://api.hcaptcha.com/siteverify", url.Values{
        "secret":   {secret},
        "response": {token},
        "remoteip": {userIP},
    })
    if err != nil {
        return false, fmt.Errorf("hcaptcha request: %w", err)
    }
    defer resp.Body.Close()
    var result struct {
        Success bool     `json:"success"`
        Errors  []string `json:"error-codes"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, fmt.Errorf("hcaptcha decode: %w", err)
    }
    return result.Success, nil
}
```

---

## Cloudflare Turnstile

### Widget Modes

| Mode | Behavior |
|------|----------|
| `managed` | CF decides when to show a challenge |
| `non-interactive` | Always non-interactive; may silently pass |
| `invisible` | No widget shown at all; purely signal-based |

```html
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>

<!-- Managed mode -->
<div class="cf-turnstile" data-sitekey="YOUR_SITE_KEY" data-theme="auto"></div>

<!-- Invisible mode -->
<div class="cf-turnstile"
     data-sitekey="YOUR_SITE_KEY"
     data-callback="turnstileCallback"
     data-appearance="interaction-only"></div>
```

### Server Verification

POST to `https://challenges.cloudflare.com/turnstile/v0/siteverify`:
```json
{ "secret": "YOUR_SECRET", "response": "TOKEN", "remoteip": "USER_IP" }
```

```go
func verifyTurnstile(token, secret, userIP string) (bool, error) {
    body, _ := json.Marshal(map[string]string{
        "secret":   secret,
        "response": token,
        "remoteip": userIP,
    })
    resp, err := http.Post(
        "https://challenges.cloudflare.com/turnstile/v0/siteverify",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return false, fmt.Errorf("turnstile request: %w", err)
    }
    defer resp.Body.Close()
    var result struct {
        Success bool `json:"success"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return false, fmt.Errorf("turnstile decode: %w", err)
    }
    return result.Success, nil
}
```

---

## Token Expiry & Single-Use Enforcement

Tokens expire after **2 minutes** (reCAPTCHA, hCaptcha) or per CF session. Enforce single-use server-side:

```go
// Redis-based token consumption
func consumeToken(ctx context.Context, rdb *redis.Client, token string) (bool, error) {
    key := "captcha:used:" + token
    // SET NX with 5-minute TTL — fails if key exists
    ok, err := rdb.SetNX(ctx, key, "1", 5*time.Minute).Result()
    if err != nil {
        return false, fmt.Errorf("redis setnx: %w", err)
    }
    return ok, nil  // false = token already used
}
```

Verification flow:
1. Verify token with provider API
2. Call `consumeToken` — reject if already consumed
3. Proceed with business action only after both pass

---

## Key Rules

- Store `sitekey` in client config / env vars; store `secret` only server-side — never expose secret to browser
- Always verify tokens server-side; never trust client-side verification alone
- Enforce token single-use with Redis or DB to prevent replay attacks
- For v3, always validate the `action` field matches what you expected
- Do not cache CAPTCHA verification results beyond single-use invalidation
- Log verification failures with IP and endpoint for abuse pattern detection
- Use Cloudflare Turnstile as a privacy-friendly drop-in when GDPR compliance is required
