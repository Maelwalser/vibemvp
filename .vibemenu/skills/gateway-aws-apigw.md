# AWS API Gateway Skill Guide

## Overview

AWS API Gateway manages REST APIs (v1, feature-rich), HTTP APIs (v2, low-latency, cheaper), and WebSocket APIs. Use HTTP API for new greenfield Lambda or HTTP integrations; use REST API when you need usage plans, API keys, or request/response transformations.

## REST API vs HTTP API

| Feature | REST API (v1) | HTTP API (v2) |
|---------|--------------|--------------|
| Latency | ~6ms overhead | ~1ms overhead |
| Cost | ~$3.50/million | ~$1.00/million |
| Usage plans / API keys | Yes | No (use Lambda auth) |
| Request/response transforms | Yes (mapping templates) | No |
| WebSocket | No | No (separate product) |
| JWT authorizer | Via Lambda | Built-in |
| VPC private integrations | Yes | Yes |

---

## HTTP API (v2) — SAM Template

```yaml
# template.yml (AWS SAM)
AWSTemplateFormatVersion: "2010-09-09"
Transform: AWS::Serverless-2016-10-31

Globals:
  Function:
    Runtime: nodejs20.x
    Timeout: 30
    MemorySize: 256
    Environment:
      Variables:
        TABLE_NAME: !Ref UsersTable

Resources:
  # HTTP API (v2)
  HttpApi:
    Type: AWS::Serverless::HttpApi
    Properties:
      StageName: v1
      Auth:
        DefaultAuthorizer: JWTAuthorizer
        Authorizers:
          JWTAuthorizer:
            JwtConfiguration:
              issuer: !Sub "https://cognito-idp.${AWS::Region}.amazonaws.com/${UserPool}"
              audience:
                - !Ref UserPoolClient
            IdentitySource: "$request.header.Authorization"
      CorsConfiguration:
        AllowOrigins:
          - "https://app.example.com"
        AllowMethods:
          - GET
          - POST
          - PUT
          - DELETE
          - OPTIONS
        AllowHeaders:
          - Authorization
          - Content-Type
        MaxAge: 600

  # Lambda function with HTTP API trigger
  GetUsersFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: dist/handlers/users.getUsers
      Events:
        GetUsers:
          Type: HttpApi
          Properties:
            ApiId: !Ref HttpApi
            Method: GET
            Path: /users
            Auth:
              Authorizer: JWTAuthorizer

  CreateUserFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: dist/handlers/users.createUser
      Events:
        CreateUser:
          Type: HttpApi
          Properties:
            ApiId: !Ref HttpApi
            Method: POST
            Path: /users

  # Public endpoint (no auth)
  HealthFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: dist/handlers/health.check
      Events:
        Health:
          Type: HttpApi
          Properties:
            ApiId: !Ref HttpApi
            Method: GET
            Path: /health
            Auth:
              Authorizer: NONE

  UsersTable:
    Type: AWS::DynamoDB::Table
    Properties:
      BillingMode: PAY_PER_REQUEST
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH

Outputs:
  ApiUrl:
    Value: !Sub "https://${HttpApi}.execute-api.${AWS::Region}.amazonaws.com/v1"
```

## Lambda Proxy Integration

The Lambda function receives a standardized event and must return a specific format.

```typescript
import { APIGatewayProxyEventV2, APIGatewayProxyResultV2 } from "aws-lambda";

export const handler = async (event: APIGatewayProxyEventV2): Promise<APIGatewayProxyResultV2> => {
  // Route info
  const { rawPath, rawQueryString, pathParameters, queryStringParameters } = event;

  // Request body
  const body = event.body
    ? event.isBase64Encoded
      ? Buffer.from(event.body, "base64").toString("utf-8")
      : event.body
    : null;

  // Auth context from JWT authorizer
  const userId = event.requestContext.authorizer?.jwt?.claims?.sub;

  try {
    const data = await processRequest(userId, body);
    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ data }),
    };
  } catch (err) {
    if (err instanceof NotFoundError) {
      return {
        statusCode: 404,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ error: { code: "not_found", message: err.message } }),
      };
    }
    console.error("Unhandled error:", err);
    return {
      statusCode: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ error: { code: "internal_error", message: "Internal server error" } }),
    };
  }
};
```

## REST API with Usage Plans and API Keys

