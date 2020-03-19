# awssecretsmanager
A package which implements a remote certificate+key store and a locking
mechanism to serialise ACME transactions using AWS Secrets Manager.

It is recommended to use an instance role to access the secret. The following
IAM policies are the minimum required to read and update the secret.
This is an example policy document statement for Terraform:

```
  statement {
    actions = [
      "secretsmanager:GetSecretValue",
      "secretsmanager:PutSecretValue",
      "secretsmanager:UpdateSecretVersionStage",
    ]

    resources = [
      "aws_secretsmanager_secret.keymaster_x509.arn",
    ]
  }
```

`aws_secretsmanager_secret.keymaster_x509.arn` should expand to the ARN for the
secret.
