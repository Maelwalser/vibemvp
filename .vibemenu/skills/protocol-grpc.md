# gRPC Skill Guide

## Overview

gRPC is an RPC framework using Protocol Buffers for serialization and HTTP/2 for transport. It provides strongly-typed service contracts, efficient binary encoding, and streaming support.

## Protobuf .proto Syntax

```protobuf
syntax = "proto3";

package myapp.v1;

option go_package = "github.com/org/service/gen/myapp/v1;myappv1";
option java_package = "com.org.myapp.v1";

import "google/protobuf/timestamp.proto";
import "google/rpc/status.proto";
import "google/protobuf/empty.proto";

// Service definition
service UserService {
  // Unary RPC
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // Server streaming — server sends multiple responses
  rpc ListUsers(ListUsersRequest) returns (stream User);

  // Client streaming — client sends multiple requests
  rpc BatchCreateUsers(stream CreateUserRequest) returns (BatchCreateResponse);

  // Bidirectional streaming
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

// Messages
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  google.protobuf.Timestamp created_at = 4;
  UserRole role = 5;
  repeated string tags = 6;
  map<string, string> metadata = 7;

  oneof contact {
    string phone = 8;
    string slack_handle = 9;
  }
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
  string filter = 3;       // e.g. "role=ADMIN"
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
  UserRole role = 3;
}

message BatchCreateResponse {
  repeated User users = 1;
  int32 created_count = 2;
  repeated string errors = 3;
}

message ChatMessage {
  string sender_id = 1;
  string content = 2;
  google.protobuf.Timestamp sent_at = 3;
}

enum UserRole {
  USER_ROLE_UNSPECIFIED = 0;  // proto3: always define 0 value
  USER_ROLE_VIEWER = 1;
  USER_ROLE_EDITOR = 2;
  USER_ROLE_ADMIN = 3;
}
```

## Unary RPC

### Go (google.golang.org/grpc)

```go
package server

import (
    "context"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    pb "github.com/org/service/gen/myapp/v1"
)

type UserServer struct {
    pb.UnimplementedUserServiceServer
    repo UserRepository
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }

    user, err := s.repo.FindByID(ctx, req.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, status.Errorf(codes.NotFound, "user %s not found", req.Id)
        }
        return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
    }

    return &pb.GetUserResponse{User: userToProto(user)}, nil
}
```

### TypeScript (@grpc/grpc-js)

```typescript
import * as grpc from "@grpc/grpc-js";
import { UserServiceHandlers } from "./gen/myapp/v1/UserService";

const handlers: UserServiceHandlers = {
  GetUser(call, callback) {
    const { id } = call.request;
    if (!id) {
      callback({ code: grpc.status.INVALID_ARGUMENT, message: "id is required" });
      return;
    }
    db.users.findById(id)
      .then(user => callback(null, { user: userToProto(user) }))
      .catch(err => callback({ code: grpc.status.INTERNAL, message: err.message }));
  },
};
```

### Python (grpcio)

```python
import grpc
from grpc import ServicerContext
import myapp_pb2
import myapp_pb2_grpc

class UserServicer(myapp_pb2_grpc.UserServiceServicer):
    def GetUser(self, request: myapp_pb2.GetUserRequest, context: ServicerContext):
        if not request.id:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "id is required")
        user = db.users.find_by_id(request.id)
        if not user:
            context.abort(grpc.StatusCode.NOT_FOUND, f"user {request.id} not found")
        return myapp_pb2.GetUserResponse(user=user_to_proto(user))
```

## Server Streaming

```go
func (s *UserServer) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
    users, err := s.repo.List(stream.Context(), req.Filter)
    if err != nil {
        return status.Errorf(codes.Internal, "list failed: %v", err)
    }

    for _, user := range users {
        if err := stream.Send(userToProto(user)); err != nil {
            return err // client disconnected
        }
    }
    return nil
}
```

## Client Streaming

