package rivulet

import (
	"context"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Reader struct {
	PublisherName  string
	CurrentMessage int
	Store          MessageStore
}

type MessageStore interface {
	All() []Message
}

func NewReader(publsherName string) *Reader {
	return &Reader{
		PublisherName:  publsherName,
		CurrentMessage: 0,
	}
}

func (r *Reader) NewMessages() []string {
	messages := r.Store.All()
	var newMessages []string
	for _, msg := range messages {
		if msg.Order > r.CurrentMessage {
			newMessages = append(newMessages, msg.Content)
		}
		r.CurrentMessage = msg.Order
	}
	return newMessages
}

func Read(ctx context.Context, publisherName string) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg)
	query := Query(publisherName)
	results, err := client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	return ParseQueryResults(results)
}

func Query(publisherName string) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName:              aws.String("rivulet"),
		KeyConditionExpression: aws.String("Publisher = :publisher"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":publisher": &types.AttributeValueMemberS{Value: publisherName},
		},
	}
}

func ParseQueryResults(query *dynamodb.QueryOutput) ([]string, error) {
	var messages []Message
	for _, item := range query.Items {
		order, err := strconv.ParseInt(item["Order"].(*types.AttributeValueMemberN).Value, 10, 64)
		if err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			Publisher: item["Publisher"].(*types.AttributeValueMemberS).Value,
			Order:     int(order),
			Content:   item["Content"].(*types.AttributeValueMemberS).Value,
		})
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Order < messages[j].Order
	})
	var contents []string
	for _, msg := range messages {
		contents = append(contents, msg.Content)
	}
	return contents, nil
}
