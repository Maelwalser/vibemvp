# Contracts: gRPC & Protobuf Skill Guide

## Overview

Protobuf syntax, service definitions, streaming RPCs, server reflection, grpcui, and buf toolchain for linting and breaking change detection.

## Protobuf Definitions (proto3)

```protobuf
// proto/user/v1/user.proto
syntax = "proto3";

package user.v1;

option go_package   = "github.com/myorg/myapp/gen/user/v1;userv1";
option java_package = "com.myorg.myapp.user.v1";
option java_multiple_files = true;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// Service definition
service UserService {
  // Unary RPC
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // Server streaming (client calls once, server streams responses)
  rpc ListUsers(ListUsersRequest) returns (stream User);

  // Client streaming (client streams, server responds once)
  rpc BatchCreateUsers(stream CreateUserRequest) returns (BatchCreateUsersResponse);

  // Bidirectional streaming
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

// Messages
message User {
  string id    = 1;
  string email = 2;
  string name  = 3;
  Role   role  = 4;
  google.protobuf.Timestamp created_at = 5;
}

message GetUserRequest {
  string id = 1;  // field numbers must never change once published
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32  page_size   = 1;
  string page_token  = 2;
  Role   role_filter = 3;
}

message CreateUserRequest {
  string email = 1;
  string name  = 2;
  Role   role  = 3;
}

message BatchCreateUsersResponse {
  repeated User users        = 1;
  int32         created_count = 2;
}

message ChatMessage {
  string user_id = 1;
  string content = 2;
  google.protobuf.Timestamp sent_at = 3;
}

// Enums
enum Role {
  ROLE_UNSPECIFIED = 0;  // proto3: zero value must be "unspecified"
  ROLE_USER        = 1;
  ROLE_ADMIN       = 2;
  ROLE_MODERATOR   = 3;
}

// Oneof for union types
message NotificationEvent {
  oneof event {
    UserCreatedEvent  user_created  = 1;
    OrderPlacedEvent  order_placed  = 2;
    PaymentFailedEvent payment_failed = 3;
  }
}

// Void return — use google.protobuf.Empty
service AdminService {
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
}

message DeleteUserRequest {
  string id = 1;
}
```

## buf Toolchain

### buf.yaml

```yaml
# buf.yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - DEFAULT
  except:
    - PACKAGE_VERSION_SUFFIX  # remove if using versioned packages
breaking:
  use:
    - FILE
```

### buf.gen.yaml

```yaml
# buf.gen.yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/myorg/myapp/gen
plugins:
  # Go: generate .pb.go files
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative

  # Go: generate gRPC service code
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative,require_unimplemented_servers=false

  # TypeScript: generate protobuf + connect-web stubs
  - remote: buf.build/bufbuild/es
    out: gen/ts
  - remote: buf.build/connectrpc/es
    out: gen/ts
```

```bash
# Generate code
buf generate

# Lint protobuf files
buf lint

# Detect breaking changes against main branch
buf breaking --against '.git#branch=main'

# Detect breaking changes against published module
buf breaking --against 'buf.build/myorg/myapp'

# Format proto files
buf format -w
```

## Server Reflection

### Go (google.golang.org/grpc/reflection)

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

func newGRPCServer() *grpc.Server {
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            authInterceptor,
            loggingInterceptor,
        ),
    )

    userv1.RegisterUserServiceServer(s, &UserServiceServer{})

    // Enable reflection (disable in production or behind auth)
    if os.Getenv("ENV") != "production" {
        reflection.Register(s)
    }

    return s
}
```

### Python (grpcio-reflection)

```python
from grpc_reflection.v1alpha import reflection
from grpc_reflection.v1alpha.reflection_pb2 import ServerReflectionRequest

server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
user_pb2_grpc.add_UserServiceServicer_to_server(UserServicer(), server)

# Add reflection
SERVICE_NAMES = (
    user_pb2.DESCRIPTOR.services_by_name['UserService'].full_name,
    reflection.SERVICE_NAME,
)
reflection.enable_server_reflection(SERVICE_NAMES, server)

server.add_insecure_port('[::]:50051')
server.start()
```

### Java (grpc-services)

```java
import io.grpc.protobuf.services.ProtoReflectionService;

Server server = ServerBuilder.forPort(50051)
    .addService(new UserServiceImpl())
    .addService(ProtoReflectionService.newInstance())
    .build();
```

## grpcui (Browser Testing)

```bash
# Install
go install github.com/fullstorydev/grpcui/cmd/grpcui@latest

# Connect to a server with reflection enabled
grpcui -plaintext localhost:50051

# Connect to a specific proto file (no reflection needed)
grpcui -plaintext -proto proto/user/v1/user.proto localhost:50051

# Run as sidecar in Docker Compose
# docker-compose.yml
# grpcui:
#   image: fullstorydev/grpcui:latest
#   command: ["-plaintext", "app:50051"]
#   ports: ["8080:8080"]
```

## Go gRPC Client Example

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    userv1 "github.com/myorg/myapp/gen/user/v1"
)

func newUserClient(addr string) (userv1.UserServiceClient, *grpc.ClientConn, error) {
    conn, err := grpc.NewClient(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithChainUnaryInterceptor(
            otelgrpc.UnaryClientInterceptor(),
        ),
    )
    if err != nil {
        return nil, nil, fmt.Errorf("dial gRPC: %w", err)
    }
    return userv1.NewUserServiceClient(conn), conn, nil
}

// Unary call
func getUser(ctx context.Context, client userv1.UserServiceClient, id string) (*userv1.User, error) {
    resp, err := client.GetUser(ctx, &userv1.GetUserRequest{Id: id})
    if err != nil {
        return nil, fmt.Errorf("GetUser: %w", err)
    }
    return resp.User, nil
}

// Server streaming
func listUsers(ctx context.Context, client userv1.UserServiceClient) error {
    stream, err := client.ListUsers(ctx, &userv1.ListUsersRequest{PageSize: 100})
    if err != nil {
        return err
    }
    for {
        user, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("stream.Recv: %w", err)
        }
        log.Printf("user: %s", user.Email)
    }
    return nil
}
```

## Key Rules

- Field numbers in proto messages are permanent — never reuse a deleted field number.
- Always define a zero value `ENUM_UNSPECIFIED = 0` for every enum — proto3 default.
- Use `google.protobuf.Timestamp` for all timestamps — never `string` or `int64`.
- Use `google.protobuf.Empty` for void responses — never define an empty message.
- Run `buf lint` and `buf breaking --against '.git#branch=main'` in CI on every PR.
- Disable reflection in production unless behind authentication — it exposes the full API surface.
- Commit generated code alongside proto files — never generate in CI and throw away.
- Use versioned package names (`user/v1`) from the start — changing them later is a breaking change.
