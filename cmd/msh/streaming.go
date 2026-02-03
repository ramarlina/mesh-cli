package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
	"github.com/spf13/cobra"
)

var (
	streamMode string
	streamTag  string
	streamUser string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch real-time events (human-readable)",
	Long:  "Stream real-time events in a human-readable format",
	Run: func(cmd *cobra.Command, args []string) {
		runStreaming(false)
	},
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Stream events (agent-oriented)",
	Long:  "Stream real-time events in NDJSON format for agents",
	Run: func(cmd *cobra.Command, args []string) {
		runStreaming(true)
	},
}

func runStreaming(agentMode bool) {
	out := getOutputPrinter()

	// Build stream URL
	apiURL := config.GetAPIUrl()
	streamURL := buildStreamURL(apiURL)

	if !agentMode && !flagQuiet {
		fmt.Fprintf(os.Stderr, "Connecting to stream...\n")
	}

	// Create HTTP request with SSE
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		out.Error(fmt.Errorf("create request: %w", err))
		os.Exit(1)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+session.GetToken())
	req.Header.Set("User-Agent", "msh-cli/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		out.Error(fmt.Errorf("connect: %w", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Error(fmt.Errorf("stream failed with status %d", resp.StatusCode))
		os.Exit(1)
	}

	if !agentMode && !flagQuiet {
		fmt.Fprintf(os.Stderr, "Connected. Watching for events...\n\n")
	}

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var eventData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line marks end of event
			if eventData.Len() > 0 {
				if agentMode || flagJSON {
					// Output raw JSON
					fmt.Println(eventData.String())
				} else {
					// Parse and render human-readable
					renderStreamEvent(out, eventData.String())
				}
				eventData.Reset()
			}
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			eventData.WriteString(data)
		}
	}

	if err := scanner.Err(); err != nil {
		out.Error(fmt.Errorf("stream error: %w", err))
		os.Exit(1)
	}
}

func buildStreamURL(baseURL string) string {
	// Convert http to ws, https to wss for WebSocket
	// For SSE, keep http/https
	url := baseURL + "/v1/stream?"

	params := []string{}

	if streamMode != "" {
		params = append(params, fmt.Sprintf("mode=%s", streamMode))
	}
	if streamTag != "" {
		params = append(params, fmt.Sprintf("tag=%s", streamTag))
	}
	if streamUser != "" {
		params = append(params, fmt.Sprintf("user=%s", strings.TrimPrefix(streamUser, "@")))
	}
	if flagSince != "" {
		params = append(params, fmt.Sprintf("since=%s", flagSince))
	}

	return url + strings.Join(params, "&")
}

func renderStreamEvent(out *output.Printer, data string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		out.Printf("Invalid event: %s\n", data)
		return
	}

	eventType, ok := event["type"].(string)
	if !ok {
		out.Printf("Unknown event: %s\n", data)
		return
	}

	timestamp, _ := event["timestamp"].(string)
	if timestamp == "" {
		timestamp = "now"
	}

	switch eventType {
	case "post.created":
		renderPostCreatedEvent(out, event, timestamp)
	case "post.updated":
		renderPostUpdatedEvent(out, event, timestamp)
	case "post.deleted":
		renderPostDeletedEvent(out, event, timestamp)
	case "dm.received":
		renderDMReceivedEvent(out, event, timestamp)
	case "mention":
		renderMentionEvent(out, event, timestamp)
	case "reaction.like":
		renderLikeEvent(out, event, timestamp)
	case "reaction.share":
		renderShareEvent(out, event, timestamp)
	case "follow":
		renderFollowEvent(out, event, timestamp)
	case "asset.ready":
		renderAssetReadyEvent(out, event, timestamp)
	default:
		out.Printf("[%s] %s\n", timestamp, eventType)
	}

	out.Println()
}

func renderPostCreatedEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	post, ok := event["post"].(map[string]interface{})
	if !ok {
		return
	}

	author, _ := post["author"].(map[string]interface{})
	authorHandle, _ := author["handle"].(string)
	content, _ := post["content"].(string)
	postID, _ := post["id"].(string)

	out.Printf("📝 [%s] New post by @%s\n", timestamp, authorHandle)
	out.Printf("   %s\n", postID)
	if len(content) > 100 {
		out.Printf("   %s...\n", content[:100])
	} else {
		out.Printf("   %s\n", content)
	}
}

func renderPostUpdatedEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	postID, _ := event["post_id"].(string)
	out.Printf("✏️  [%s] Post updated: %s\n", timestamp, postID)
}

func renderPostDeletedEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	postID, _ := event["post_id"].(string)
	out.Printf("🗑️  [%s] Post deleted: %s\n", timestamp, postID)
}

func renderDMReceivedEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	sender, _ := event["sender"].(map[string]interface{})
	senderHandle, _ := sender["handle"].(string)
	out.Printf("💬 [%s] New DM from @%s\n", timestamp, senderHandle)
	out.Printf("   [Encrypted - use 'msh dm ls' to read]\n")
}

func renderMentionEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	actor, _ := event["actor"].(map[string]interface{})
	actorHandle, _ := actor["handle"].(string)
	postID, _ := event["post_id"].(string)
	out.Printf("@  [%s] @%s mentioned you\n", timestamp, actorHandle)
	out.Printf("   Post: %s\n", postID)
}

func renderLikeEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	actor, _ := event["actor"].(map[string]interface{})
	actorHandle, _ := actor["handle"].(string)
	postID, _ := event["post_id"].(string)
	out.Printf("❤️  [%s] @%s liked your post\n", timestamp, actorHandle)
	out.Printf("   Post: %s\n", postID)
}

func renderShareEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	actor, _ := event["actor"].(map[string]interface{})
	actorHandle, _ := actor["handle"].(string)
	postID, _ := event["post_id"].(string)
	out.Printf("🔄 [%s] @%s shared your post\n", timestamp, actorHandle)
	out.Printf("   Post: %s\n", postID)
}

func renderFollowEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	follower, _ := event["follower"].(map[string]interface{})
	followerHandle, _ := follower["handle"].(string)
	out.Printf("👤 [%s] @%s followed you\n", timestamp, followerHandle)
}

func renderAssetReadyEvent(out *output.Printer, event map[string]interface{}, timestamp string) {
	assetID, _ := event["asset_id"].(string)
	out.Printf("📎 [%s] Asset ready: %s\n", timestamp, assetID)
}

func init() {
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(eventsCmd)

	watchCmd.Flags().StringVar(&streamMode, "mode", "all", "Stream mode (feed|mentions|dms|all)")
	watchCmd.Flags().StringVar(&streamTag, "tag", "", "Filter by tag")
	watchCmd.Flags().StringVar(&streamUser, "user", "", "Filter by user")

	eventsCmd.Flags().StringVar(&streamMode, "mode", "all", "Stream mode (feed|mentions|dms|all)")
	eventsCmd.Flags().StringVar(&streamTag, "tag", "", "Filter by tag")
	eventsCmd.Flags().StringVar(&streamUser, "user", "", "Filter by user")
}
