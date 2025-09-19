package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nishad/srake/internal/embeddings"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage embedding models",
	Long:  `Download and manage ONNX models for generating embeddings.`,
	Example: `  srake models list
  srake models download Xenova/SapBERT-from-PubMedBERT-fulltext
  srake models test Xenova/SapBERT-from-PubMedBERT-fulltext "test text"`,
}

// Models list subcommand
var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed models",
	RunE:  runModelsList,
}

func runModelsList(cmd *cobra.Command, args []string) error {
	config := embeddings.DefaultEmbedderConfig()
	manager, err := embeddings.NewManager(config.ModelsDir)
	if err != nil {
		return fmt.Errorf("failed to create model manager: %v", err)
	}

	models, err := manager.ListModels()
	if err != nil {
		return fmt.Errorf("failed to list models: %v", err)
	}

	if len(models) == 0 {
		printInfo("No models installed")
		fmt.Println("\nAvailable models to download:")
		for _, modelID := range embeddings.ListAvailableModels() {
			fmt.Printf("  %s\n", modelID)
		}
		fmt.Printf("\nUse 'srake models download <model-id>' to download a model\n")
		return nil
	}

	printInfo("Installed Models")
	fmt.Println(colorize(colorGray, strings.Repeat("─", 80)))

	for _, model := range models {
		fmt.Printf("%s %s\n", colorize(colorBold, "Model:"), model.ID)
		fmt.Printf("  Path: %s\n", model.Path)
		fmt.Printf("  Active variant: %s\n", colorize(colorCyan, model.ActiveVariant))

		fmt.Printf("  Variants:\n")
		for _, variant := range model.Variants {
			status := "not downloaded"
			if variant.Downloaded {
				status = colorize(colorGreen, fmt.Sprintf("downloaded (%s)", embeddings.FormatSize(variant.Size)))
			}
			fmt.Printf("    - %s: %s\n", variant.Name, status)
		}
		fmt.Println()
	}

	return nil
}

// Models download subcommand
var modelsDownloadCmd = &cobra.Command{
	Use:   "download <model-id> [--variant <variant>]",
	Short: "Download a model",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsDownload,
}

var downloadVariant string

func init() {
	// Models download command flags
	modelsDownloadCmd.Flags().StringVar(&downloadVariant, "variant", "", "Model variant to download (quantized|fp16|full)")

	// Add subcommands to models
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsDownloadCmd)
	modelsCmd.AddCommand(modelsTestCmd)
}

func runModelsDownload(cmd *cobra.Command, args []string) error {
	modelID := args[0]

	config := embeddings.DefaultEmbedderConfig()
	manager, err := embeddings.NewManager(config.ModelsDir)
	if err != nil {
		return fmt.Errorf("failed to create model manager: %v", err)
	}

	// Create progress channel
	progress := make(chan embeddings.DownloadProgress, 100)
	done := make(chan bool)

	// Display progress
	go func() {
		for p := range progress {
			fmt.Printf("\r%s: %.1f%% (%.1f MB/s, ETA: %s)",
				p.File,
				p.Percentage,
				p.Speed,
				p.ETA.Round(time.Second))
		}
		done <- true
	}()

	printInfo("Downloading model %s...", modelID)

	downloader := embeddings.NewDownloader(manager, progress)
	err = downloader.DownloadModel(modelID, downloadVariant)

	close(progress)
	<-done
	fmt.Println() // New line after progress

	if err != nil {
		return fmt.Errorf("failed to download model: %v", err)
	}

	printSuccess("Model %s downloaded successfully", modelID)
	return nil
}

// Models test subcommand
var modelsTestCmd = &cobra.Command{
	Use:   "test <model-id> <text>",
	Short: "Test a model by generating an embedding",
	Args:  cobra.ExactArgs(2),
	RunE:  runModelsTest,
}

func runModelsTest(cmd *cobra.Command, args []string) error {
	modelID := args[0]
	text := args[1]

	config := embeddings.DefaultEmbedderConfig()
	embedder, err := embeddings.NewEmbedder(config)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	printInfo("Loading model %s...", modelID)
	if err := embedder.LoadModel(modelID); err != nil {
		return fmt.Errorf("failed to load model: %v", err)
	}

	printInfo("Generating embedding for: \"%s\"", text)
	embedding, err := embedder.EmbedText(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %v", err)
	}

	printSuccess("Embedding generated successfully")
	fmt.Printf("Dimension: %d\n", len(embedding))
	fmt.Printf("First 10 values: [")
	for i := 0; i < 10 && i < len(embedding); i++ {
		fmt.Printf("%.4f", embedding[i])
		if i < 9 && i < len(embedding)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Printf("...]\n")

	return nil
}
