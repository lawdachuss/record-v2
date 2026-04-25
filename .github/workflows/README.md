# GoondVR GitHub Actions Continuous Runner

This workflow enables 24/7 recording of Chaturbate and Stripchat streams using GitHub Actions.

## Quick Start

1. **Add channels to record** - Edit `.github/workflows/channels.txt`
2. **Configure secrets** (optional but recommended) - See below
3. **Run workflow** - Go to Actions tab → GoondVR Continuous Runner → Run workflow

## Configuration

### Required Files

- **channels.txt** - List of channels to record (one per line)
  ```
  # Format: site:username OR just username (defaults to chaturbate)
  chaturbate:alice
  stripchat:bob
  charlie
  ```

### GitHub Secrets (Optional but Recommended)

Go to **Settings → Secrets and variables → Actions** and add:

#### 1. Cloudflare Bypass (Highly Recommended)

**`CHATURBATE_COOKIES`** - Required to bypass Cloudflare protection

**How to get cookies:**

1. Open Chrome/Firefox and go to https://chaturbate.com
2. Open Developer Tools (F12)
3. Go to **Network** tab
4. Refresh the page
5. Click on any request to chaturbate.com
6. Find the **Cookie** header in Request Headers
7. Copy the entire cookie string

**Example cookie format:**
```
affid=12345; __cf_bm=abc123...; cf_clearance=xyz789...; csrftoken=token123
```

**Important Notes:**
- Cookies expire after ~24 hours - you'll need to update them regularly
- Without cookies, Cloudflare will block most requests
- The workflow will show warnings if cookies are not configured

#### 2. Upload Services (Optional)

**`GOFILE_API_KEY`** - For uploading to Gofile.io
- Get your API key from https://gofile.io/myProfile

**`FILESTER_API_KEY`** - For uploading to Filester
- Get your API key from https://filester.me/account

**Without upload keys:**
- Recordings are saved locally in the workflow
- Files are uploaded to GitHub Artifacts on failure
- You can download from Artifacts section (7-day retention)

#### 3. Notifications (Optional)

**`DISCORD_WEBHOOK_URL`** - Discord notifications
- Create webhook in Discord Server Settings → Integrations

**`NTFY_TOKEN`** - Ntfy.sh notifications (optional)
- Get token from https://ntfy.sh

## How It Works

### Workflow Lifecycle

1. **Validation** - Reads channels.txt and validates configuration
2. **Matrix Jobs** - Creates one job per channel (max 20 parallel)
3. **Recording** - Each job:
   - Checks if channel is online every 1 minute
   - Records when online (maximum quality up to 4K 60fps)
   - Saves to `./videos/` directory
   - Uploads to Gofile and Filester (if API keys configured)
   - Stores metadata in `database/` folder
4. **Auto-Restart** - After 5.5 hours, gracefully shuts down and triggers next run
5. **Continuous** - Runs 24/7 with automatic restarts

### Recording Quality

- **Always maximum quality** - Up to 4K 60fps
- **Automatic fallback** - 2160p60 → 1080p60 → 720p60 → highest available
- **No configuration needed** - Quality selection is automatic

### Database Structure

Uploaded video links are stored in JSON files:

```
database/
├── chaturbate/
│   ├── username1/
│   │   └── 2026-04-26.json
│   └── username2/
│       └── 2026-04-26.json
└── stripchat/
    └── username3/
        └── 2026-04-26.json
```

Each JSON file contains:
```json
[
  {
    "timestamp": "2026-04-26T14:30:00Z",
    "duration_seconds": 3600,
    "file_size_bytes": 2147483648,
    "quality": "2160p60",
    "gofile_url": "https://gofile.io/d/abc123",
    "filester_url": "https://filester.me/file/xyz789",
    "session_id": "run-20260426-143000-12345",
    "matrix_job": "1"
  }
]
```

## Troubleshooting

### Cloudflare Blocking

**Error:** `channel was blocked by Cloudflare`

**Solution:**
1. Add `CHATURBATE_COOKIES` secret (see above)
2. Update cookies every 24 hours
3. Use a browser extension to export cookies automatically

### Cache Errors

**Warning:** `Cache save failed` or `tar failed with exit code 2`

**Solution:** These warnings are harmless and can be ignored. The workflow uses `continue-on-error` to prevent failures.

### No Recordings

**Check:**
1. Is the channel online? The workflow only records when streams are live
2. Are cookies configured? Cloudflare may block without valid cookies
3. Check workflow logs for specific errors

### Upload Failures

**If uploads fail:**
- Recordings are saved to GitHub Artifacts
- Download from Actions → Workflow Run → Artifacts section
- Artifacts are kept for 7 days

## Advanced Configuration

### Cost-Saving Mode

To reduce GitHub Actions minutes usage, you can enable cost-saving mode in the workflow file:

```yaml
env:
  COST_SAVING: true  # Change from false to true
```

This will:
- Reduce polling interval to 10 minutes (instead of 1 minute)
- Limit concurrent recordings to 2 channels

### Custom Polling Interval

Edit the workflow file to change polling frequency:

```yaml
env:
  POLLING_INTERVAL: "5m"  # Check every 5 minutes instead of 1
```

## Limitations

- **GitHub Actions limits:**
  - 6 hours maximum per job (workflow restarts automatically)
  - 20 concurrent jobs maximum
  - 2000 minutes/month on free plan

- **Storage:**
  - Recordings are uploaded to external services (Gofile/Filester)
  - Local storage is temporary (cleared after upload)
  - GitHub Artifacts: 7-day retention, 500MB per artifact

## Support

For issues or questions:
1. Check workflow logs in Actions tab
2. Review this README
3. Open an issue on GitHub

## License

This project is open source. See LICENSE file for details.
