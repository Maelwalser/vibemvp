# Ruby + Roda Skill Guide

## Project Layout

```
app.rb              # Roda application class
config.ru           # Rack entry point
routes/
├── users.rb        # route files loaded into the tree
└── posts.rb
models/
└── user.rb
```

## Application Bootstrap

```ruby
# app.rb
require "roda"
require "json"

class App < Roda
  plugin :json                 # auto-sets Content-Type and serializes return values
  plugin :json_parser          # parses JSON request bodies into r.POST
  plugin :all_verbs            # enables r.put, r.patch, r.delete
  plugin :halt                 # r.halt for early termination
  plugin :error_handler        # centralized error handling
  plugin :not_found            # 404 handler

  error do |e|
    response.status = 500
    { error: e.message }
  end

  not_found do
    { error: "Not found" }
  end

  route do |r|
    # Mount sub-trees
    r.on "api" do
      r.on "v1" do
        r.on "users",  &method(:users_routes)
        r.on "posts",  &method(:posts_routes)
      end
    end

    r.get "health" do
      { status: "ok" }
    end
  end
end

# config.ru
require_relative "app"
run App.freeze.app
```

## Tree-Based Routing

```ruby
# Roda routing is a decision tree: each r.on consumes a path segment.
# First-match semantics — order matters.

route do |r|
  r.on "users" do
    # POST /users
    r.post do
      user = User.create(r.POST)
      response.status = 201
      user.to_h
    end

    r.on Integer do |id|
      user = User[id]
      r.halt(404, { error: "User not found" }.to_json) unless user

      # GET /users/:id
      r.get do
        user.to_h
      end

      # PATCH /users/:id
      r.patch do
        user.update(r.POST)
        user.to_h
      end

      # DELETE /users/:id
      r.delete do
        user.destroy
        response.status = 204
        ""
      end
    end

    # GET /users  (must come after r.on Integer to avoid conflict)
    r.get do
      User.all.map(&:to_h)
    end
  end
end
```

## r.is for Exact Matches

```ruby
r.on "settings" do
  r.is do
    # Matches exactly /settings (no trailing segments)
    r.get  { current_user.settings.to_h }
    r.post { current_user.update_settings(r.POST); { ok: true } }
  end

  r.is "notifications" do
    # Matches exactly /settings/notifications
    r.get  { current_user.notification_prefs.to_h }
    r.post { current_user.update_notification_prefs(r.POST); { ok: true } }
  end
end
```

## r.remaining_path for Nested Routing

```ruby
r.on "admin" do
  # Guard: only admins reach nested routes
  r.halt(403, { error: "Forbidden" }.to_json) unless current_user&.admin?

  # r.remaining_path holds the unconsumed portion of the path
  r.on "users" do
    r.get do
      { data: User.all.map(&:to_h), remaining: r.remaining_path }
    end
  end
end
```

## Plugin Architecture

```ruby
class App < Roda
  # Security
  plugin :csrf                    # CSRF tokens for form submissions
  plugin :content_security_policy do |csp|
    csp.default_src :none
    csp.script_src  :self
  end

  # Sessions
  plugin :sessions,
    secret:      ENV.fetch("SESSION_SECRET"),
    key:         "_app_session",
    expire_after: 86_400

  # Request/Response helpers
  plugin :symbol_status           # r.halt :not_found
  plugin :typecast_params         # params.pos_int("id")
  plugin :request_headers         # r.headers["X-Request-Id"]

  # Static files (development)
  plugin :public, root: "public"
end
```

## Request Body Parsing

```ruby
# With plugin :json_parser, JSON bodies are available via r.POST
r.post do
  data = r.POST   # Hash with string keys
  name  = data["name"]
  email = data["email"]

  r.halt(422, { error: "name required" }.to_json) if name.nil? || name.empty?

  user = User.create(name: name, email: email)
  response.status = 201
  user.to_h
end
```

## r.halt for Early Termination

```ruby
# r.halt immediately stops routing and returns the response.
# Accepts: status, body; or just status; or just body.

r.halt(401, { error: "Unauthorized" }.to_json)   # status + body
r.halt(204)                                        # status only
r.halt({ error: "Bad request" }.to_json)           # body only (200)
```

## Session Plugin

```ruby
class App < Roda
  plugin :sessions, secret: ENV.fetch("SESSION_SECRET")

  route do |r|
    r.post "login" do
      user = User.authenticate(r.POST["email"], r.POST["password"])
      r.halt(401, { error: "Invalid credentials" }.to_json) unless user

      session[:user_id] = user.id
      { ok: true }
    end

    r.post "logout" do
      session.clear
      { ok: true }
    end
  end
end
```

## Error Handling

- Use `plugin :error_handler` for centralized exception handling — keep route blocks free of rescue.
- Use `plugin :not_found` for consistent 404 responses.
- Prefer `r.halt` over raising exceptions in route logic.
- Return plain Hashes from route blocks — `plugin :json` serializes them automatically.
- Freeze the app with `App.freeze.app` in config.ru for thread safety and performance.
