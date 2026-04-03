# Multi-Factor Authentication (MFA) Skill Guide

## Overview

MFA requires users to provide a second factor beyond their password. The three common factors are TOTP (time-based one-time passwords), SMS/email OTP, and WebAuthn/Passkeys (hardware-bound credentials). Implement at least TOTP and backup codes for all sensitive applications.

---

## TOTP (Time-Based One-Time Passwords)

TOTP generates a 6-digit code that changes every 30 seconds. Users enroll by scanning a QR code with an authenticator app (Google Authenticator, Authy, 1Password).

### Go — TOTP with `pquerna/otp`

```go
import "github.com/pquerna/otp/totp"

// Enrollment: generate a new TOTP secret for a user
func GenerateTOTPSecret(accountName, issuer string) (secret, qrURI string, err error) {
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      issuer,
        AccountName: accountName,
        SecretSize:  20, // 160-bit secret
    })
    if err != nil {
        return "", "", fmt.Errorf("generate TOTP key: %w", err)
    }
    return key.Secret(), key.URL(), nil
    // key.URL() is the otpauth:// URI for QR code generation
}

// Verification with ±1 window tolerance (30s before and after)
func VerifyTOTP(secret, code string) bool {
    return totp.Validate(code, secret) // validates current window + adjacent windows
}

// Store encrypted TOTP secret in DB, never plaintext
func storeTOTPSecret(userID, secret string) error {
    encrypted, err := encryptAES256(secret, os.Getenv("ENCRYPTION_KEY"))
    if err != nil {
        return fmt.Errorf("encrypt TOTP secret: %w", err)
    }
    _, err = db.Exec(`UPDATE users SET totp_secret = $1, totp_enabled = TRUE WHERE id = $2`,
        encrypted, userID)
    return err
}
```

### Python — TOTP with `pyotp`

```python
import pyotp
import qrcode
import io
import base64

def generate_totp_secret(account_name: str, issuer: str) -> tuple[str, str]:
    secret = pyotp.random_base32()
    uri = pyotp.totp.TOTP(secret).provisioning_uri(
        name=account_name,
        issuer_name=issuer,
    )
    return secret, uri

def generate_qr_code_base64(uri: str) -> str:
    img = qrcode.make(uri)
    buffer = io.BytesIO()
    img.save(buffer, format="PNG")
    return base64.b64encode(buffer.getvalue()).decode()

def verify_totp(secret: str, code: str) -> bool:
    totp = pyotp.TOTP(secret)
    return totp.verify(code, valid_window=1)  # ±1 window = ±30 seconds
```

### TypeScript — TOTP with `speakeasy`

```typescript
import speakeasy from 'speakeasy'
import qrcode from 'qrcode'

function generateTOTPSecret(accountName: string, issuer: string) {
  const secret = speakeasy.generateSecret({
    name: `${issuer} (${accountName})`,
    issuer,
    length: 20,
  })
  return { secret: secret.base32, otpauthURL: secret.otpauth_url! }
}

async function generateQRCode(otpauthURL: string): Promise<string> {
  return qrcode.toDataURL(otpauthURL) // base64 PNG for embedding in img src
}

function verifyTOTP(secret: string, token: string): boolean {
  return speakeasy.totp.verify({
    secret,
    encoding: 'base32',
    token,
    window: 1, // ±1 30-second window
  })
}
```

---

## Backup Codes

One-time backup codes allow account recovery if the authenticator device is lost.

