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

func main() {
	source := envOrDefault("RIVULET_SOURCE", "rivulet")
	bus := envOrDefault("RIVULET_BUS", "default")
	name := envOrDefault("RIVULET_NAME", "publish")
	message := strings.Join(os.Args[1:], " ")

	if message == "" {
		fmt.Fprintln(os.Stderr, "usage: publish <message>")
		os.Exit(1)
	}

	runID := loadOrCreateRunID(name)

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
		name+" "+runID,
		client,
		rivulet.WithSource(source),
		rivulet.WithEventBusName(bus),
		rivulet.WithDetailType("notification"),
		rivulet.WithTransform(rivulet.DefaultTransform),
	)

	if err := publisher.Publish(message); err != nil {
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

func runIDPath(name string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("rivulet-%s.id", name))
}

func loadOrCreateRunID(name string) string {
	data, err := os.ReadFile(runIDPath(name))
	if err == nil && len(data) > 0 {
		return strings.TrimSpace(string(data))
	}
	id := uuid.New().String()
	os.WriteFile(runIDPath(name), []byte(id), 0644)
	return id
}
