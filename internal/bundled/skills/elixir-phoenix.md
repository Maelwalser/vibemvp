# Elixir + Phoenix Skill Guide

## Project Setup

```bash
mix phx.new my_app --no-html --no-assets   # API-only
mix phx.new my_app                          # Full with LiveView
cd my_app
mix deps.get
mix ecto.create
mix ecto.migrate
mix phx.server
```

## Project Layout

```
lib/
├── my_app/
│   ├── accounts/
│   │   ├── user.ex              # Ecto schema
│   │   └── accounts.ex          # context (business logic)
│   └── repo.ex
├── my_app_web/
│   ├── controllers/
│   │   ├── user_controller.ex
│   │   └── fallback_controller.ex
│   ├── channels/
│   │   ├── user_socket.ex
│   │   └── room_channel.ex
│   ├── live/
│   │   └── user_live.ex
│   ├── router.ex
│   └── endpoint.ex
└── my_app_web.ex
```

## Router

```elixir
# lib/my_app_web/router.ex
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  pipeline :api do
    plug :accepts, ["json"]
    plug MyAppWeb.Plugs.AuthPlug
  end

  pipeline :browser do
    plug :accepts, ["html"]
    plug :fetch_session
    plug :fetch_live_flash
    plug :put_root_layout, html: {MyAppWeb.Layouts, :root}
    plug :protect_from_forgery
    plug :put_secure_browser_headers
  end

  scope "/api/v1", MyAppWeb do
    pipe_through :api

    resources "/users", UserController, except: [:new, :edit]
    resources "/posts",  PostController, only: [:index, :show, :create]
  end

  scope "/", MyAppWeb do
    pipe_through :browser
    live "/users", UserLive.Index, :index
    live "/users/:id", UserLive.Show, :show
  end

  get "/health", HealthController, :show
end
```

## Ecto Schema

```elixir
# lib/my_app/accounts/user.ex
defmodule MyApp.Accounts.User do
  use Ecto.Schema
  import Ecto.Changeset

  schema "users" do
    field :name,       :string
    field :email,      :string
    field :role,       :string, default: "member"
    field :is_active,  :boolean, default: true
    field :password,   :string, virtual: true
    field :password_hash, :string

    belongs_to :organization, MyApp.Organizations.Organization
    has_many   :posts, MyApp.Blog.Post

    timestamps()
  end

  @valid_roles ~w[admin member viewer]

  def changeset(user, attrs) do
    user
    |> cast(attrs, [:name, :email, :role, :is_active, :password])
    |> validate_required([:name, :email])
    |> validate_format(:email, ~r/^[^\s]+@[^\s]+$/, message: "must be a valid email")
    |> validate_inclusion(:role, @valid_roles)
    |> validate_length(:password, min: 8)
    |> unique_constraint(:email)
    |> put_password_hash()
  end

  defp put_password_hash(%{valid?: true, changes: %{password: pwd}} = changeset) do
    put_change(changeset, :password_hash, Bcrypt.hash_pwd_salt(pwd))
  end
  defp put_password_hash(changeset), do: changeset
end
```

## Context (Business Logic)

```elixir
# lib/my_app/accounts/accounts.ex
defmodule MyApp.Accounts do
  import Ecto.Query
  alias MyApp.Repo
  alias MyApp.Accounts.User

  def list_users(opts \\ []) do
    page     = Keyword.get(opts, :page, 1)
    per_page = Keyword.get(opts, :per_page, 25)
    offset   = (page - 1) * per_page

    Repo.all(from u in User, limit: ^per_page, offset: ^offset, order_by: [desc: u.inserted_at])
  end

  def get_user!(id), do: Repo.get!(User, id)

  def get_user(id), do: Repo.get(User, id)

  def create_user(attrs) do
    %User{}
    |> User.changeset(attrs)
    |> Repo.insert()
  end

  def update_user(%User{} = user, attrs) do
    user
    |> User.changeset(attrs)
    |> Repo.update()
  end

  def delete_user(%User{} = user) do
    Repo.delete(user)
  end
end
```

## Controller

