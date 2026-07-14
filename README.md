# Notion Daily Planner Notifier 🚀

A lightweight, automated daily assistant written in **Go** that scans your Notion workspace every morning and pushes a complete summary of your day's schedule directly to your devices using **ntfy.sh**. 

Powered entirely by **GitHub Actions**, this project runs on a serverless cron schedule with zero infrastructure overhead or hosting costs.

---

## 🛠️ Features
- **Daily Digest:** Runs automatically every morning at 08:00 AM (São Paulo time) to fetch all tasks scheduled for the current day.
- **State Management:** Automatically toggles a `notified` status checkbox in Notion to ensure you are only alerted once per task.
- **Instant Pushes:** Leverages `ntfy.sh` for high-priority, real-time mobile and desktop notifications.
- **Serverless Automation:** Utilizes GitHub Actions containers managed via environment variables and encrypted secrets.

---

## 📐 Project Architecture
The project is built around Go's core philosophy of structural simplicity. Everything runs linearly inside a highly verbosed `main.go` script:

1. **Trigger:** GitHub Actions wakes up the container via a cron schedule.
2. **Fetch:** Go queries the Notion API targeting a custom-scoped time window ($00:00:00$ to $23:59:59$) matching today's date.
3. **Dispatch:** For each pending event found, an HTTP POST request sends an alert payload containing context tags to your custom `ntfy` topic.
4. **Sync:** A persistent `PATCH` request updates the Notion page state, flipping the `notified` checkbox to `true`.

---

## 📋 Prerequisites & Configuration

### Notion Database Schema
Your target Notion database must contain at least the following explicit properties:
| Property Name | Type | Description |
| :--- | :--- | :--- |
| `Name` | Title | The main title/name of your scheduled task |
| `Due date` | Date | Scheduled date (including optional timestamps) |
| `notified` | Checkbox | State flag to prevent alert duplication |
| `Blocks` | Select | Custom category/area tag (e.g., *Work*, *Personal*, *Hobbies*) |

### GitHub Secrets
To safely run the automated routine without exposing your private credentials, add the following encrypted variables in your repository settings under **Settings -> Secrets and variables -> Actions**:

- `NOTION_TOKEN`: Your internal integration token generated via the Notion Developers Portal.
- `NOTION_DATABASE_ID`: The exact 32-character structural ID of your specific database.
- `NTFY_TOPIC`: Your private endpoint topic string on `ntfy.sh`.

---

## 🚀 How It Runs

```
GitHub Actions (Cron)
          │
          ▼
      Go Application
          │
          ├────────► Notion API
          │              │
          │              ▼
          │       Fetch today's tasks
          │
          ▼
     ntfy.sh API
          │
          ▼
 Phone/Desktop Notification
          │
          ▼
Update "notified" field in Notion
```

The automated lifecycle is controlled by `.github/workflows/cron.yml` using the environment configuration below:

```yaml
on:
  schedule:
    - cron: '0 11 * * *' # Fires daily at 11:00 UTC (08:00 AM BRT)
  workflow_dispatch:      # Allows manual trigger button from the Actions tab
