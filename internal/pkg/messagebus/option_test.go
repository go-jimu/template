package messagebus_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/messagebus"
)

func TestOptionValidate(t *testing.T) {
	valid := messagebus.Option{
		Brokers:       []string{"localhost:9092"},
		ClientID:      "go-jimu-template",
		ConsumerGroup: "go-jimu-template",
		Topics:        []string{"user.integration.v1.UserRegisteredV1"},
		FailurePolicy: "drop",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*messagebus.Option)
	}{
		{name: "brokers", mutate: func(option *messagebus.Option) { option.Brokers = nil }},
		{name: "client id", mutate: func(option *messagebus.Option) { option.ClientID = "" }},
		{name: "consumer group", mutate: func(option *messagebus.Option) { option.ConsumerGroup = "" }},
		{name: "topics", mutate: func(option *messagebus.Option) { option.Topics = nil }},
		{name: "failure policy", mutate: func(option *messagebus.Option) { option.FailurePolicy = "retry" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			option := valid
			test.mutate(&option)
			if err := option.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
