# Ruby + Hanami Skill Guide

## Project Layout

```
app/
├── actions/
│   └── users/
│       ├── index.rb
│       ├── show.rb
│       └── create.rb
├── entities/
│   └── user.rb
├── repositories/
│   └── user_repository.rb
└── views/
    └── users/
        └── index.rb
config/
├── app.rb
└── routes.rb
lib/
└── my_app/
    └── types.rb
```

## Router DSL

```ruby
# config/routes.rb
module MyApp
  class Routes < Hanami::Routes
    root to: "home#index"

    get  "/health",     to: "health#show"

    scope "api" do
      scope "v1" do
        get    "/users",     to: "users#index"
        get    "/users/:id", to: "users#show"
        post   "/users",     to: "users#create"
        patch  "/users/:id", to: "users#update"
        delete "/users/:id", to: "users#destroy"
      end
    end
  end
end
```

## Action Objects

```ruby
# app/actions/users/create.rb
module MyApp
  module Actions
    module Users
      class Create < MyApp::Action
        include Deps[
          repo: "repositories.user_repository"
        ]

        params do
          required(:user).hash do
            required(:name).filled(:string)
            required(:email).filled(:string, format?: URI::MailTo::EMAIL_REGEXP)
            optional(:role).filled(:string, included_in?: %w[admin member viewer])
          end
        end

        def handle(request, response)
          halt 422, serialize_errors(request.params.errors) unless request.params.valid?

          user = repo.create(request.params[:user])
          response.status  = 201
          response.body    = serialize(user)
        end

        private

        def serialize_errors(errors)
          { errors: errors.to_h }.to_json
        end

        def serialize(user)
          { data: user.to_h }.to_json
        end
      end
    end
  end
end
```

## Action with dry-types Validation

```ruby
# lib/my_app/types.rb
module MyApp
  module Types
    include Dry.Types()

    Email = String.constrained(format: URI::MailTo::EMAIL_REGEXP)
    Role  = String.enum("admin", "member", "viewer")
  end
end

# app/actions/users/update.rb
module MyApp
  module Actions
    module Users
      class Update < MyApp::Action
        include Deps[repo: "repositories.user_repository"]

        params do
          required(:id).filled(:integer)
          optional(:user).hash do
            optional(:name).filled(:string)
            optional(:email).filled(MyApp::Types::Email)
            optional(:role).filled(MyApp::Types::Role)
          end
        end

        def handle(request, response)
          halt 422, { errors: request.params.errors.to_h }.to_json unless request.params.valid?

          user = repo.find(request.params[:id])
          halt 404, { error: "Not found" }.to_json unless user

          updated = repo.update(user.id, request.params[:user] || {})
          response.body = { data: updated.to_h }.to_json
        end
      end
    end
  end
end
```

## Immutable Entities

```ruby
# app/entities/user.rb
module MyApp
  class User < Hanami::Entity
    # Hanami::Entity is a plain struct — attributes are frozen after init.
    # Define no methods that mutate state.

    def admin?
      role == "admin"
    end

    def display_name
      "#{name} <#{email}>"
    end
  end
end
```

## Repository Pattern

```ruby
# app/repositories/user_repository.rb
module MyApp
  class UserRepository < Hanami::Repository
    # Built-in: #find, #all, #create, #update, #delete

    def find_by_email(email)
      users.where(email: email.downcase).one
    end

    def active
      users.where(active: true).to_a
    end

    def paginate(page:, per: 25)
      users.offset((page - 1) * per).limit(per).to_a
    end

    def create_with_profile(user_attrs, profile_attrs)
      # compose multiple writes explicitly — no magic callbacks
      user    = create(user_attrs)
      profile = profiles.create(profile_attrs.merge(user_id: user.id))
      [user, profile]
    end
  end
end
```

## View Exposure

```ruby
# app/views/users/index.rb
module MyApp
  module Views
    module Users
      class Index < MyApp::View
        expose :users do |users:|
          users.map(&:to_h)
        end

        expose :meta do |page: 1, total:|
          { page: page, total: total }
        end
      end
    end
  end
end
```

## Dependency Injection

```ruby
# Hanami uses dry-system under the hood.
# Register components in config/app.rb:

module MyApp
  class App < Hanami::App
    config.logger.level = :info
  end
end

# Inject via Deps mixin in actions:
include Deps[
  repo:    "repositories.user_repository",
  mailer:  "mailers.user_mailer",
  logger:  "logger"
]
```

## Error Handling

- Use `halt status_code, body` in actions for early termination — never raise unhandled exceptions.
- Validate params with the built-in `params` block before touching repositories.
- Return consistent JSON: `{ data: ... }` on success, `{ error: "..." }` or `{ errors: {...} }` on failure.
- Repositories raise `Hanami::Repository::Error` on constraint violations — rescue explicitly.
