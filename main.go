package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		fmt.Println("Missing environment variables.")
		return
	}

	now := time.Now()
	fiveMinAgo := now.Add(-5 * time.Minute).Format(time.RFC3339)
	twentyMinAhead := now.Add(20 * time.Minute).Format(time.RFC3339)

	payload := NotionQueryPayload{
		Filter: map[string]interface{}{
			"and": []map[string]interface{}{
				{
					"property": "Due date",
					"date": map[string]interface{}{
						"on_or_after": fiveMinAgo,
					},
				},
				{
					"property": "Due date",
					"date": map[string]interface{}{
						"on_or_before": twentyMinAhead,
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
		fmt.Printf("Failed to query Notion: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result NotionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Failed to decode response: %v\n", err)
		return
	}

	for _, page := range result.Results {
		eventName := ""
		if len(page.Properties.Name.Title) > 0 {
			eventName = page.Properties.Name.Title[0].PlainText
		}

		blockName := "General"
		if page.Properties.Blocks.Select != nil {
			blockName = page.Properties.Blocks.Select.Name
		}

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
	_, _ = client.Do(req)
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
	_, _ = client.Do(req)
}