```go
// Generate 10 single-use backup codes
func GenerateBackupCodes() ([]string, error) {
    codes := make([]string, 10)
    for i := range codes {
        b := make([]byte, 5)
        if _, err := rand.Read(b); err != nil {
            return nil, fmt.Errorf("generate backup code: %w", err)
        }
        codes[i] = fmt.Sprintf("%x", b) // 10-char hex code
    }
    return codes, nil
}

// Hash and store backup codes
func StoreBackupCodes(userID string, codes []string) error {
    for _, code := range codes {
        hash := sha256.Sum256([]byte(code))
        _, err := db.Exec(
            `INSERT INTO backup_codes (user_id, code_hash, used) VALUES ($1, $2, FALSE)`,
            userID, hex.EncodeToString(hash[:]),
        )
        if err != nil {
            return fmt.Errorf("store backup code: %w", err)
        }
    }
    return nil
}

// Consume a backup code (one-time use)
func ConsumeBackupCode(userID, code string) (bool, error) {
    hash := hex.EncodeToString(sha256.Sum256([]byte(code))[:])
    result, err := db.Exec(
        `UPDATE backup_codes SET used = TRUE, used_at = NOW()
          WHERE user_id = $1 AND code_hash = $2 AND used = FALSE`,
        userID, hash,
    )
    if err != nil {
        return false, fmt.Errorf("consume backup code: %w", err)
    }
    return result.RowsAffected() == 1, nil
}
```

---

## SMS OTP

```typescript
import twilio from 'twilio'

const twilioClient = twilio(
  process.env.TWILIO_ACCOUNT_SID,
  process.env.TWILIO_AUTH_TOKEN
)

// Send OTP
async function sendSMSOTP(phone: string, userID: string): Promise<void> {
  const otp = Math.floor(100000 + Math.random() * 900000).toString() // 6 digits
  const expiresAt = new Date(Date.now() + 5 * 60 * 1000) // 5 minutes

  // Store hashed OTP
  const hash = crypto.createHash('sha256').update(otp).digest('hex')
  await db.query(
    `INSERT INTO otp_codes (user_id, code_hash, expires_at, attempts)
     VALUES ($1, $2, $3, 0)
     ON CONFLICT (user_id) DO UPDATE
     SET code_hash = $2, expires_at = $3, attempts = 0`,
    [userID, hash, expiresAt]
  )

  await twilioClient.messages.create({
    to: phone,
    from: process.env.TWILIO_FROM_NUMBER,
    body: `Your verification code is ${otp}. Expires in 5 minutes.`,
  })
}

// Verify OTP with brute-force protection
async function verifySMSOTP(userID: string, code: string): Promise<boolean> {
  const { rows } = await db.query(
    `SELECT code_hash, expires_at, attempts FROM otp_codes
      WHERE user_id = $1 AND expires_at > NOW()`,
    [userID]
  )
  if (!rows[0]) return false

  if (rows[0].attempts >= 5) {
    throw new Error('Too many attempts — request a new code')
  }

  // Increment attempts before checking
  await db.query(
    `UPDATE otp_codes SET attempts = attempts + 1 WHERE user_id = $1`,
    [userID]
  )

  const hash = crypto.createHash('sha256').update(code).digest('hex')
  if (hash !== rows[0].code_hash) return false

  // Invalidate after successful use
  await db.query(`DELETE FROM otp_codes WHERE user_id = $1`, [userID])
  return true
}
```

---

## Email OTP

```python
import secrets
import hashlib
from datetime import datetime, timedelta, timezone
from django.core.mail import send_mail

def send_email_otp(user) -> None:
    otp = secrets.randbelow(900000) + 100000  # 6-digit
    otp_str = str(otp)
    expires_at = datetime.now(timezone.utc) + timedelta(minutes=10)

    user.otp_hash = hashlib.sha256(otp_str.encode()).hexdigest()
    user.otp_expires_at = expires_at
    user.otp_attempts = 0
    user.save(update_fields=["otp_hash", "otp_expires_at", "otp_attempts"])

    send_mail(
        subject="Your verification code",
        message=f"Your code is {otp_str}. It expires in 10 minutes.",
        from_email="noreply@yourapp.com",
        recipient_list=[user.email],
    )

def verify_email_otp(user, code: str) -> bool:
    if user.otp_attempts >= 5:
        raise ValueError("Too many attempts — request a new code")
    if not user.otp_expires_at or user.otp_expires_at < datetime.now(timezone.utc):
        raise ValueError("Code expired")

    user.otp_attempts += 1
    user.save(update_fields=["otp_attempts"])

    provided_hash = hashlib.sha256(code.encode()).hexdigest()
    if provided_hash != user.otp_hash:
        return False

    user.otp_hash = None
    user.otp_expires_at = None
    user.save(update_fields=["otp_hash", "otp_expires_at"])
    return True
```

