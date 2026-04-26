# Supabase Integration Setup

This guide explains how to set up Supabase to store recording links and metadata in addition to the JSON file database.

## Overview

The system now supports **dual database storage**:
1. **JSON Files** (Primary) - Stored in the repository under `database/`
2. **Supabase** (Optional) - Cloud PostgreSQL database for better querying and API access

## Benefits of Supabase Integration

✅ **Better Querying** - SQL queries instead of parsing JSON files  
✅ **API Access** - REST API for external applications  
✅ **Real-time Updates** - Subscribe to new recordings  
✅ **Advanced Filtering** - Filter by date, channel, quality, size, etc.  
✅ **Scalability** - Handles large datasets efficiently  
✅ **Backup** - Additional redundancy for recording metadata  

---

## Setup Instructions

### 1. Create a Supabase Project

1. Go to [https://supabase.com](https://supabase.com)
2. Sign up or log in
3. Click "New Project"
4. Fill in project details:
   - **Name**: `goondvr-recordings` (or your preferred name)
   - **Database Password**: Choose a strong password
   - **Region**: Select closest to your location
5. Click "Create new project"
6. Wait for the project to be provisioned (~2 minutes)

### 2. Get Your Supabase Credentials

Once your project is ready:

1. Go to **Project Settings** (gear icon in sidebar)
2. Navigate to **API** section
3. Copy the following values:
   - **Project URL**: `https://xxxxx.supabase.co`
   - **anon public key**: `eyJhbGc...` (long string)
   - **service_role key**: `eyJhbGc...` (different long string)

**Important**: Use the **service_role key** for GitHub Actions (it has full access)

### 3. Create the Database Table

#### Option A: Using Supabase Dashboard (Recommended)

1. Go to **SQL Editor** in your Supabase dashboard
2. Click "New Query"
3. Copy the contents of `supabase/migrations/001_create_recordings_table.sql`
4. Paste into the SQL editor
5. Click "Run" or press `Ctrl+Enter`
6. Verify the table was created in the **Table Editor**

#### Option B: Using Supabase CLI

```bash
# Install Supabase CLI (if not already installed)
npm install -g supabase

# Login to Supabase
supabase login

# Link to your project
supabase link --project-ref your-project-ref

# Run the migration
supabase db push
```

### 4. Configure GitHub Actions Secrets

Add the following secrets to your GitHub repository:

1. Go to your repository on GitHub
2. Navigate to **Settings** > **Secrets and variables** > **Actions**
3. Click "New repository secret"
4. Add these secrets:

| Secret Name | Value | Description |
|-------------|-------|-------------|
| `SUPABASE_URL` | `https://xxxxx.supabase.co` | Your Supabase project URL |
| `SUPABASE_KEY` | `eyJhbGc...` | Your Supabase **service_role** key |

**Security Note**: Never commit these values to your repository!

### 5. Update Workflow Configuration

The workflow template (`github_actions/TEMPLATE.yml`) needs to be updated to pass Supabase credentials:

Add these environment variables to the "Start recording" step:

```yaml
- name: Start recording
  env:
    GOFILE_API_KEY: ${{ secrets.GOFILE_API_KEY }}
    FILESTER_API_KEY: ${{ secrets.FILESTER_API_KEY }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    DISCORD_WEBHOOK_URL: ${{ secrets.DISCORD_WEBHOOK_URL }}
    NTFY_TOKEN: ${{ secrets.NTFY_TOKEN }}
    SUPABASE_URL: ${{ secrets.SUPABASE_URL }}          # Add this
    SUPABASE_KEY: ${{ secrets.SUPABASE_KEY }}          # Add this
    # ... rest of environment variables
```

### 6. Verify Setup

After the workflow runs, verify that recordings are being stored:

1. Go to your Supabase dashboard
2. Navigate to **Table Editor**
3. Select the `recordings` table
4. You should see new rows appearing as recordings complete

---

## Database Schema

### Table: `recordings`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key (auto-generated) |
| `site` | TEXT | Streaming site (chaturbate, stripchat) |
| `channel` | TEXT | Channel username |
| `timestamp` | TIMESTAMPTZ | Recording start time (with timezone) |
| `date` | TEXT | Date in YYYY-MM-DD format |
| `duration_seconds` | INTEGER | Recording duration in seconds |
| `file_size_bytes` | BIGINT | File size in bytes |
| `quality` | TEXT | Quality string (e.g., "2160p60") |
| `gofile_url` | TEXT | Download URL from Gofile |
| `filester_url` | TEXT | Download URL from Filester |
| `filester_chunks` | TEXT[] | Array of chunk URLs (for files >10GB) |
| `session_id` | TEXT | Workflow session identifier |
| `matrix_job` | TEXT | Matrix job identifier |
| `created_at` | TIMESTAMPTZ | Record creation timestamp |

### Indexes

The following indexes are created for optimal query performance:

- `idx_recordings_site_channel` - For queries by site and channel
- `idx_recordings_date` - For queries by date
- `idx_recordings_timestamp` - For sorting by timestamp
- `idx_recordings_session_id` - For queries by workflow session
- `idx_recordings_site_channel_date` - Composite index for common queries

---

## Querying the Database

### Using Supabase Dashboard

1. Go to **Table Editor**
2. Select the `recordings` table
3. Use the filters and search to find recordings

### Using SQL Editor

```sql
-- Get all recordings for a specific channel
SELECT * FROM recordings 
WHERE site = 'chaturbate' AND channel = 'username1'
ORDER BY timestamp DESC;

-- Get recordings from a specific date
SELECT * FROM recordings 
WHERE date = '2024-01-15'
ORDER BY timestamp DESC;

-- Get recordings by quality
SELECT * FROM recordings 
WHERE quality = '2160p60'
ORDER BY timestamp DESC;

-- Get total storage used per channel
SELECT 
    site,
    channel,
    COUNT(*) as recording_count,
    SUM(file_size_bytes) as total_bytes,
    SUM(file_size_bytes) / 1024 / 1024 / 1024 as total_gb
FROM recordings
GROUP BY site, channel
ORDER BY total_bytes DESC;

-- Get recordings from a specific workflow session
SELECT * FROM recordings 
WHERE session_id = 'run-20240115-143000-abc'
ORDER BY timestamp DESC;
```

### Using REST API

You can query the database using Supabase's REST API:

```bash
# Get all recordings for a channel
curl "https://xxxxx.supabase.co/rest/v1/recordings?site=eq.chaturbate&channel=eq.username1&order=timestamp.desc" \
  -H "apikey: YOUR_ANON_KEY" \
  -H "Authorization: Bearer YOUR_ANON_KEY"

# Get recordings from a specific date
curl "https://xxxxx.supabase.co/rest/v1/recordings?date=eq.2024-01-15&order=timestamp.desc" \
  -H "apikey: YOUR_ANON_KEY" \
  -H "Authorization: Bearer YOUR_ANON_KEY"

# Get recordings by session
curl "https://xxxxx.supabase.co/rest/v1/recordings?session_id=eq.run-20240115-143000-abc" \
  -H "apikey: YOUR_ANON_KEY" \
  -H "Authorization: Bearer YOUR_ANON_KEY"
```

### Using JavaScript/TypeScript

```typescript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  'https://xxxxx.supabase.co',
  'YOUR_ANON_KEY'
)

// Get all recordings for a channel
const { data, error } = await supabase
  .from('recordings')
  .select('*')
  .eq('site', 'chaturbate')
  .eq('channel', 'username1')
  .order('timestamp', { ascending: false })

// Get recordings from a specific date
const { data, error } = await supabase
  .from('recordings')
  .select('*')
  .eq('date', '2024-01-15')
  .order('timestamp', { ascending: false })

// Subscribe to new recordings (real-time)
const subscription = supabase
  .channel('recordings')
  .on('postgres_changes', 
    { event: 'INSERT', schema: 'public', table: 'recordings' },
    (payload) => {
      console.log('New recording:', payload.new)
    }
  )
  .subscribe()
```

---

## Optional: Making Supabase Optional

If you want to make Supabase completely optional (not required):

1. Don't add the `SUPABASE_URL` and `SUPABASE_KEY` secrets
2. The system will automatically skip Supabase inserts
3. Only the JSON database will be used

The code checks if Supabase is configured before attempting to insert:

```go
if rch.supabaseManager != nil {
    // Insert to Supabase
} else {
    log.Printf("Supabase not configured, skipping")
}
```

---

## Troubleshooting

### Error: "Supabase API returned status 401"

**Cause**: Invalid or missing API key

**Solution**: 
- Verify you're using the **service_role** key (not anon key)
- Check that the secret is correctly set in GitHub Actions
- Ensure there are no extra spaces in the secret value

### Error: "relation 'recordings' does not exist"

**Cause**: Database table not created

**Solution**:
- Run the migration SQL in Supabase SQL Editor
- Verify the table exists in Table Editor

### Error: "Failed to add recording to Supabase"

**Cause**: Network issues or invalid data

**Solution**:
- Check the workflow logs for detailed error message
- Verify Supabase project is running (not paused)
- Check if you've exceeded free tier limits

### Recordings not appearing in Supabase

**Cause**: Supabase credentials not configured or invalid

**Solution**:
- Check workflow logs for Supabase-related errors
- Verify secrets are correctly set in GitHub Actions
- Test connection using the `TestConnection()` method

---

## Cost Considerations

### Supabase Free Tier

- **Database Size**: 500 MB
- **Bandwidth**: 5 GB/month
- **API Requests**: Unlimited

### Estimated Usage

Assuming each recording metadata entry is ~500 bytes:

- **1,000 recordings** = ~0.5 MB
- **10,000 recordings** = ~5 MB
- **100,000 recordings** = ~50 MB

The free tier should be sufficient for most use cases. If you exceed limits, consider:

1. **Upgrading to Pro** ($25/month) - 8 GB database, 250 GB bandwidth
2. **Archiving old records** - Delete records older than X months
3. **Using only JSON database** - Disable Supabase integration

---

## Security Best Practices

1. ✅ **Use service_role key** for GitHub Actions (full access needed)
2. ✅ **Use anon key** for public web applications (read-only)
3. ✅ **Enable RLS policies** to control access (already configured)
4. ✅ **Never commit secrets** to the repository
5. ✅ **Rotate keys periodically** in Supabase dashboard
6. ✅ **Monitor API usage** in Supabase dashboard

---

## Next Steps

After setup is complete:

1. ✅ Run a test workflow to verify recordings are stored
2. ✅ Query the database to confirm data is correct
3. ✅ Set up a web dashboard to display recordings (optional)
4. ✅ Configure real-time subscriptions for notifications (optional)
5. ✅ Set up automated backups (optional)

For questions or issues, check the [Supabase Documentation](https://supabase.com/docs) or open an issue in the repository.
