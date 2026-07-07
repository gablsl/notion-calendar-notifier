package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const NotionVersion = "2022-06-28"

type NotionQueryPayload struct {
	Filter map[string]interface{} `json:"filter"`
}

type NotionResponse struct {
	Results []NotionPage `json:"results"`
}

type NotionPage struct {
	ID         string         `json:"id"`
	Properties PageProperties `json:"properties"`
}

type PageProperties struct {
	Name     TitleProperty    `json:"Name"`
	Blocks   SelectProperty   `json:"Blocks"`
	Notified CheckboxProperty `json:"notified"`
}

type TitleProperty struct {
	Title []TextObject `json:"title"`
}

type TextObject struct {
	PlainText string `json:"plain_text"`
}

type SelectProperty struct {
	Select *SelectOptions `json:"select"`
}

type SelectOptions struct {
	Name string `json:"name"`
}

type CheckboxProperty struct {
	Checkbox bool `json:"checkbox"`
}

func main() {
	notionToken := os.Getenv("NOTION_TOKEN")
	databaseID := os.Getenv("NOTION_DATABASE_ID")
	ntfyTopic := os.Getenv("NTFY_TOPIC")

	if notionToken == "" || databaseID == "" || ntfyTopic == "" {
		fmt.Println("[DEBUG] Error: Missing environment variables.")
		return
	}

	location, _ := time.LoadLocation("America/Sao-Paulo")
	now := time.Now().In(location)

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, location)

	fmt.Printf("[DEBUG] Current time (São Paulo): %s\n", now.Format(time.RFC3339))
	fmt.Printf("[DEBUG] Searching all events for today between %s and %s\n", startOfDay, endOfDay)

	payload := NotionQueryPayload{
		Filter: map[string]interface{}{
			"and": []map[string]interface{}{
				{
					"property": "Due date",
					"date": map[string]interface{}{
						"on_or_after": startOfDay,
					},
				},
				{
					"property": "Due date",
					"date": map[string]interface{}{
						"on_or_before": endOfDay,
					},
				},
				{
					"property": "notified",
					"checkbox": map[string]interface{}{
						"equals": false,
					},
				},
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", databaseID)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+notionToken)
	req.Header.Set("Notion-Version", NotionVersion)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] HTTP request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Notion API HTTP Status: %d\n", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to read response body: %v\n", err)
		return
	}

	// Print incoming JSON from Notion to inspect structural issues
	fmt.Printf("[DEBUG] Raw Notion Response: %s\n", string(bodyBytes))

	var result NotionResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		fmt.Printf("[DEBUG] Failed to unmarshal JSON: %v\n", err)
		return
	}

	fmt.Printf("[DEBUG] Total items found matching filter: %d\n", len(result.Results))

	for i, page := range result.Results {
		eventName := "Untitled"
		if len(page.Properties.Name.Title) > 0 {
			eventName = page.Properties.Name.Title[0].PlainText
		}

		blockName := "General"
		if page.Properties.Blocks.Select != nil {
			blockName = page.Properties.Blocks.Select.Name
		}

		fmt.Printf("[DEBUG] Processing item [%d]: %s (%s) - ID: %s\n", i, eventName, blockName, page.ID)

		sendNtfyNotification(ntfyTopic, eventName, blockName)
		updateNotionStatus(notionToken, page.ID)
	}
}

func sendNtfyNotification(topic, eventName, blockName string) {
	url := fmt.Sprintf("https://ntfy.sh/%s", topic)
	message := fmt.Sprintf("[%s] %s is starting soon!", blockName, eventName)

	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(message))
	req.Header.Set("Title", "Calendar Alert")
	req.Header.Set("Priority", "high")
	req.Header.Set("Tags", "calendar")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] ntfy push request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[DEBUG] ntfy API HTTP Status: %d\n", resp.StatusCode)
}

func updateNotionStatus(token, pageID string) {
	url := fmt.Sprintf("https://api.notion.com/v1/pages/%s", pageID)

	updateData := map[string]interface{}{
		"properties": map[string]interface{}{
			"notified": map[string]interface{}{
				"checkbox": true,
			},
		},
	}
	jsonUpdate, _ := json.Marshal(updateData)

	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonUpdate))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", NotionVersion)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] Notion checkbox update failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[DEBUG] Notion checkbox update HTTP Status: %d\n", resp.StatusCode)
}
