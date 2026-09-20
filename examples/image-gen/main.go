package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	xai "github.com/subtletydev/xai-go"
	xaiv1 "github.com/subtletydev/xai-go/gen/go/xai/api/v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiKey := os.Getenv(xai.APIKeyEnvVar)
	if apiKey == "" {
		log.Fatalf("$%s must be set", xai.APIKeyEnvVar)
	}

	client, err := xai.NewClient(xai.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	models, err := client.Models.ListImageGenerationModels(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("unable to list image generation models: %s", err)
	}
	for _, m := range models.GetModels() {
		log.Printf("image model: %s (aliases: %v)", m.GetName(), m.GetAliases())
	}

	response, err := client.Image.GenerateImage(ctx, &xaiv1.GenerateImageRequest{
		Prompt:      "generate image of an eagle sitting on a branch in 8bit stylistic",
		Model:       "grok-2-image",
		N:           proto.Int32(1),
		Format:      xaiv1.ImageFormat_IMG_FORMAT_URL,
		AspectRatio: xaiv1.ImageAspectRatio_IMG_ASPECT_RATIO_1_1.Enum(),
		Resolution:  xaiv1.ImageResolution_IMG_RESOLUTION_1K.Enum(),
	})
	if err != nil {
		log.Fatalf("unable to generate image: %s", err)
	}

	log.Printf("generated %d image(s) with %s", len(response.GetImages()), response.GetModel())
	for i, img := range response.GetImages() {
		if url := img.GetUrl(); url != "" {
			log.Printf("image %d: %s", i, url)
			continue
		}
		log.Printf("image %d: %d bytes of base64", i, len(img.GetBase64()))
	}
}