```yaml
# REST API (v1) with usage plans
Resources:
  RestApi:
    Type: AWS::ApiGateway::RestApi
    Properties:
      Name: my-api
      EndpointConfiguration:
        Types: [REGIONAL]

  UsersResource:
    Type: AWS::ApiGateway::Resource
    Properties:
      RestApiId: !Ref RestApi
      ParentId: !GetAtt RestApi.RootResourceId
      PathPart: users

  GetUsersMethod:
    Type: AWS::ApiGateway::Method
    Properties:
      RestApiId: !Ref RestApi
      ResourceId: !Ref UsersResource
      HttpMethod: GET
      AuthorizationType: NONE
      ApiKeyRequired: true      # require API key
      Integration:
        Type: AWS_PROXY
        IntegrationHttpMethod: POST
        Uri: !Sub "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/${GetUsersFunction.Arn}/invocations"

  # API Key
  ApiKey:
    Type: AWS::ApiGateway::ApiKey
    Properties:
      Name: partner-api-key
      Enabled: true

  # Usage Plan (rate + quota)
  UsagePlan:
    Type: AWS::ApiGateway::UsagePlan
    Properties:
      UsagePlanName: standard-plan
      Throttle:
        RateLimit: 100      # requests per second
        BurstLimit: 200
      Quota:
        Limit: 100000
        Period: MONTH
      ApiStages:
        - ApiId: !Ref RestApi
          Stage: prod

  # Associate key with plan
  UsagePlanKey:
    Type: AWS::ApiGateway::UsagePlanKey
    Properties:
      KeyId: !Ref ApiKey
      KeyType: API_KEY
      UsagePlanId: !Ref UsagePlan
```

## Lambda Authorizer

```typescript
// Custom authorizer — validates tokens and returns IAM policy
import { APIGatewayAuthorizerResult, APIGatewayRequestAuthorizerEvent } from "aws-lambda";

export const handler = async (event: APIGatewayRequestAuthorizerEvent): Promise<APIGatewayAuthorizerResult> => {
  const token = event.headers?.Authorization?.replace("Bearer ", "");

  if (!token) {
    throw new Error("Unauthorized");  // returns 401
  }

  try {
    const payload = await verifyJWT(token);
    return {
      principalId: payload.sub,
      policyDocument: {
        Version: "2012-10-17",
        Statement: [{
          Action: "execute-api:Invoke",
          Effect: "Allow",
          Resource: event.methodArn,
        }],
      },
      context: {
        userId: payload.sub,
        role: payload.role,
      },
    };
  } catch {
    throw new Error("Unauthorized");
  }
};
```

```yaml
# Register authorizer in SAM
Authorizer:
  Type: AWS::ApiGateway::Authorizer
  Properties:
    Name: token-authorizer
    RestApiId: !Ref RestApi
    Type: REQUEST
    AuthorizerUri: !Sub "arn:aws:apigateway:${AWS::Region}:lambda:path/2015-03-31/functions/${AuthorizerFunction.Arn}/invocations"
    IdentitySource: method.request.header.Authorization
    AuthorizerResultTtlInSeconds: 300
```

## CORS Configuration

```yaml
# HTTP API CORS (SAM)
HttpApi:
  Type: AWS::Serverless::HttpApi
  Properties:
    CorsConfiguration:
      AllowOrigins:
        - "https://app.example.com"
        - "https://admin.example.com"
      AllowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
      AllowHeaders: ["Authorization", "Content-Type", "X-Request-ID"]
      ExposeHeaders: ["X-Request-ID"]
      AllowCredentials: true
      MaxAge: 600
```

For REST API, add method responses and integration responses for OPTIONS.

## Rules

- Use HTTP API (v2) for new projects — it is faster and cheaper than REST API (v1)
- Always return a structured error body from Lambda — API Gateway does not generate error bodies
- Set `AuthorizerResultTtlInSeconds: 0` during development to avoid caching stale auth decisions
- Keep Lambda handler code thin — business logic goes in service classes, not in the event handler
- Never hardcode region or account ID — use `${AWS::Region}` and `${AWS::AccountId}` in SAM/CloudFormation
- Set `MemorySize` intentionally — Lambda CPU scales proportionally with memory
- Use SAM or CDK for all API Gateway resources — never manage via console (drift risk)
