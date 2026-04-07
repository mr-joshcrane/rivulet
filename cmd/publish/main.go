package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/google/uuid"
	"github.com/mr-joshcrane/rivulet"
)

type runState struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

func main() {
	source := envOrDefault("RIVULET_SOURCE", "rivulet")
	bus := envOrDefault("RIVULET_BUS", "default")
	name := envOrDefault("RIVULET_NAME", "publish")
	message := strings.Join(os.Args[1:], " ")

	if message == "" {
		fmt.Fprintln(os.Stderr, "usage: publish <message>")
		os.Exit(1)
	}

	state := loadOrCreateState(name)
	state.Order++
	saveState(name, state)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load AWS config: %v\n", err)
		os.Exit(1)
	}
	cfg.Region = "us-west-2"

	client := eventbridge.NewFromConfig(cfg)

	rivulet.DefaultTransform = func(m rivulet.Message) (string, error) {
		data, err := json.Marshal(m)
		if err != nil {
			return "", err
		}
		envelope := struct {
			Detail string `json:"detail"`
			Type   string `json:"type"`
		}{
			Type:   "rivulet",
			Detail: string(data),
		}
		out, err := json.Marshal(envelope)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	publisher := rivulet.NewEventBridgePublisher(
		name+" "+state.ID,
		client,
		rivulet.WithSource(source),
		rivulet.WithEventBusName(bus),
		rivulet.WithDetailType("notification"),
		rivulet.WithTransform(rivulet.DefaultTransform),
	)

	msg := rivulet.Message{
		Publisher: name + " " + state.ID,
		Order:     state.Order,
		Content:   message,
	}
	if err := publisher.Transport.Publish(msg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to publish: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func statePath(name string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("rivulet-%s.json", name))
}

func loadOrCreateState(name string) runState {
	data, err := os.ReadFile(statePath(name))
	if err != nil {
		return runState{ID: uuid.New().String(), Order: 0}
	}
	var state runState
	if json.Unmarshal(data, &state) != nil {
		return runState{ID: uuid.New().String(), Order: 0}
	}
	return state
}

func saveState(name string, state runState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	os.WriteFile(statePath(name), data, 0644)
}