---

## WebAuthn / Passkeys

```typescript
import {
  generateRegistrationOptions,
  verifyRegistrationResponse,
  generateAuthenticationOptions,
  verifyAuthenticationResponse,
} from '@simplewebauthn/server'

const rpName = 'Your App'
const rpID = 'yourapp.com'
const origin = 'https://yourapp.com'

// Registration ceremony — Step 1: generate options
async function beginRegistration(userID: string, userName: string) {
  const options = await generateRegistrationOptions({
    rpName,
    rpID,
    userID,
    userName,
    attestationType: 'none',
    authenticatorSelection: {
      residentKey: 'preferred',
      userVerification: 'preferred',
    },
  })
  // Store challenge in session for verification
  req.session.registrationChallenge = options.challenge
  return options
}

// Registration ceremony — Step 2: verify response
async function finishRegistration(userID: string, response: any) {
  const verification = await verifyRegistrationResponse({
    response,
    expectedChallenge: req.session.registrationChallenge,
    expectedOrigin: origin,
    expectedRPID: rpID,
  })

  if (!verification.verified) throw new Error('Registration failed')

  const { credentialID, credentialPublicKey, counter } =
    verification.registrationInfo!

  // Store credential (public key only — private key never leaves device)
  await db.query(
    `INSERT INTO webauthn_credentials (user_id, credential_id, public_key, counter)
     VALUES ($1, $2, $3, $4)`,
    [userID, credentialID, credentialPublicKey, counter]
  )
}

// Authentication ceremony — Step 1: generate options
async function beginAuthentication(userID: string) {
  const credentials = await getUserCredentials(userID)
  const options = await generateAuthenticationOptions({
    rpID,
    allowCredentials: credentials.map(c => ({
      id: c.credential_id,
      type: 'public-key',
    })),
    userVerification: 'preferred',
  })
  req.session.authChallenge = options.challenge
  return options
}

// Authentication ceremony — Step 2: verify assertion
async function finishAuthentication(userID: string, response: any) {
  const credential = await getCredentialByID(response.id)
  const verification = await verifyAuthenticationResponse({
    response,
    expectedChallenge: req.session.authChallenge,
    expectedOrigin: origin,
    expectedRPID: rpID,
    authenticator: {
      credentialID: credential.credential_id,
      credentialPublicKey: credential.public_key,
      counter: credential.counter,
    },
  })

  if (!verification.verified) throw new Error('Authentication failed')

  // Update counter (replay attack prevention)
  await db.query(
    `UPDATE webauthn_credentials SET counter = $1 WHERE credential_id = $2`,
    [verification.authenticationInfo.newCounter, credential.credential_id]
  )
}
```

---

## Security Rules

- Store TOTP secrets encrypted at rest (AES-256); never store them plaintext.
- Enforce a maximum of 5 failed OTP attempts before requiring a new code or temporary lockout.
- Hash backup codes with SHA-256 before storage; mark them used atomically.
- SMS OTP: 5-minute maximum TTL; never reuse across sessions.
- WebAuthn: always verify the signature counter increases to prevent replay attacks.
- Enforce TOTP enrollment before granting access to sensitive operations; don't allow permanent bypass.

---

## Key Rules

- TOTP: `valid_window=1` allows ±30 seconds clock skew tolerance.
- QR code URI format: `otpauth://totp/Issuer:account?secret=BASE32&issuer=Issuer`.
- Backup codes: 10 single-use codes, hashed with SHA-256, marked used atomically.
- SMS/email OTP: hash before storage, 5-attempt limit, short TTL (5–10 min).
- WebAuthn: registration stores public key only; authentication verifies signature; increment counter.
- Rate-limit all OTP endpoints aggressively (separate limit from regular API rate limits).
