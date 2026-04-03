# Ruby + RSpec Skill Guide

## Project Layout

```
spec/
├── spec_helper.rb
├── rails_helper.rb          # Rails projects only
├── factories/
│   └── users.rb
├── models/
│   └── user_spec.rb
├── requests/                # integration / API tests
│   └── users_spec.rb
├── services/
│   └── user_service_spec.rb
└── support/
    ├── shared_examples/
    │   └── paginatable.rb
    └── matchers/
        └── json_matchers.rb
```

## Basic Structure

```ruby
# spec/models/user_spec.rb
RSpec.describe User, type: :model do
  # subject is the described class instantiated by default
  subject(:user) { build(:user) }

  describe "validations" do
    it { is_expected.to validate_presence_of(:name) }
    it { is_expected.to validate_presence_of(:email) }
    it { is_expected.to validate_uniqueness_of(:email).case_insensitive }
    it { is_expected.to validate_inclusion_of(:role).in_array(%w[admin member viewer]) }
  end

  describe "associations" do
    it { is_expected.to belong_to(:organization) }
    it { is_expected.to have_many(:posts).dependent(:destroy) }
  end

  describe "#admin?" do
    context "when role is admin" do
      subject(:user) { build(:user, role: "admin") }
      it { is_expected.to be_admin }
    end

    context "when role is not admin" do
      subject(:user) { build(:user, role: "member") }
      it { is_expected.not_to be_admin }
    end
  end
end
```

## let and let!

```ruby
RSpec.describe OrderService do
  # let: lazy — evaluated on first use
  let(:user)    { create(:user) }
  let(:product) { create(:product, price: 100) }
  let(:service) { described_class.new(user: user) }

  # let!: eager — evaluated before each example (like before :each)
  let!(:existing_order) { create(:order, user: user) }

  it "creates a new order" do
    result = service.place(product_id: product.id, quantity: 2)
    expect(result).to be_a(Order)
    expect(result.total).to eq(200)
  end
end
```

## before / after Hooks

```ruby
RSpec.describe NotificationJob do
  before(:all)  { DatabaseCleaner.strategy = :truncation }
  after(:all)   { DatabaseCleaner.strategy = :transaction }

  before(:each) do
    allow(Mailer).to receive(:deliver)
    @job = described_class.new
  end

  after(:each) { ActionMailer::Base.deliveries.clear }

  it "sends a welcome email" do
    @job.perform(user_id: create(:user).id)
    expect(Mailer).to have_received(:deliver).once
  end
end
```

## Matchers

```ruby
# Equality
expect(result).to eq(42)
expect(result).to eql("exact string")
expect(result).to be(true)           # object identity

# Type
expect(result).to be_a(User)
expect(result).to be_an(Array)
expect(result).to be_kind_of(Enumerable)
expect(result).to be_instance_of(Hash)

# Collections
expect(list).to include("alice", "bob")
expect(list).to contain_exactly(:a, :b, :c)   # order-independent
expect(list).to match_array([:a, :b, :c])       # alias

# Attributes
expect(user).to have_attributes(name: "Alice", role: "admin")

# Predicates
expect(user).to be_active       # calls user.active?
expect(list).to be_empty

# Numeric
expect(value).to be > 0
expect(price).to be_within(0.01).of(expected_price)

# Strings / regex
expect(message).to match(/welcome/i)
expect(message).to start_with("Hello")
expect(message).to end_with("!")
```

## Mocking and Stubbing

```ruby
RSpec.describe PaymentService do
  let(:gateway) { instance_double("StripeGateway") }
  let(:service) { described_class.new(gateway: gateway) }

  describe "#charge" do
    context "when payment succeeds" do
      before do
        allow(gateway).to receive(:charge)
          .with(amount: 500, currency: "usd")
          .and_return({ id: "ch_123", status: "succeeded" })
      end

      it "returns a successful result" do
        result = service.charge(amount: 500)
        expect(result.success?).to be true
      end

      it "calls the gateway once" do
        service.charge(amount: 500)
        expect(gateway).to have_received(:charge).once
      end
    end

    context "when gateway raises" do
      before do
        allow(gateway).to receive(:charge).and_raise(StripeGateway::Error, "card declined")
      end

      it "raises PaymentService::ChargeError" do
        expect { service.charge(amount: 500) }.to raise_error(PaymentService::ChargeError, /card declined/)
      end
    end
  end
end
```

## Shared Examples

```ruby
# spec/support/shared_examples/paginatable.rb
RSpec.shared_examples "a paginatable resource" do |factory_name|
  let!(:items) { create_list(factory_name, 30) }

  it "returns the first page" do
    get "/api/v1/#{factory_name}s?page=1&per_page=10"
    expect(response).to have_http_status(:ok)
    expect(json_body[:data].length).to eq(10)
    expect(json_body[:meta][:total_count]).to eq(30)
  end
end

# Usage in request spec:
RSpec.describe "Users API" do
  include_examples "a paginatable resource", :user
end
```

## FactoryBot Integration

```ruby
# spec/factories/users.rb
FactoryBot.define do
  factory :user do
    name  { Faker::Name.name }
    email { Faker::Internet.unique.email }
    role  { "member" }
    active { true }

    trait :admin do
      role { "admin" }
    end

    trait :inactive do
      active { false }
    end

    factory :admin_user, traits: [:admin]
  end
end

# Usage
create(:user)                    # persisted
build(:user)                     # not persisted
build_stubbed(:user)             # stubbed (fastest)
create(:admin_user)
create(:user, :inactive, name: "Bob")
create_list(:user, 5, role: "viewer")
```

## Request Specs (API Integration)

```ruby
RSpec.describe "POST /api/v1/users", type: :request do
  let(:valid_params) { { user: { name: "Alice", email: "alice@example.com", role: "member" } } }
  let(:headers) { { "Authorization" => "Bearer #{token}", "Content-Type" => "application/json" } }

  context "with valid params" do
    it "creates a user and returns 201" do
      post "/api/v1/users", params: valid_params.to_json, headers: headers
      expect(response).to have_http_status(:created)
      expect(json_body[:data][:email]).to eq("alice@example.com")
    end
  end

  context "with invalid params" do
    it "returns 422 with errors" do
      post "/api/v1/users", params: { user: { name: "" } }.to_json, headers: headers
      expect(response).to have_http_status(:unprocessable_entity)
      expect(json_body[:errors]).to include("Name can't be blank")
    end
  end

  def json_body
    JSON.parse(response.body, symbolize_names: true)
  end
end
```

## Error Handling in Tests

- Use `expect { ... }.to raise_error(ExceptionClass)` — never rescue in examples.
- Prefer `allow` over `expect` for setup stubs; use `expect(...).to have_received` for assertions.
- Use `instance_double` and `class_double` to get verified doubles — they fail if the interface changes.
- Run `bundle exec rspec --format documentation` to catch unclear test descriptions.
