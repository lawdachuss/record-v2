# Quick Fix Checklist

## ✅ What I Fixed

1. **Database sync before recording** - Added `SyncDatabase()` method that runs before starting each recording
   - File: `github_actions/database_manager.go` - Added `SyncDatabase()` method
   - File: `github_actions/github_actions_mode.go` - Added sync step in `StartWorkflowLifecycle()`

## ⚠️ What You Need to Check

### 1. Finalize Mode Configuration (Fixes .ts upload issue)

**Check**: `conf/settings.json`

```bash
cat conf/settings.json | grep finalize
```

**Should be**:
```json
{
  "finalize_mode": "remux"
}
```

**If it's "none" or missing**, change it to "remux" or "transcode"

### 2. Filester API Key (Fixes "only Gofile" issue)

**Check**: GitHub repository secrets

```bash
gh secret list | grep FILESTER_API_KEY
```

**If missing**, add it:
```bash
gh secret set FILESTER_API_KEY
# Paste your API key from https://filester.me/account
```

### 3. Verify Both API Keys in Workflow

**Add this to your workflow** (`.github/workflows/continuous-runner.yml`):

```yaml
- name: Debug Upload Configuration
  run: |
    echo "=== Upload Configuration Check ==="
    if [ -n "$GOFILE_API_KEY" ]; then
      echo "✅ GOFILE_API_KEY: Configured (${#GOFILE_API_KEY} chars)"
    else
      echo "❌ GOFILE_API_KEY: NOT CONFIGURED"
    fi
    
    if [ -n "$FILESTER_API_KEY" ]; then
      echo "✅ FILESTER_API_KEY: Configured (${#FILESTER_API_KEY} chars)"
    else
      echo "❌ FILESTER_API_KEY: NOT CONFIGURED"
    fi
  env:
    GOFILE_API_KEY: ${{ secrets.GOFILE_API_KEY }}
    FILESTER_API_KEY: ${{ secrets.FILESTER_API_KEY }}
```

## 🧪 Testing

### Quick Test (5 minutes)

1. Start a 2-minute test recording
2. Cancel the workflow after recording finishes
3. Check the logs for:

**Good signs**:
```
✅ FFmpeg remux completed
✅ Completed recording moved to videos/completed/channel.mp4
✅ Starting parallel upload to Gofile and Filester
✅ Gofile upload succeeded: https://gofile.io/...
✅ Filester upload succeeded: https://filester.me/...
```

**Bad signs**:
```
❌ Completed recording moved to videos/completed/channel.ts
❌ Skipping Filester upload - API key not configured
❌ keeping original recording because finalization failed
```

## 📋 Summary

| Issue | Cause | Fix | Status |
|-------|-------|-----|--------|
| Uploading .ts files | finalize_mode = "none" | Set to "remux" | ⚠️ Check config |
| Only Gofile uploads | Missing FILESTER_API_KEY | Add GitHub secret | ⚠️ Check secrets |
| DB not synced | No sync before recording | Added SyncDatabase() | ✅ Fixed |

## 🚀 After Fixes

Your workflow should:
1. ✅ Sync database before starting recording
2. ✅ Record stream to .ts file
3. ✅ Remux .ts to .mp4 after recording
4. ✅ Move .mp4 to completed/ directory
5. ✅ Upload .mp4 to BOTH Gofile and Filester on shutdown
6. ✅ Update database with both URLs
7. ✅ Commit and push database

## 📞 Need Help?

If issues persist after checking these items, provide:
1. Contents of `conf/settings.json`
2. Output of `gh secret list`
3. Workflow logs from the "Upload completed recordings" step
4. List of files in `videos/completed/` directory
