# Elixir + Plug / Bandit Skill Guide

## Project Setup

```bash
mix new my_plug_app --sup
cd my_plug_app

# mix.exs deps:
# {:plug, "~> 1.16"},
# {:bandit, "~> 1.5"},
# {:jason, "~> 1.4"}

mix deps.get
```

## Project Layout

```
lib/
├── my_plug_app/
│   ├── application.ex        # OTP Application + supervision tree
│   ├── router.ex             # Plug.Router
│   ├── plugs/
│   │   ├── auth_plug.ex
│   │   └── json_parser_plug.ex
│   └── handlers/
│       └── user_handler.ex
└── my_plug_app.ex
```

## Plug Behaviour

```elixir
# A Plug must implement init/1 and call/2.
# init/1 is called at compile-time (or app start) to transform options.
# call/2 receives %Plug.Conn{} and the initialized opts; returns a conn.

defmodule MyPlugApp.Plugs.AuthPlug do
  @behaviour Plug

  import Plug.Conn

  @impl true
  def init(opts), do: opts   # pass-through; transform opts here if needed

  @impl true
  def call(conn, _opts) do
    case get_req_header(conn, "authorization") do
      ["Bearer " <> token] ->
        case validate_token(token) do
          {:ok, user_id} ->
            assign(conn, :current_user_id, user_id)

          {:error, _reason} ->
            conn
            |> put_resp_content_type("application/json")
            |> send_resp(401, Jason.encode!(%{error: "Unauthorized"}))
            |> halt()
        end

      _ ->
        conn
        |> put_resp_content_type("application/json")
        |> send_resp(401, Jason.encode!(%{error: "Missing authorization header"}))
        |> halt()
    end
  end

  defp validate_token(token) do
    # Real implementation: verify JWT or DB lookup
    if token == System.get_env("API_TOKEN") do
      {:ok, 1}
    else
      {:error, :invalid_token}
    end
  end
end
```

## JSON Parser Plug

```elixir
defmodule MyPlugApp.Plugs.JsonParserPlug do
  @behaviour Plug

  import Plug.Conn

  @impl true
  def init(opts), do: opts

  @impl true
  def call(conn, _opts) do
    with ["application/json" <> _] <- get_req_header(conn, "content-type"),
         {:ok, body, conn}         <- Plug.Conn.read_body(conn),
         {:ok, parsed}             <- Jason.decode(body) do
      %{conn | body_params: parsed, params: Map.merge(conn.params, parsed)}
    else
      _ -> conn   # Not JSON — pass through unchanged
    end
  end
end
```

## Plug.Builder Pipeline

```elixir
defmodule MyPlugApp.Pipeline do
  use Plug.Builder

  # Plugs run in declaration order
  plug Plug.RequestId
  plug Plug.Logger, log: :info
  plug MyPlugApp.Plugs.JsonParserPlug
  plug :set_content_type
  plug MyPlugApp.Router

  defp set_content_type(conn, _opts) do
    put_resp_content_type(conn, "application/json")
  end
end
```

## Plug.Router

