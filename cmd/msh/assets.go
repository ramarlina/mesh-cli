package main

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/context"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	assetAlt        string
	assetName       string
	assetVisibility string
	assetTags       []string
	assetExpires    string
)

var uploadCmd = &cobra.Command{
	Use:   "upload <path>",
	Short: "Upload an asset",
	Long:  "Upload a file to Mesh and receive an asset ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		// Check if file exists
		fileInfo, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if fileInfo.IsDir() {
			fmt.Fprintf(os.Stderr, "error: %s is a directory\n", path)
			os.Exit(1)
		}

		// Determine MIME type
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Use filename if no name specified
		name := assetName
		if name == "" {
			name = filepath.Base(path)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		// Step 1: Create asset and get presigned URL
		createReq := &client.CreateAssetRequest{
			Name:       name,
			MimeType:   mimeType,
			SizeBytes:  fileInfo.Size(),
			Alt:        assetAlt,
			Visibility: assetVisibility,
			Tags:       assetTags,
			Expires:    assetExpires,
		}

		createResp, err := c.CreateAsset(createReq)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		// Step 2: Upload file to S3
		if !flagQuiet && !flagJSON {
			out.Printf("Uploading %s...\n", name)
		}

		err = uploadFileToS3(path, createResp.UploadURL, mimeType)
		if err != nil {
			out.Error(fmt.Errorf("upload failed: %w", err))
			os.Exit(1)
		}

		// Step 3: Complete the asset
		asset, err := c.CompleteAsset(createResp.Asset.ID)
		if err != nil {
			out.Error(fmt.Errorf("failed to complete upload: %w", err))
			os.Exit(1)
		}

		context.Set(asset.ID, "asset")

		if flagJSON {
			out.Success(asset)
		} else if !flagQuiet {
			out.Printf("✓ Uploaded: %s\n", asset.ID)
			out.Printf("  URL: %s\n", asset.URL)
		}
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download <as_id|this> [-o path]",
	Short: "Download an asset",
	Long:  "Download an asset by ID to a local file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		asset, err := c.GetAsset(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = asset.Name
			if outputPath == "" {
				outputPath = id
			}
		}

		if !flagQuiet && !flagJSON {
			out.Printf("Downloading %s...\n", asset.Name)
		}

		err = downloadFileFromURL(asset.URL, outputPath)
		if err != nil {
			out.Error(fmt.Errorf("download failed: %w", err))
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{
				"status": "downloaded",
				"path":   outputPath,
				"id":     id,
			})
		} else if !flagQuiet {
			out.Printf("✓ Downloaded to: %s\n", outputPath)
		}
	},
}

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage assets",
	Long:  "View and manage your uploaded assets",
}

var assetLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List assets",
	Long:  "List your uploaded assets",
	Run: func(cmd *cobra.Command, args []string) {
		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		assets, cursor, err := c.ListAssets(flagLimit, flagBefore, flagAfter)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if len(assets) == 0 {
			if !flagQuiet {
				out.Println("No assets")
			}
			return
		}

		// Update context to first asset
		if len(assets) > 0 {
			context.Set(assets[0].ID, "asset")
		}

		if flagJSON {
			result := map[string]interface{}{
				"assets": assets,
				"cursor": cursor,
			}
			out.Success(result)
		} else {
			for _, asset := range assets {
				renderAsset(out, asset)
			}
			if cursor != "" && !flagQuiet {
				out.Printf("\nNext page: --after %s\n", cursor)
			}
		}
	},
}

var assetShowCmd = &cobra.Command{
	Use:   "show <as_id|this>",
	Short: "Show asset details",
	Long:  "Display detailed information about an asset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		asset, err := c.GetAsset(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		context.Set(asset.ID, "asset")

		if flagJSON {
			out.Success(asset)
		} else {
			renderAssetDetailed(out, asset)
		}
	},
}

