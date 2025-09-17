package main

import (
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Manage embeddings",
	Long:  `Generate and manage embeddings for SRA metadata.`,
}