```elixir
defmodule MyPlugApp.Router do
  use Plug.Router

  plug :match
  plug MyPlugApp.Plugs.AuthPlug
  plug :dispatch

  get "/health" do
    send_resp(conn, 200, Jason.encode!(%{status: "ok"}))
  end

  get "/api/v1/users" do
    page  = conn.params["page"] || "1"
    users = MyPlugApp.Accounts.list_users(page: String.to_integer(page))
    send_resp(conn, 200, Jason.encode!(%{data: users}))
  end

  get "/api/v1/users/:id" do
    case MyPlugApp.Accounts.get_user(conn.params["id"]) do
      nil  -> send_resp(conn, 404, Jason.encode!(%{error: "Not found"}))
      user -> send_resp(conn, 200, Jason.encode!(%{data: user}))
    end
  end

  post "/api/v1/users" do
    case MyPlugApp.Accounts.create_user(conn.body_params) do
      {:ok, user}      -> send_resp(conn, 201, Jason.encode!(%{data: user}))
      {:error, errors} -> send_resp(conn, 422, Jason.encode!(%{errors: errors}))
    end
  end

  patch "/api/v1/users/:id" do
    case MyPlugApp.Accounts.get_user(conn.params["id"]) do
      nil ->
        send_resp(conn, 404, Jason.encode!(%{error: "Not found"}))

      user ->
        case MyPlugApp.Accounts.update_user(user, conn.body_params) do
          {:ok, updated}   -> send_resp(conn, 200, Jason.encode!(%{data: updated}))
          {:error, errors} -> send_resp(conn, 422, Jason.encode!(%{errors: errors}))
        end
    end
  end

  delete "/api/v1/users/:id" do
    case MyPlugApp.Accounts.get_user(conn.params["id"]) do
      nil  -> send_resp(conn, 404, Jason.encode!(%{error: "Not found"}))
      user ->
        MyPlugApp.Accounts.delete_user(user)
        send_resp(conn, 204, "")
    end
  end

  match _ do
    send_resp(conn, 404, Jason.encode!(%{error: "Not found"}))
  end
end
```

## Bandit Web Server in Supervision Tree

```elixir
# lib/my_plug_app/application.ex
defmodule MyPlugApp.Application do
  use Application

  @impl true
  def start(_type, _args) do
    port = String.to_integer(System.get_env("PORT", "4000"))

    children = [
      # Database repo, etc.
      MyPlugApp.Repo,

      # Bandit serves the Plug pipeline
      {Bandit, plug: MyPlugApp.Pipeline, port: port}
    ]

    opts = [strategy: :one_for_one, name: MyPlugApp.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
```

## Plug.Conn Manipulation Reference

```elixir
import Plug.Conn

# Reading
get_req_header(conn, "authorization")   # ["Bearer token"] | []
conn.params["id"]                        # query + path params merged
conn.body_params["name"]                 # parsed body (after read_body)
conn.assigns[:current_user_id]           # data set by previous plugs

# Writing headers
conn = put_resp_header(conn, "x-request-id", request_id)
conn = put_resp_content_type(conn, "application/json")
conn = delete_resp_header(conn, "x-powered-by")

# Sending responses
conn = send_resp(conn, 200, body)        # status + body string
conn = send_file(conn, 200, path)
conn = send_chunked(conn, 200)

# Halting pipeline
conn = halt(conn)    # stops further plugs from running

# Session
conn = fetch_session(conn)
conn = put_session(conn, :user_id, 42)
conn = delete_session(conn, :user_id)
conn = clear_session(conn)

# Assigns (pass data between plugs)
conn = assign(conn, :user, user)
```

## Route-Level Auth Guard

```elixir
defmodule MyPlugApp.AdminRouter do
  use Plug.Router

  plug :match
  plug MyPlugApp.Plugs.AuthPlug
  plug :require_admin
  plug :dispatch

  defp require_admin(conn, _opts) do
    if conn.assigns[:is_admin] do
      conn
    else
      conn
      |> put_resp_content_type("application/json")
      |> send_resp(403, Jason.encode!(%{error: "Forbidden"}))
      |> halt()
    end
  end

  get "/stats" do
    send_resp(conn, 200, Jason.encode!(%{users: 42}))
  end
end
```

## Environment Variables

```elixir
# config/runtime.exs
config :my_plug_app,
  port:       String.to_integer(System.fetch_env!("PORT")),
  api_token:  System.fetch_env!("API_TOKEN"),
  database_url: System.fetch_env!("DATABASE_URL")
```

## Error Handling

- Always `halt/1` after sending an error response — otherwise downstream plugs still run.
- `Plug.Router` catches unmatched routes with the `match _` catch-all; always include it.
- Prefer returning tagged tuples `{:ok, result}` / `{:error, reason}` from business logic; pattern match in route handlers.
- Never raise exceptions in plug `call/2` — catch and convert to HTTP responses with `send_resp` + `halt`.
- Use `Plug.Conn.read_body/2` once per request — the body stream is consumed after first read.
