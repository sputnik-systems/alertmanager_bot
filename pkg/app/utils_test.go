package app

import (
	"testing"

	"github.com/prometheus/alertmanager/config"
	"github.com/stretchr/testify/assert"
)

func TestRemoveReceiverFromConfig(t *testing.T) {
	testCases := []struct {
		name       string
		cfg_before *config.Config
		cfg_after  *config.Config
		receiver   int64
	}{
		{
			name:       "empty",
			cfg_before: &config.Config{},
			cfg_after:  &config.Config{},
			receiver:   1,
		},
		{
			name: "1_user",
			cfg_before: &config.Config{
				Receivers: []*config.Receiver{
					{
						Name: "1",
					},
				},
			},
			cfg_after: &config.Config{
				Receivers: []*config.Receiver{},
			},
			receiver: 1,
		},
		{
			name: "2_user",
			cfg_before: &config.Config{
				Receivers: []*config.Receiver{
					{
						Name: "1",
					},
					{
						Name: "2",
					},
				},
			},
			cfg_after: &config.Config{
				Receivers: []*config.Receiver{
					{
						Name: "2",
					},
				},
			},
			receiver: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := removeReceiverFromConfig(tc.cfg_before, tc.receiver)
			if err != nil {
				t.Errorf("failed remove receiver: %s", err)
			}
			assert.Equal(t, tc.cfg_before, tc.cfg_after)
		})
	}
}
