# HTMX Skill Guide

## Concept

HTMX allows HTML elements to make HTTP requests and swap fragments of the page. The server returns HTML (not JSON), keeping logic server-side. JavaScript is optional.

## Project Layout

```
project/
├── main.go (or app.py / server.js)
├── templates/
│   ├── base.html           # Full page shell
│   ├── partials/           # HTML fragments for HTMX responses
│   │   ├── user-row.html
│   │   └── user-list.html
│   └── pages/
│       └── users.html
└── static/
    └── htmx.min.js         # htmx 2.x
```

## Setup

```html
<!DOCTYPE html>
<html>
<head>
  <meta name="htmx-config" content='{"defaultSwapStyle":"outerHTML"}' />
  <script src="/static/htmx.min.js" defer></script>
</head>
<body hx-boost="true">  <!-- upgrade all links/forms to AJAX -->
  ...
</body>
</html>
```

## Core Attributes

```html
<!-- GET request, replace #result innerHTML -->
<button hx-get="/api/users" hx-target="#result" hx-swap="innerHTML">
  Load Users
</button>
<div id="result"></div>

<!-- POST on form submit -->
<form hx-post="/api/users" hx-target="#list" hx-swap="beforeend">
  <input name="name" required />
  <button type="submit">Add</button>
</form>

<!-- PUT / DELETE -->
<button hx-put="/api/users/123" hx-target="closest li" hx-swap="outerHTML">
  Save
</button>
<button hx-delete="/api/users/123" hx-target="closest li" hx-swap="delete"
        hx-confirm="Delete this user?">
  Delete
</button>
```

## hx-swap Modes

| Mode | Effect |
|------|--------|
| `innerHTML` | Replace target's inner HTML (default) |
| `outerHTML` | Replace the target element itself |
| `beforeend` | Append inside target (list append) |
| `afterend` | Insert after target element |
| `beforebegin` | Insert before target element |
| `afterbegin` | Prepend inside target |
| `delete` | Remove the target element |
| `none` | Don't swap (use for side effects only) |

## hx-trigger

```html
<!-- Default: click for buttons, submit for forms, change for inputs -->

<!-- Custom trigger -->
<input hx-get="/search" hx-target="#results" hx-trigger="keyup changed delay:300ms"
       name="q" />

<!-- Poll every 2 seconds -->
<div hx-get="/api/status" hx-trigger="every 2s" hx-target="this">
  Checking...
</div>

<!-- On event from another element -->
<div hx-get="/api/detail" hx-trigger="itemSelected from:body" hx-target="#detail">
</div>

<!-- On page load -->
<div hx-get="/api/dashboard" hx-trigger="load" hx-target="this">
  Loading dashboard...
</div>
```

## hx-target Selectors

```html
<!-- CSS selector -->
hx-target="#result"
hx-target=".notification"

<!-- Relative selectors -->
hx-target="this"            <!-- the element itself -->
hx-target="closest tr"      <!-- nearest ancestor matching -->
hx-target="next .error"     <!-- next sibling matching -->
hx-target="previous input"  <!-- previous sibling matching -->
```

## Server-Sent Events (SSE)

```html
<!-- Connect to SSE endpoint -->
<div hx-sse="connect:/events">
  <!-- Inner element swapped on matching event name -->
  <div hx-sse="swap:notification" hx-target="#notifications" hx-swap="beforeend">
  </div>
</div>
```

Server sends:
```
event: notification
data: <li>New message arrived</li>

```

## History / Push URL

```html
<!-- Push URL to browser history when request completes -->
<a hx-get="/users/123" hx-target="#content" hx-push-url="true">
  View User
</a>

<!-- Explicitly set the pushed URL -->
<button hx-get="/api/search?q=foo" hx-target="#results" hx-push-url="/search?q=foo">
  Search
</button>
```

## Response Headers (Server-Side)

```
HX-Redirect: /login          → client-side redirect
HX-Refresh: true             → full page refresh
HX-Trigger: itemDeleted      → fire JS event on client
HX-Trigger-After-Swap: saved → fire event after swap
HX-Retarget: #errors         → override hx-target
HX-Reswap: outerHTML         → override hx-swap
HX-Push-Url: /new-url        → push URL from server
```

## Request Headers (Client Sends)

```
HX-Request: true             → always present, detect HTMX request
HX-Trigger: button-id        → ID of triggering element
HX-Target: result            → ID of target element
HX-Current-URL: /page        → current URL
HX-Boosted: true             → request via hx-boost
```

## Server-Side Pattern (Go Example)

```go
func usersHandler(w http.ResponseWriter, r *http.Request) {
    users := db.FindAll()

    // Return full page for direct navigation, fragment for HTMX
    if r.Header.Get("HX-Request") == "true" {
        renderPartial(w, "partials/user-list.html", users)
    } else {
        renderPage(w, "pages/users.html", users)
    }
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    name := r.FormValue("name")
    user, err := db.Create(name)
    if err != nil {
        w.Header().Set("HX-Reswap", "outerHTML")
        w.WriteHeader(422)
        renderPartial(w, "partials/form-error.html", err.Error())
        return
    }
    // Return new row to append
    renderPartial(w, "partials/user-row.html", user)
}
```

## Loading States

```html
<!-- Show/hide spinner during request -->
<button hx-get="/slow" hx-indicator="#spinner">
  Load
</button>
<span id="spinner" class="htmx-indicator">Loading...</span>

<!-- CSS: .htmx-indicator is hidden by default, shown during request -->
```

## Out-of-Band Swaps (OOB)

```html
<!-- Server can update multiple elements in one response -->
<!-- Main response: replaces hx-target -->
<li>New item</li>

<!-- OOB: also update the count badge -->
<span id="count" hx-swap-oob="true">42 items</span>
```

## Key Rules

- The server ALWAYS returns HTML fragments, never JSON in HTMX handlers.
- Check `HX-Request: true` header to distinguish HTMX from full-page requests.
- Use `hx-boost="true"` on `<body>` to upgrade all links and forms for free.
- Use `hx-confirm` for destructive actions (delete, reset).
- Respond with HTTP 422 (Unprocessable Entity) for validation errors — HTMX will swap.
- Use OOB swaps to update multiple page regions from a single response.
- Prefer `delay:300ms` on search inputs to debounce requests.
- Use `hx-indicator` for loading states instead of manual JS.