```go
func (s *UserServer) BatchCreateUsers(stream pb.UserService_BatchCreateUsersServer) error {
    var users []*pb.User
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            // Client done sending, send final response
            return stream.SendAndClose(&pb.BatchCreateResponse{
                Users:        users,
                CreatedCount: int32(len(users)),
            })
        }
        if err != nil {
            return status.Errorf(codes.Internal, "recv error: %v", err)
        }

        user, err := s.repo.Create(stream.Context(), req)
        if err != nil {
            return status.Errorf(codes.Internal, "create failed: %v", err)
        }
        users = append(users, userToProto(user))
    }
}
```

## Bidirectional Streaming

```go
func (s *ChatServer) Chat(stream pb.UserService_ChatServer) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        // Broadcast or process message
        reply := &pb.ChatMessage{
            SenderId: "server",
            Content:  "Echo: " + msg.Content,
        }
        if err := stream.Send(reply); err != nil {
            return err
        }
    }
}
```

## Interceptors / Middleware

### Go (Unary + Streaming)

```go
import "google.golang.org/grpc"

// Unary interceptor
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
    return resp, err
}

// Auth interceptor
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    token := md["authorization"]
    if len(token) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing token")
    }
    // validate token...
    return handler(ctx, req)
}

// Register
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
    grpc.ChainStreamInterceptor(streamLoggingInterceptor),
)
```

## gRPC Status Codes

| Code | Name | HTTP Equiv | Use For |
|------|------|------------|---------|
| 0 | OK | 200 | Success |
| 1 | CANCELLED | 499 | Client cancelled |
| 2 | UNKNOWN | 500 | Unknown error |
| 3 | INVALID_ARGUMENT | 400 | Bad input |
| 4 | DEADLINE_EXCEEDED | 504 | Timeout |
| 5 | NOT_FOUND | 404 | Resource missing |
| 6 | ALREADY_EXISTS | 409 | Duplicate |
| 7 | PERMISSION_DENIED | 403 | Forbidden |
| 8 | RESOURCE_EXHAUSTED | 429 | Rate limited |
| 9 | FAILED_PRECONDITION | 400 | Wrong state |
| 13 | INTERNAL | 500 | Server error |
| 16 | UNAUTHENTICATED | 401 | Auth required |

```go
// Return rich error with details
st := status.New(codes.InvalidArgument, "validation failed")
st, _ = st.WithDetails(&errdetails.BadRequest{
    FieldViolations: []*errdetails.BadRequest_FieldViolation{
        {Field: "email", Description: "must be a valid email"},
    },
})
return nil, st.Err()
```

## Deadlines and Timeouts

```go
// Client side: set deadline
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: id})
if status.Code(err) == codes.DeadlineExceeded {
    log.Println("request timed out")
}

// Server side: check if context is still valid
func (s *Server) SlowOp(ctx context.Context, req *pb.Req) (*pb.Resp, error) {
    // Check before expensive operations
    select {
    case <-ctx.Done():
        return nil, status.Error(codes.DeadlineExceeded, "context cancelled")
    default:
    }
    // proceed with work...
}
```

## Reflection Service

```go
import "google.golang.org/grpc/reflection"

server := grpc.NewServer()
pb.RegisterUserServiceServer(server, &UserServer{})

// Enable reflection for grpcurl and other tools
reflection.Register(server)

// Now usable with:
// grpcurl -plaintext localhost:50051 list
// grpcurl -plaintext localhost:50051 myapp.v1.UserService/GetUser
```

## Server Setup

```go
func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
        grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
        grpc.KeepaliveParams(keepalive.ServerParameters{
            MaxConnectionIdle: 15 * time.Second,
            Time:              5 * time.Second,
            Timeout:           1 * time.Second,
        }),
    )

    pb.RegisterUserServiceServer(server, &UserServer{repo: repo})
    reflection.Register(server)

    log.Printf("gRPC server listening on :50051")
    if err := server.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

## Rules

- Always embed `Unimplemented*Server` to forward-compatible new methods
- Use `codes.InvalidArgument` for client mistakes, `codes.Internal` for server faults
- Always propagate context through all calls for cancellation and deadline support
- Define 0-value enum as `UNSPECIFIED` — proto3 default is 0
- Run `buf lint` and `buf breaking` in CI to catch schema regressions
- Use interceptor chains (`grpc.ChainUnaryInterceptor`) instead of nesting interceptors manually
- Enable reflection in development, disable or restrict in production
