# aws-identity-provider (cloudformation custom resource)

## WARNING: aws-identity-provider is deprecated in favor of [appspero/cfn-custom-provider](https://github.com/appspero/cfn-resource-provider)!

[![Build Status](https://travis-ci.org/appspero/nelly.svg?branch=master)](https://travis-ci.org/appspero/aws-identity-provider)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/appspero/aws-identity-provider)
[![Go Report Card](https://goreportcard.com/badge/github.com/appspero/aws-identity-provider)](https://goreportcard.com/report/github.com/appspero/aws-identity-provider)
[![Coverage](http://gocover.io/_badge/github.com/appspero/aws-identity-provider)](http://gocover.io/github.com/appspero/aws-identity-provider)

AWS is not supporting creating `OIDC/SAML` identity providers using CloudFormation. This custom resource will extend CloudFormation (using Go lambda function) to create identity providers. Further it supports automatic retrieving of root CA thumbprint for an OpenID connect identity provider.

## Installation

The custom resource package should be installed in S3 bucket to be used by CFN stacks. The package itself is a `Zip` file contains a compiled version of the Go lambda handler. It could be downloaded from [here](https://github.com/appspero/aws-identity-provider/releases).

To build the package run `make` then upload `aws-identity-provider.zip` to the S3 bucket:

To use the custom resource in other regions, the `Zip` should be uploaded in these regions. Further make sure to allow another accounts to read the file (as desired) using canonical user ID.

## Usage

The custom resource supports both types of identity provider `OIDC` and `SAML`. For case of `OIDC`, The `ThumbprintList` is optional; if it is not specified, the root CA of issuer server will be retrieved and used.

To use the custom resource, add the following:

```yaml
Resources:

  Provider:
    Type: Custom::IdentityProvider
    Properties:
      ServiceToken: !GetAtt ProviderCreator.Arn
      ProviderType: SAML # or OIDC
      IssuerURL: https://example.com/... # required if type is OIDC
      ClientIDList: # required if type is OIDC
        - clientID... 
      ThumbprintList: # optional if type is OIDC 
        - thumbprintList...
      SAMLProviderName: example # required if type is SAMl 
      SAMLMetadataDocument: "<?xml version=\"1.0\"..." # required if type is SAMl

  ProviderCreator:
    Type: AWS::Lambda::Function
    Properties:
      Runtime: go1.x
      Handler: aws-identity-provider
      MemorySize: 128
      Role: !GetAtt LambdaExecutionRole.Arn
      Timeout: 30
      Code:
        S3Bucket: example # s3 bucket contains lambda zip file
        S3Key: !Ref PackageS3Key # s3 bucket key of zip file
    DependsOn: LambdaExecutionRole

  LambdaExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      Path: /
      AssumeRolePolicyDocument:
        Version: 2012-10-17
        Statement:
          - Effect: Allow
            Principal:
              Service:
                - lambda.amazonaws.com
            Action:
              - sts:AssumeRole
      Policies:
        - PolicyName: root
          PolicyDocument:
            Version: 2012-10-17
            Statement:
              - Effect: Allow
                Action:
                  - iam:*OpenIDConnectProvider* # for OIDC
                  - iam:*SAMLProvider # for SAML
                Resource: "*"
              - Effect: Allow
                Action:
                  - logs:CreateLogGroup
                  - logs:CreateLogStream
                  - logs:PutLogEvents
                Resource: "*"

Outputs:
  ProviderArn:
    Value:
      Fn::GetAtt:
      - Provider
      - ProviderArn
  OpenIDConnectProviderUrl: # if type is OIDC
    Value:
      Ref: Provider
```

**Note:** to pass `SAMLMetadataDocument` parameter value as one line and escape the double-quote (") character, copy the contents of the file `out.xml` after running:

```bash
tr -d '\n' <metadata.xml | sed -e 's/"/\"/g' > out.xml
```
