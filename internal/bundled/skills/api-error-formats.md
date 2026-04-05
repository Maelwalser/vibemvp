# API Error Formats Skill Guide

## Overview

Consistent error formats let API consumers parse errors programmatically. Two standards dominate: RFC 7807 Problem Details (W3C standard) and custom envelope patterns. Pick one and apply it everywhere.

## RFC 7807 Problem Details

```
Content-Type: application/problem+json
```

### JSON Structure

```json
{
  "type": "https://api.example.com/errors/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "The request body contains invalid fields.",
  "instance": "/api/v1/users/requests/req-abc123",
  "errors": [
    {
      "field": "email",
      "message": "Must be a valid email address",
      "code": "invalid_format"
    },
    {
      "field": "age",
      "message": "Must be between 0 and 150",
      "code": "out_of_range",
      "context": { "min": 0, "max": 150, "value": 200 }
    }
  ],
  "traceId": "d5f3a2b1-4c89-4f1e-9d2a-8b7c3e6f0a12"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | URI | Yes | Stable URI identifying the error type. Dereferenceable documentation preferred. |
| `title` | string | Yes | Human-readable summary. Should be stable (not change between occurrences). |
| `status` | integer | Yes | HTTP status code. Must match the actual response status. |
| `detail` | string | No | Human-readable explanation of this specific occurrence. |
| `instance` | URI | No | URI identifying this specific error occurrence (log correlation). |
| Custom fields | any | No | Extend freely — `errors`, `traceId`, `context` etc. |

### Standard Error Type URIs

```
https://api.example.com/errors/validation-error        → 422
https://api.example.com/errors/not-found               → 404
https://api.example.com/errors/unauthorized             → 401
https://api.example.com/errors/forbidden                → 403
https://api.example.com/errors/conflict                 → 409
https://api.example.com/errors/rate-limit-exceeded      → 429
https://api.example.com/errors/internal-error           → 500

# Use "about:blank" as type if you have no documentation URI
{ "type": "about:blank", "title": "Not Found", "status": 404 }
```

## Custom Envelope Pattern

Used when RFC 7807 is overkill or when your API predates it.

```json
{
  "success": false,
  "code": "VALIDATION_ERROR",
  "message": "Request validation failed",
  "data": null,
  "errors": [
    { "field": "email", "message": "Invalid email format", "code": "INVALID_FORMAT" },
    { "field": "password", "message": "Must be at least 8 characters", "code": "TOO_SHORT" }
  ],
  "meta": {
    "requestId": "req-abc123",
    "timestamp": "2026-04-02T10:30:00Z"
  }
}
```

## Per-Framework Error Middleware

### Express (TypeScript) — RFC 7807

```typescript
import { Request, Response, NextFunction } from "express";

interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  [key: string]: unknown;
}

// Custom error classes
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public detail?: string,
    public extensions: Record<string, unknown> = {},
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export class ValidationError extends ApiError {
  constructor(public fieldErrors: Array<{ field: string; message: string; code: string }>) {
    super(422, "VALIDATION_ERROR", "Validation failed");
  }
}

export class NotFoundError extends ApiError {
  constructor(resource: string, id: string) {
    super(404, "NOT_FOUND", `${resource} not found`, `No ${resource} with id ${id}`);
  }
}

// Global error handler (MUST have 4 params for Express to recognize it)
export function errorHandler(err: Error, req: Request, res: Response, _next: NextFunction) {
  if (err instanceof ValidationError) {
    const problem: ProblemDetail = {
      type: "https://api.example.com/errors/validation-error",
      title: "Validation Error",
      status: 422,
      detail: "One or more fields are invalid",
      instance: req.path,
      errors: err.fieldErrors,
      traceId: req.headers["x-request-id"],
    };
    return res.status(422).json(problem);
  }

  if (err instanceof ApiError) {
    const problem: ProblemDetail = {
      type: `https://api.example.com/errors/${err.code.toLowerCase().replace(/_/g, "-")}`,
      title: err.message,
      status: err.status,
      detail: err.detail,
      instance: req.path,
      traceId: req.headers["x-request-id"],
      ...err.extensions,
    };
    return res.status(err.status)
      .set("Content-Type", "application/problem+json")
      .json(problem);
  }

  // Unknown error — don't leak details
  console.error("Unhandled error:", err);
  return res.status(500)
    .set("Content-Type", "application/problem+json")
    .json({
      type: "https://api.example.com/errors/internal-error",
      title: "Internal Server Error",
      status: 500,
      instance: req.path,
      traceId: req.headers["x-request-id"],
    });
}