var assetRmCmd = &cobra.Command{
	Use:   "rm <as_id|this>",
	Short: "Delete an asset",
	Long:  "Permanently delete an asset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		id, _, err := context.ResolveTarget(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Confirm deletion unless --yes is set
		if !flagYes {
			fmt.Printf("Delete asset %s? [y/N]: ", id)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled")
				return
			}
		}

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		err = c.DeleteAsset(id)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		if flagJSON {
			out.Success(map[string]string{"status": "deleted", "id": id})
		} else if !flagQuiet {
			out.Printf("✓ Deleted: %s\n", id)
		}
	},
}

var assetSetCmd = &cobra.Command{
	Use:   "set <as_id> [flags]",
	Short: "Update asset metadata",
	Long:  "Update asset properties like name, visibility, or tags",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		// cfg, _ := config.Load()
		c := getClient()
		out := getOutputPrinter()

		req := &client.UpdateAssetRequest{
			Name:       assetName,
			Alt:        assetAlt,
			Visibility: assetVisibility,
			Tags:       assetTags,
		}

		asset, err := c.UpdateAsset(id, req)
		if err != nil {
			out.Error(err)
			os.Exit(1)
		}

		context.Set(asset.ID, "asset")

		if flagJSON {
			out.Success(asset)
		} else if !flagQuiet {
			out.Printf("✓ Updated: %s\n", asset.ID)
		}
	},
}

func uploadFileToS3(filePath, uploadURL, mimeType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func downloadFileFromURL(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func renderAsset(out *output.Printer, asset *client.Asset) {
	if out.IsRaw() {
		out.Printf("%s\n", asset.ID)
		return
	}

	sizeKB := float64(asset.SizeBytes) / 1024.0
	out.Printf("%s • %s • %.1f KB • %s\n",
		asset.ID,
		asset.Name,
		sizeKB,
		asset.CreatedAt.Format("2006-01-02"),
	)
}

func renderAssetDetailed(out *output.Printer, asset *client.Asset) {
	if out.IsRaw() {
		out.Printf("%s\n", asset.URL)
		return
	}

	out.Printf("ID: %s\n", asset.ID)
	out.Printf("Name: %s\n", asset.Name)
	if asset.OriginalName != "" && asset.OriginalName != asset.Name {
		out.Printf("Original: %s\n", asset.OriginalName)
	}
	out.Printf("Type: %s\n", asset.MimeType)
	out.Printf("Size: %d bytes (%.2f KB)\n", asset.SizeBytes, float64(asset.SizeBytes)/1024.0)
	out.Printf("Visibility: %s\n", asset.Visibility)
	if asset.Alt != "" {
		out.Printf("Alt: %s\n", asset.Alt)
	}
	if len(asset.Tags) > 0 {
		out.Printf("Tags: %s\n", strings.Join(asset.Tags, ", "))
	}
	out.Printf("URL: %s\n", asset.URL)
	out.Printf("Created: %s\n", asset.CreatedAt.Format("2006-01-02 15:04:05"))
	if asset.ExpiresAt != nil {
		out.Printf("Expires: %s\n", asset.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(assetCmd)

	assetCmd.AddCommand(assetLsCmd)
	assetCmd.AddCommand(assetShowCmd)
	assetCmd.AddCommand(assetRmCmd)
	assetCmd.AddCommand(assetSetCmd)

	uploadCmd.Flags().StringVar(&assetAlt, "alt", "", "Alt text for accessibility")
	uploadCmd.Flags().StringVar(&assetName, "name", "", "Display name (defaults to filename)")
	uploadCmd.Flags().StringVar(&assetVisibility, "visibility", "", "Visibility (public|unlisted|followers|private)")
	uploadCmd.Flags().StringSliceVar(&assetTags, "tag", []string{}, "Add tag (can be repeated)")
	uploadCmd.Flags().StringVar(&assetExpires, "expires", "", "Expiration duration (e.g., 1h, 7d, 30d)")

	downloadCmd.Flags().StringP("output", "o", "", "Output file path")

	assetSetCmd.Flags().StringVar(&assetName, "name", "", "Display name")
	assetSetCmd.Flags().StringVar(&assetAlt, "alt", "", "Alt text")
	assetSetCmd.Flags().StringVar(&assetVisibility, "visibility", "", "Visibility")
	assetSetCmd.Flags().StringSliceVar(&assetTags, "tag", []string{}, "Tags")
}
