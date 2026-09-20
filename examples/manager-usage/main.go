package main

import (
	"context"
	"log"
	"os"

	xai "github.com/subtletydev/xai-go"
	xaimv1 "github.com/subtletydev/xai-go/gen/go/xai/management_api/v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiKey := os.Getenv(xai.APIKeyEnvVar)
	if apiKey == "" {
		log.Fatalf("$%s must be set", xai.APIKeyEnvVar)
	}

	client, err := xai.NewManagementClient(xai.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	res, err := client.UI.GetSpendingLimits(ctx, &xaimv1.GetSpendingLimitsReq{})
	if err != nil {
		log.Fatal(err)
	}

	limits := res.GetSpendingLimits()
	log.Printf("effective hard spending limit: %d cents", limits.GetEffectiveHardSl().GetVal())
	log.Printf("soft spending limit:           %d cents", limits.GetSoftSl().GetVal())
	log.Printf("effective spending limit:      %d cents", limits.GetEffectiveSl().GetVal())
}