// Register last in the middleware chain
app.use(errorHandler);
```

### FastAPI (Python) — RFC 7807

```python
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
from pydantic import BaseModel
from typing import Any

app = FastAPI()

class ProblemDetail(BaseModel):
    type: str
    title: str
    status: int
    detail: str | None = None
    instance: str | None = None

def problem_response(detail: ProblemDetail, extra: dict[str, Any] | None = None) -> JSONResponse:
    body = detail.model_dump(exclude_none=True)
    if extra:
        body.update(extra)
    return JSONResponse(
        content=body,
        status_code=detail.status,
        headers={"Content-Type": "application/problem+json"},
    )

# Override default Pydantic validation errors
@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    errors = [
        {"field": ".".join(str(loc) for loc in e["loc"][1:]), "message": e["msg"], "code": e["type"]}
        for e in exc.errors()
    ]
    return problem_response(
        ProblemDetail(
            type="https://api.example.com/errors/validation-error",
            title="Validation Error",
            status=422,
            detail="One or more fields are invalid",
            instance=str(request.url.path),
        ),
        extra={"errors": errors, "traceId": request.headers.get("x-request-id")},
    )

# Custom exception
class NotFoundError(Exception):
    def __init__(self, resource: str, id: str):
        self.resource = resource
        self.id = id

@app.exception_handler(NotFoundError)
async def not_found_handler(request: Request, exc: NotFoundError):
    return problem_response(ProblemDetail(
        type="https://api.example.com/errors/not-found",
        title="Not Found",
        status=404,
        detail=f"No {exc.resource} with id {exc.id}",
        instance=str(request.url.path),
    ))
```

### Go (net/http)

```go
package apierror

import (
    "encoding/json"
    "net/http"
)

type ProblemDetail struct {
    Type     string      `json:"type"`
    Title    string      `json:"title"`
    Status   int         `json:"status"`
    Detail   string      `json:"detail,omitempty"`
    Instance string      `json:"instance,omitempty"`
    Errors   interface{} `json:"errors,omitempty"`
    TraceID  string      `json:"traceId,omitempty"`
}

func WriteProblem(w http.ResponseWriter, r *http.Request, p ProblemDetail) {
    if p.Instance == "" {
        p.Instance = r.URL.Path
    }
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(p.Status)
    json.NewEncoder(w).Encode(p)
}

func WriteNotFound(w http.ResponseWriter, r *http.Request, resource string) {
    WriteProblem(w, r, ProblemDetail{
        Type:   "https://api.example.com/errors/not-found",
        Title:  "Not Found",
        Status: http.StatusNotFound,
        Detail: resource + " not found",
    })
}

func WriteValidationError(w http.ResponseWriter, r *http.Request, errs []FieldError) {
    WriteProblem(w, r, ProblemDetail{
        Type:   "https://api.example.com/errors/validation-error",
        Title:  "Validation Error",
        Status: http.StatusUnprocessableEntity,
        Errors: errs,
    })
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}
```

### Spring Boot (Java)

```java
@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<Map<String, Object>> handleValidation(
            MethodArgumentNotValidException ex, HttpServletRequest request) {

        List<Map<String, String>> errors = ex.getBindingResult().getFieldErrors().stream()
            .map(e -> Map.of(
                "field", e.getField(),
                "message", e.getDefaultMessage(),
                "code", e.getCode()
            ))
            .toList();

        return ResponseEntity.status(422)
            .contentType(MediaType.parseMediaType("application/problem+json"))
            .body(Map.of(
                "type", "https://api.example.com/errors/validation-error",
                "title", "Validation Error",
                "status", 422,
                "instance", request.getRequestURI(),
                "errors", errors
            ));
    }

    @ExceptionHandler(EntityNotFoundException.class)
    public ResponseEntity<Map<String, Object>> handleNotFound(
            EntityNotFoundException ex, HttpServletRequest request) {
        return ResponseEntity.status(404)
            .contentType(MediaType.parseMediaType("application/problem+json"))
            .body(Map.of(
                "type", "https://api.example.com/errors/not-found",
                "title", "Not Found",
                "status", 404,
                "detail", ex.getMessage(),
                "instance", request.getRequestURI()
            ));
    }
}
```

## Rules

- Set `Content-Type: application/problem+json` for RFC 7807 responses (not `application/json`)
- Always include a machine-readable `code` or `type` field — never rely only on the human-readable message
- Never expose stack traces, SQL errors, or internal paths in production error responses
- Use a single global error handler / middleware — never serialize errors inline in route handlers
- Map all unhandled exceptions to 500 with a generic message — let logging capture the details
- Include a `traceId` / `requestId` in every error response to correlate with server logs
- Return `errors[]` array with field-level details for validation failures — not a single string message