```elixir
# lib/my_app_web/controllers/user_controller.ex
defmodule MyAppWeb.UserController do
  use MyAppWeb, :controller

  alias MyApp.Accounts
  alias MyApp.Accounts.User

  action_fallback MyAppWeb.FallbackController

  def index(conn, params) do
    page  = Map.get(params, "page", "1") |> String.to_integer()
    users = Accounts.list_users(page: page)
    render(conn, :index, users: users)
  end

  def show(conn, %{"id" => id}) do
    user = Accounts.get_user!(id)
    render(conn, :show, user: user)
  end

  def create(conn, %{"user" => user_params}) do
    with {:ok, %User{} = user} <- Accounts.create_user(user_params) do
      conn
      |> put_status(:created)
      |> put_resp_header("location", ~p"/api/v1/users/#{user}")
      |> render(:show, user: user)
    end
  end

  def update(conn, %{"id" => id, "user" => user_params}) do
    user = Accounts.get_user!(id)

    with {:ok, %User{} = updated} <- Accounts.update_user(user, user_params) do
      render(conn, :show, user: updated)
    end
  end

  def delete(conn, %{"id" => id}) do
    user = Accounts.get_user!(id)

    with {:ok, %User{}} <- Accounts.delete_user(user) do
      send_resp(conn, :no_content, "")
    end
  end
end
```

## Fallback Controller

```elixir
# lib/my_app_web/controllers/fallback_controller.ex
defmodule MyAppWeb.FallbackController do
  use MyAppWeb, :controller

  def call(conn, {:error, %Ecto.Changeset{} = changeset}) do
    conn
    |> put_status(:unprocessable_entity)
    |> put_view(json: MyAppWeb.ChangesetJSON)
    |> render(:error, changeset: changeset)
  end

  def call(conn, {:error, :not_found}) do
    conn
    |> put_status(:not_found)
    |> put_view(json: MyAppWeb.ErrorJSON)
    |> render(:"404")
  end

  def call(conn, {:error, :unauthorized}) do
    conn
    |> put_status(:unauthorized)
    |> put_view(json: MyAppWeb.ErrorJSON)
    |> render(:"401")
  end
end
```

## Channels (WebSocket)

```elixir
# lib/my_app_web/channels/room_channel.ex
defmodule MyAppWeb.RoomChannel do
  use Phoenix.Channel

  def join("room:" <> room_id, _payload, socket) do
    # Authorize and assign state to socket
    socket = assign(socket, :room_id, room_id)
    {:ok, socket}
  end

  def handle_in("new_message", %{"body" => body}, socket) do
    broadcast!(socket, "new_message", %{body: body, user_id: socket.assigns.user_id})
    {:noreply, socket}
  end

  def handle_in("ping", _payload, socket) do
    {:reply, {:ok, %{status: "pong"}}, socket}
  end
end
```

## LiveView

```elixir
# lib/my_app_web/live/user_live.ex
defmodule MyAppWeb.UserLive do
  use MyAppWeb, :live_view

  alias MyApp.Accounts

  def mount(_params, _session, socket) do
    users = Accounts.list_users()
    {:ok, assign(socket, users: users, loading: false)}
  end

  def render(assigns) do
    ~H"""
    <div>
      <h1>Users</h1>
      <%= for user <- @users do %>
        <p phx-click="select_user" phx-value-id={user.id}>
          <%= user.name %>
        </p>
      <% end %>
    </div>
    """
  end

  def handle_event("select_user", %{"id" => id}, socket) do
    user   = Accounts.get_user!(id)
    socket = assign(socket, selected_user: user)
    {:noreply, socket}
  end
end
```

## Repo CRUD Patterns

```elixir
# Insert
{:ok, user}  = Repo.insert(changeset)
{:error, cs} = Repo.insert(invalid_changeset)

# Update
{:ok, user}  = Repo.update(changeset)

# Delete
{:ok, user}  = Repo.delete(user)

# Query
Repo.get(User, id)           # returns nil if not found
Repo.get!(User, id)          # raises Ecto.NoResultsError
Repo.one(query)              # returns nil or single result
Repo.all(query)              # list
Repo.exists?(query)          # boolean

# Preloading
Repo.preload(user, [:posts, :organization])
```

## Environment Variables

```elixir
# config/runtime.exs
config :my_app, MyApp.Repo,
  url: System.fetch_env!("DATABASE_URL"),
  pool_size: String.to_integer(System.get_env("POOL_SIZE", "10"))

config :my_app_web, MyAppWeb.Endpoint,
  secret_key_base: System.fetch_env!("SECRET_KEY_BASE")
```

## Error Handling

- Use `with` chains for multi-step operations that can fail — avoid nested `case` blocks.
- Use `action_fallback` in controllers to centralize error response rendering.
- Changesets carry validation errors — return `{:error, changeset}` from contexts.
- Never call `Repo.get!` unless you want `Ecto.NoResultsError` to propagate; use `Repo.get` + explicit nil handling in the controller.
