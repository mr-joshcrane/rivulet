package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mr-joshcrane/rivulet"
	"github.com/mr-joshcrane/rivulet/store"
)

func handler(ctx context.Context, event events.EventBridgeEvent) error {
	store := store.NewDynamoDBStore()
	s := rivulet.NewEventBridgeSubscriber(event, store)
	err := s.Receive(ctx)
	if err != nil {
		return err
	}
	var msg rivulet.Message
	err = json.Unmarshal([]byte(event.Detail), &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message", err)
		return err
	}
	fmt.Println("Received message", msg)
	storedMessages, err := s.Store.Messages(msg.Publisher)
	if err != nil {
		fmt.Println("Error retrieving stored messages", err)
		return err
	}
	fmt.Println("Stored messages", storedMessages)
	return nil
}

func main() {
	lambda.Start(handler)
}
