# Ruby + Rails / Sinatra Skill Guide

## Rails Project Layout

```
app/
├── controllers/
│   ├── application_controller.rb
│   └── api/
│       └── v1/
│           └── users_controller.rb
├── models/
│   └── user.rb
├── views/           # omit for API-only apps
└── concerns/
    └── paginatable.rb
config/
└── routes.rb
```

## Routes

```ruby
# config/routes.rb
Rails.application.routes.draw do
  namespace :api do
    namespace :v1 do
      resources :users, only: [:index, :show, :create, :update, :destroy]
      resources :posts do
        resources :comments, shallow: true
      end
    end
  end

  get "/health", to: "health#show"
end
```

## ApplicationController

```ruby
class ApplicationController < ActionController::API
  before_action :authenticate!
  rescue_from ActiveRecord::RecordNotFound, with: :not_found
  rescue_from ActiveRecord::RecordInvalid,  with: :unprocessable

  private

  def authenticate!
    token = request.headers["Authorization"]&.split(" ")&.last
    @current_user = User.find_by!(api_token: token)
  rescue ActiveRecord::RecordNotFound
    render json: { error: "Unauthorized" }, status: :unauthorized
  end

  def not_found(err)
    render json: { error: err.message }, status: :not_found
  end

  def unprocessable(err)
    render json: { errors: err.record.errors.full_messages }, status: :unprocessable_entity
  end
end
```

## Controller with JSON Responses

```ruby
module Api
  module V1
    class UsersController < ApplicationController
      before_action :set_user, only: [:show, :update, :destroy]

      def index
        users = User.active.page(params[:page]).per(params[:per_page] || 25)
        render json: { data: users, meta: pagination_meta(users) }, status: :ok
      end

      def show
        render json: @user, status: :ok
      end

      def create
        user = User.create!(user_params)
        render json: user, status: :created
      end

      def update
        @user.update!(user_params)
        render json: @user, status: :ok
      end

      def destroy
        @user.destroy!
        head :no_content
      end

      private

      def set_user
        @user = User.find(params[:id])
      end

      def user_params
        params.require(:user).permit(:name, :email, :role)
      end
    end
  end
end
```

## ActiveRecord Model

```ruby
class User < ApplicationRecord
  # Associations
  belongs_to :organization
  has_many   :posts, dependent: :destroy
  has_one    :profile, dependent: :destroy

  # Validations
  validates :name,  presence: true, length: { maximum: 100 }
  validates :email, presence: true, uniqueness: { case_sensitive: false },
                    format: { with: URI::MailTo::EMAIL_REGEXP }
  validates :role,  inclusion: { in: %w[admin member viewer] }

  # Callbacks
  before_save :normalize_email

  # Scopes
  scope :active,    -> { where(active: true) }
  scope :admins,    -> { where(role: "admin") }
  scope :recent,    -> { order(created_at: :desc) }
  scope :with_posts, -> { includes(:posts) }

  private

  def normalize_email
    self.email = email.downcase.strip
  end
end
```

## Concern for Shared Logic

```ruby
# app/concerns/paginatable.rb
module Paginatable
  extend ActiveSupport::Concern

  included do
    # hook into including class if needed
  end

  def pagination_meta(collection)
    {
      current_page: collection.current_page,
      total_pages:  collection.total_pages,
      total_count:  collection.total_count,
      per_page:     collection.limit_value
    }
  end
end
```

## Sinatra Application

```ruby
# app.rb
require "sinatra"
require "sinatra/json"
require "json"

set :show_exceptions, false

error 404 do
  json error: "Not found"
end

error 500 do
  json error: "Internal server error"
end

before do
  content_type :json
  if request.post? || request.put? || request.patch?
    body = request.body.read
    @payload = body.empty? ? {} : JSON.parse(body, symbolize_names: true)
  end
end

get "/health" do
  json status: "ok"
end

get "/users" do
  users = User.all
  json data: users
end

post "/users" do
  user = User.create!(@payload)
  status 201
  json user
end

get "/users/:id" do
  user = User.find(params[:id])
  json user
end
```

## Sinatra Modular App

```ruby
# config.ru
require_relative "app/users_app"
require_relative "app/posts_app"

map "/users" do
  run UsersApp
end

map "/posts" do
  run PostsApp
end

# app/users_app.rb
class UsersApp < Sinatra::Base
  get "/" do
    json data: User.all
  end

  post "/" do
    user = User.create!(JSON.parse(request.body.read, symbolize_names: true))
    status 201
    json user
  end

  get "/:id" do
    json User.find(params[:id])
  end
end
```

## Environment Variables

```ruby
# Always read from ENV, never hardcode
db_url  = ENV.fetch("DATABASE_URL") { raise "DATABASE_URL not set" }
secret  = ENV.fetch("SECRET_KEY_BASE") { raise "SECRET_KEY_BASE not set" }
```

## Error Handling

- Use `rescue_from` in ApplicationController for domain-level errors.
- Return consistent JSON envelopes: `{ error: "..." }` or `{ errors: [...] }`.
- Use `head :no_content` for 204 responses — never render nil.
- Never rescue `StandardError` globally without re-raising or logging.
