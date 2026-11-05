package messagebus_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/messagebus"
)

func TestOptionValidate(t *testing.T) {
	valid := messagebus.Option{
		Brokers:        []string{"localhost:9092"},
		ClientID:       "go-jimu-template",
		ConsumerGroup:  "go-jimu-template",
		Topics:         []string{"user.integration.v1.UserRegisteredV1"},
		FailurePolicy:  "retry",
		HandlerTimeout: "5s",
		MaxPollRecords: 10,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	drop := valid
	drop.FailurePolicy = "drop"
	if err := drop.Validate(); err != nil {
		t.Fatalf("Validate() with failure-policy drop error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*messagebus.Option)
	}{
		{name: "brokers", mutate: func(option *messagebus.Option) { option.Brokers = nil }},
		{name: "client id", mutate: func(option *messagebus.Option) { option.ClientID = "" }},
		{name: "consumer group", mutate: func(option *messagebus.Option) { option.ConsumerGroup = "" }},
		{name: "topics", mutate: func(option *messagebus.Option) { option.Topics = nil }},
		// dlq was removed with the retry topics in contrib message/kafka v0.5.0.
		{name: "failure policy dlq", mutate: func(option *messagebus.Option) { option.FailurePolicy = "dlq" }},
		{name: "handler timeout missing", mutate: func(option *messagebus.Option) { option.HandlerTimeout = "" }},
		{name: "handler timeout invalid", mutate: func(option *messagebus.Option) { option.HandlerTimeout = "soon" }},
		{name: "handler timeout zero", mutate: func(option *messagebus.Option) { option.HandlerTimeout = "0s" }},
		{name: "max poll records zero", mutate: func(option *messagebus.Option) { option.MaxPollRecords = 0 }},
		// A partition's records run one after another, so a full batch of
		// slow records must still end within the 60s rebalance timeout.
		{name: "batch exceeds rebalance timeout", mutate: func(option *messagebus.Option) {
			option.HandlerTimeout = "6s"
			option.MaxPollRecords = 10
		}},
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
