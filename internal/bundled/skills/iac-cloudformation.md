# AWS CloudFormation Skill Guide

## Overview

CloudFormation provisions AWS infrastructure via JSON/YAML templates. Use Parameters, Resources, Conditions, Outputs, and intrinsic functions to build reusable, environment-aware stacks.

## Template Structure

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Description: API service stack

Parameters:
  Environment:
    Type: String
    AllowedValues: [dev, staging, prod]
    Default: dev
  DBPassword:
    Type: String
    NoEcho: true   # hidden in console/API output

Conditions:
  IsProd: !Equals [!Ref Environment, prod]
  IsNotProd: !Not [!Condition IsProd]

Resources:
  ApiInstance:
    Type: AWS::EC2::Instance
    Properties:
      InstanceType: !If [IsProd, t3.medium, t3.micro]
      ImageId: ami-0c02fb55956c7d316
      Tags:
        - Key: Environment
          Value: !Ref Environment
        - Key: Name
          Value: !Sub "api-${Environment}"

  DBInstance:
    Type: AWS::RDS::DBInstance
    DeletionPolicy: !If [IsProd, Retain, Delete]
    Properties:
      DBInstanceClass: !If [IsProd, db.t3.medium, db.t3.micro]
      Engine: postgres
      EngineVersion: "16"
      DBName: appdb
      MasterUsername: admin
      MasterUserPassword: !Ref DBPassword
      MultiAZ: !If [IsProd, true, false]
      StorageEncrypted: true
      BackupRetentionPeriod: !If [IsProd, 7, 1]

Outputs:
  DBEndpoint:
    Value: !GetAtt DBInstance.Endpoint.Address
    Export:
      Name: !Sub "${AWS::StackName}-DBEndpoint"
```

## Intrinsic Functions

```yaml
# !Ref — reference parameter or resource logical ID
InstanceType: !Ref InstanceTypeParam

# !GetAtt — get resource attribute
DBHost: !GetAtt DBInstance.Endpoint.Address

# !Sub — string substitution
BucketName: !Sub "${AWS::StackName}-${Environment}-assets"

# !If — conditional value
MultiAZ: !If [IsProd, true, false]

# !Select — pick from list
AZ: !Select [0, !GetAZs !Ref AWS::Region]

# !Join — join list
CIDR: !Join [".", ["10", "0", "0", "0/16"]]

# !FindInMap — lookup value from mapping
AMI: !FindInMap [RegionMap, !Ref AWS::Region, AMI]

# Mappings
Mappings:
  RegionMap:
    us-east-1:
      AMI: ami-0c02fb55956c7d316
    eu-west-1:
      AMI: ami-0d71ea30463e0ff49
```

## Dynamic References (Secrets Manager / SSM)

```yaml
Resources:
  DBInstance:
    Type: AWS::RDS::DBInstance
    Properties:
      MasterUserPassword: "{{resolve:secretsmanager:prod/db:SecretString:password}}"
      # SSM Parameter Store
      InstanceType: "{{resolve:ssm:/myapp/instance-type:1}}"
      # SSM SecureString
      ApiKey: "{{resolve:ssm-secure:/myapp/api-key:1}}"
```

## Nested Stacks

```yaml
Resources:
  VPCStack:
    Type: AWS::CloudFormation::Stack
    Properties:
      TemplateURL: https://s3.amazonaws.com/my-bucket/vpc.yaml
      Parameters:
        Environment: !Ref Environment
        CIDR: "10.0.0.0/16"

  DatabaseStack:
    Type: AWS::CloudFormation::Stack
    DependsOn: VPCStack
    Properties:
      TemplateURL: https://s3.amazonaws.com/my-bucket/database.yaml
      Parameters:
        VpcId: !GetAtt VPCStack.Outputs.VpcId
        SubnetIds: !GetAtt VPCStack.Outputs.PrivateSubnetIds
```

## Change Sets (safe review before apply)

```bash
# Create change set
aws cloudformation create-change-set \
  --stack-name my-stack \
  --template-body file://template.yaml \
  --parameters ParameterKey=Environment,ParameterValue=prod \
  --change-set-name my-update-$(date +%s)

# Review changes
aws cloudformation describe-change-set \
  --stack-name my-stack \
  --change-set-name my-update-xxx

# Execute after review
aws cloudformation execute-change-set \
  --stack-name my-stack \
  --change-set-name my-update-xxx
```

## Custom Resource (Lambda-backed)

```yaml
Resources:
  MyCustomResource:
    Type: AWS::CloudFormation::CustomResource
    Properties:
      ServiceToken: !GetAtt CustomResourceFunction.Arn
      Environment: !Ref Environment

  CustomResourceFunction:
    Type: AWS::Lambda::Function
    Properties:
      Handler: index.handler
      Runtime: python3.12
      Role: !GetAtt LambdaRole.Arn
      Code:
        ZipFile: |
          import cfnresponse
          import boto3

          def handler(event, context):
              try:
                  request_type = event['RequestType']
                  # Handle Create/Update/Delete
                  if request_type == 'Create':
                      # Custom logic here
                      data = {"Result": "created"}
                  cfnresponse.send(event, context, cfnresponse.SUCCESS, data)
              except Exception as e:
                  cfnresponse.send(event, context, cfnresponse.FAILED, {"Error": str(e)})
```

## CloudFormation CLI

```bash
# Deploy stack (create or update)
aws cloudformation deploy \
  --template-file template.yaml \
  --stack-name api-prod \
  --parameter-overrides Environment=prod DBPassword=secret \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
  --no-fail-on-empty-changeset

# Describe stack events (for debugging)
aws cloudformation describe-stack-events \
  --stack-name api-prod \
  --query "StackEvents[?ResourceStatus=='CREATE_FAILED']"
```

## Key Rules

- Use `DeletionPolicy: Retain` for stateful resources (RDS, S3) in production.
- Use `NoEcho: true` for all sensitive parameters — they won't appear in console or API.
- Use change sets before any update to production stacks — avoid surprise replacements.
- Dynamic references (`{{resolve:secretsmanager:...}}`) avoid storing secrets in parameter values.
- Conditions (`IsProd`) allow a single template to serve multiple environments.
- Avoid circular dependencies between resources — use `DependsOn` to explicit order if needed.
- Never delete a stack without checking `DeletionPolicy` on RDS/S3 — default is `Delete`.
