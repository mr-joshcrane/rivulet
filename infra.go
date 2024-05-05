package rivulet

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebTypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/glambda"
)

func (r EventBridgeReceiver) Receive(ctx context.Context) ([]Message, error) {
	var message Message
	err := json.Unmarshal(r.event.Detail, &message)
	if err != nil {
		return []Message{}, err
	}
	return []Message{message}, nil
}

func SetupEventBridgeReceiverInfrastructure(cfg aws.Config, p *Publisher) error {
	err := createLambdaFunction(cfg, p.name)
	if err != nil {
		return err
	}
	functionName := fmt.Sprintf("arn:aws:lambda:%s:265693559009:function:%s", cfg.Region, p.name)
	err = createEventBridgeRule(cfg, p.name, functionName)
	if err != nil {
		return err
	}
	timestamp := fmt.Sprint(time.Now().Unix())
	err = p.Transport.Publish(Message{
		Publisher: p.name,
		Order:     0,
		Content:   timestamp,
	})
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	c := dynamodb.NewFromConfig(cfg)
	result, err := c.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String("rivulet"),
		KeyConditionExpression: aws.String("Publisher = :publisher AND #Order = :order"),
		ExpressionAttributeNames: map[string]string{
			"#Order": "Order",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":publisher": &types.AttributeValueMemberS{Value: p.name},
			":order":     &types.AttributeValueMemberN{Value: "0"},
		},
	})
	if err != nil {
		return err
	}
	if len(result.Items) == 0 {
		return fmt.Errorf("no items found")
	}
	if len(result.Items) > 1 {
		return fmt.Errorf("multiple items found")
	}
	messages, err := ParseQueryResults(result)
	if err != nil {
		return err
	}
	if !cmp.Equal(messages, []string{timestamp}) {
		return fmt.Errorf("expected %s, got %s", timestamp, messages)
	}
	return nil
}

func createEventBridgeRule(cfg aws.Config, name string, targetARN string) error {
	// Create EventBridge Rule
	client := eventbridge.NewFromConfig(cfg)
	out, err := client.PutRule(context.Background(), &eventbridge.PutRuleInput{
		Name:         aws.String(name),
		EventPattern: aws.String(`{"source": ["rivulet"], "detail-type": ["rivulet"]}`),
		State:        ebTypes.RuleStateEnabled,
		RoleArn:      aws.String("arn:aws:iam::265693559009:role/testrole"),
	})
	if err != nil {
		return err
	}
	_, err = client.PutTargets(context.Background(), &eventbridge.PutTargetsInput{
		Rule: aws.String(name),
		Targets: []ebTypes.Target{
			{
				Arn: aws.String(targetARN),
				Id:  aws.String("1"),
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = json.Marshal(out)
	if err != nil {
		return err
	}
	return err
}

func createLambdaFunction(cfg aws.Config, name string) error {
	inlinePolicy := glambda.WithInlinePolicy(`{"Version": "2012-10-17","Statement":{"Effect": "Allow","Action": "dynamodb:*","Resource": "*"}}`)
	executionRole := glambda.WithExecutionRole("rivulet-lambda-role", inlinePolicy)
	resourcePolicy := glambda.WithResourcePolicy("events.amazonaws.com")
	l := glambda.NewLambda(name, "cmd/lambda/main.go", executionRole, resourcePolicy, glambda.WithAWSConfig(cfg))
	return l.Deploy()
}
