package store

//
// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"
//
// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
// 	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
// 	"github.com/aws/aws-sdk-go-v2/service/iam"
// )
//
// type RivuletEvent struct {
// 	Version    string    `json:"version"`
// 	ID         string    `json:"id"`
// 	DetailType string    `json:"detail-type"`
// 	Source     string    `json:"source"`
// 	Account    string    `json:"account"`
// 	Time       time.Time `json:"time"`
// 	Region     string    `json:"region"`
// 	Resources  []string  `json:"resources"`
// 	Detail     struct {
// 		Message string `json:"message"`
// 	} `json:"detail"`
// }
//
// type EventBridgeStore struct {
// 	input       chan string
// 	buffer      []string
// 	eventBridge *eventbridge.Client
// 	iam         *iam.Client
// 	Shutdown    chan struct{}
// }
//
// func NewEventBridgeStore() *EventBridgeStore {
//
// 	iamClient := iam.NewFromConfig(cfg)
// 	return &EventBridgeStore{
// 		eventBridge: ebClient,
// 		iam:         iamClient,
// 		buffer:      make([]string, 10),
// 	}
// }
//
// func (s *EventBridgeStore) Close() {
// 	close(s.input)
// }
//
// func (s *EventBridgeStore) Receive() {
// 	for {
// 		fmt.Println("receiving")
// 		data, ok := <-s.input
// 		if !ok {
// 			fmt.Println("shutting down")
// 			s.Shutdown <- struct{}{}
// 			fmt.Println("sent shutdown")
// 			s.Close()
// 			fmt.Println("Shutdown")
// 			return
// 		}
// 		err := s.Write(data)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 	}
// }
//
// func (s *EventBridgeStore) Write(data string) error {
// 	data = strings.Trim(data, "\n")
// 	putEventsInput := &eventbridge.PutEventsInput{
// 		Entries: []types.PutEventsRequestEntry{
// 			{
// 				Detail:     aws.String(fmt.Sprintf(`{"message": "%s"}`, data)),
// 				DetailType: aws.String("rivulet"),
// 				Source:     aws.String("rivulet"),
// 			},
// 		},
// 	}
// 	resp, err := s.Client.PutEvents(context.Background(), putEventsInput)
// 	if err != nil {
// 		return err
// 	}
// 	if resp.FailedEntryCount != 0 {
// 		for _, failedEntry := range resp.Entries {
// 			fmt.Println(failedEntry)
// 			fmt.Println(*failedEntry.ErrorMessage)
// 		}
//
// 		return fmt.Errorf("failed to put events")
// 	}
// 	fmt.Println("wrote")
// 	return err
// }
// func (s *EventBridgeStore) Read() string {
// 	var filtered []string
// 	for _, v := range s.buffer {
// 		if v != "" {
// 			filtered = append(filtered, v)
// 		}
// 	}
// 	output := strings.Join(filtered, "\n")
// 	s.buffer = s.buffer[:0]
// 	return output
// }
//
// func (s *EventBridgeStore) Register(c chan string) {
// 	s.input = c
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		data, err := io.ReadAll(r.Body)
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		defer r.Body.Close()
// 		var event RivuletEvent
// 		err = json.Unmarshal(data, &event)
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		a := strings.Split(event.Detail.Message, "###")
// 		msg := a[1]
// 		i, err := strconv.Atoi(a[0])
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		s.buffer[i-1] = msg
// 	})
// 	go func() {
// 		err := http.ListenAndServe(":8080", nil)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		<-s.Shutdown
// 		s.Close()
// 		fmt.Println("shutting down")
// 	}()
// 	time.Sleep(1 * time.Second)
// }
