package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBStore struct {
	client *dynamodb.Client
	table  string
}

func NewDynamoDBStore() *DynamoDBStore {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	return &DynamoDBStore{
		client: dynamodb.NewFromConfig(cfg),
		table:  "rivulet",
	}
}

func (s *DynamoDBStore) Save(m []Message) error {
	ctx := context.Background()
	command := &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item:      map[string]types.AttributeValue{},
	}
	for _, msg := range m {
		command.Item["Publisher"] = &types.AttributeValueMemberS{Value: msg.Publisher}
		command.Item["Order"] = &types.AttributeValueMemberN{Value: fmt.Sprint(msg.Order)}
		command.Item["Content"] = &types.AttributeValueMemberS{Value: msg.Content}
	}
	_, err := s.client.PutItem(ctx, command)
	if err != nil {
		return err
	}
	return nil
}

func (s *DynamoDBStore) Messages(publisher string) ([]Message, error) {
	command := dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("Publisher = :publisher"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":publisher": &types.AttributeValueMemberS{Value: publisher},
		},
	}
	results, err := s.client.Query(context.Background(), &command)
	if err != nil {
		return []Message{}, err
	}
	var messages []Message
	for _, item := range results.Items {
		order, err := strconv.ParseInt(item["Order"].(*types.AttributeValueMemberN).Value, 10, 64)
		if err != nil {
			panic(err)
		}
		messages = append(messages, Message{
			Publisher: item["Publisher"].(*types.AttributeValueMemberS).Value,
			Order:     int(order),
			Content:   item["Content"].(*types.AttributeValueMemberS).Value,
		})
	}
	return messages, nil
}
