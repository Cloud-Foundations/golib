# route53
A package which implements a dns-01 ACME protocol responder using AWS Route 53.

It is recommended to use an instance role to access the Route 53 zone. The
following IAM policies are the minimum required to update the zone record set.
This is an example policy document statement for Terraform:

```
  statement {
    actions = [
      "route53:ChangeResourceRecordSets",
    ]

    resources = [
      "arn:aws:route53:::hostedzone/${var.zone_id}",
    ]
  }

  statement {
    actions = [
      "route53:GetChange",
    ]

    resources = [
      "*",
    ]
  }
```

`var.zone_id` should expand to the Route 53 Hosted Zone ID which contains the
FQDN for which the ACME challenge is being made.

Note how the `route53:GetChange` action requires access to _all_ resources, as
the change ID is dynamic.
