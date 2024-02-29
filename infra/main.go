package main

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const AssumeRolePolicy = `{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Principal": {
			"Service": "events.amazonaws.com"
			},
			"Action": "sts:AssumeRole"
		}	
	]
}`

const InlinePolicy = `{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [ "events:InvokeApiDestination" ],
			"Resource": "arn:aws:events:*:*:api-destination/rivulet/*"
		}
	]
}`

const EventPattern = `{"source": ["rivulet"]}`

func main() {
	ctx := context.Background()
	stack, err := auto.SelectStackInlineSource(ctx, "dev", "dev", func(ctx *pulumi.Context) error {
		args := &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(AssumeRolePolicy),
			InlinePolicies: iam.RoleInlinePolicyArray{
				iam.RoleInlinePolicyArgs{
					Name:   pulumi.String("rivuletApiDestinationServicePolicy"),
					Policy: pulumi.String(InlinePolicy),
				},
			},
		}

		role, err := iam.NewRole(ctx, "rivuletApiDestinationServiceRole", args)
		if err != nil {
			return err
		}
		connection, err := cloudwatch.NewEventConnection(ctx, "rivuletApiDestinationServiceConnection", &cloudwatch.EventConnectionArgs{
			AuthParameters: cloudwatch.EventConnectionAuthParametersArgs{
				Basic: &cloudwatch.EventConnectionAuthParametersBasicArgs{
					Username: pulumi.String("admin"),
					Password: pulumi.String("admin"),
				},
			},
			AuthorizationType: pulumi.String("BASIC"),
			Description:       pulumi.String("Rivulet API Destination Service Connection"),
			Name:              pulumi.String("rivulet-api-destination-service-connection"),
		})
		if err != nil {
			return err
		}
		destination, err := cloudwatch.NewEventApiDestination(ctx, "rivuletApiDestinationService", &cloudwatch.EventApiDestinationArgs{
			ConnectionArn:      connection.Arn,
			HttpMethod:         pulumi.String("POST"),
			InvocationEndpoint: pulumi.String("https:"),
			Name:               pulumi.String("rivulet-api-destination-service"),
		})
		if err != nil {
			return err
		}

		rule, err := cloudwatch.NewEventRule(ctx, "rivuletApiDestinationServiceRule", &cloudwatch.EventRuleArgs{
			Description:  pulumi.String("Rivulet API Destination Service Rule"),
			EventPattern: pulumi.String(EventPattern),
			Name:         pulumi.String("rivulet-api-destination-service-rule"),
		})
		if err != nil {
			return err
		}
		_, err = cloudwatch.NewEventTarget(ctx, "rivuletApiDestinationServiceTarget", &cloudwatch.EventTargetArgs{
			Arn:     destination.Arn,
			RoleArn: role.Arn,
			Rule:    rule.Name,
		})
		if err != nil {
			return err
		}
		ctx.Export("roleArn", role.Arn)
		return nil
	})
	if err != nil {
		panic(err)
	}

	result, err := stack.Up(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result.Outputs)
	ex, err := stack.Export(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	data, err := ex.Deployment.MarshalJSON()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))

}
