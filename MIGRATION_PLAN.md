# Migration Plan: Extract Working Logic from vasud3v/record

## Key Differences Found

### 1. **Primary API Approach** (CRITICAL)
**Working Repo (`vasud3v/record`):**
- Uses `PostChaturbateAPI()` with POST request to `/get_edge_hls_url_ajax/`
- Generates CSRF token: `fmt.Sprintf("%032x", time.Now().UnixNano())`
- Falls back to FlareSolverr scraping only if Cloudflare blocks the API

**Current Repo (`goondvr-main`):**
- In FlareSolverr mode: Goes directly to scraping `initialRoomDossier` from HTML
- In normal mode: Uses POST API
- **Problem**: FlareSolverr mode skips the API entirely

### 2. **Scraping Functions**
**Working Repo has TWO scraping methods:**
1. `ScrapeChaturbateStream()` - Uses FlareSolverr for cookies, then regular request
2. `ScrapeChaturbateStreamWithFlareSolverr()` - Full FlareSolverr with JS execution

**Current Repo:**
- Only has `fetchStreamViaFlareSolverr()` which tries to parse `initialRoomDossier`
- **Problem**: Chaturbate may have changed page structure, `initialRoomDossier` not always present

### 3. **Retry Logic with Multiple Strategies**
**Working Repo:**
```go
for attempt := 1; attempt <= 5; attempt++ {
    if attempt <= 3 {
        // First 3 attempts: Use FlareSolverr with sessions
        hlsURL, status, scrapeErr = internal.ScrapeChaturbateStreamWithFlareSolverr(attemptCtx, username)
    } else {
        // Last 2 attempts: Try direct scraping
        hlsURL, status, scrapeErr = internal.ScrapeChaturbateStream(attemptCtx, username)
    }
    // Exponential backoff with jitter
}
```

**Current Repo:**
- No retry logic in FlareSolverr path
- Single attempt to parse `initialRoomDossier`

### 4. **HLS URL Extraction**
**Working Repo uses multiple regex patterns:**
```go
patterns := []string{
    `"hls_source":\s*"([^"]+)"`,
    `"hlsSource":\s*"([^"]+)"`,
    `https://[^"'\s]+\.m3u8[^"'\s]*`,
}
```
Plus unicode unescaping and cleanup

**Current Repo:**
- Only looks for `window.initialRoomDossier` JavaScript variable
- Parses as JSON
- **Problem**: Brittle, depends on specific page structure

### 5. **Cookie Management**
**Working Repo:**
- Extracts ALL cookies from FlareSolverr response
- Updates global config with cookies AND User-Agent
- Sanitizes cookie strings (removes newlines, control characters)

**Current Repo:**
- Passes `nil` cookies to FlareSolverr (intentionally for fresh session)
- Doesn't update global config with FlareSolverr cookies

### 6. **Quality Selection**
**Working Repo:**
```go
// Always select the highest available quality
variant = lo.MaxBy(allResolutions, func(a, b *VideoResolution) bool {
    return a.Width > b.Width
})
```

**Current Repo:**
- Tries exact resolution match first
- Falls back to lower quality if exact not found

### 7. **Stream Staleness Detection**
**Working Repo:**
```go
lastSegmentTime := time.Now()
const staleTimeout = 3 * time.Minute
// Check if we haven't received new segments for too long
if time.Since(lastSegmentTime) > staleTimeout {
    return internal.ErrChannelOffline
}
```

**Current Repo:**
- No staleness detection
- Could hang indefinitely on stale streams

## Files to Create/Modify

### New Files Needed:
1. `internal/chaturbate_req.go` - POST API function
2. `internal/chaturbate_scrape.go` - Scraping functions with regex

### Files to Modify:
1. `chaturbate/chaturbate.go` - Update `FetchStream()` logic
2. `internal/internal_req.go` - Add sanitization, random UA
3. `internal/flaresolverr.go` - Add response struct with cookies

## Implementation Steps

1. ✅ Create `internal/chaturbate_req.go` with `PostChaturbateAPI()`
2. ✅ Create `internal/chaturbate_scrape.go` with scraping functions
3. ✅ Update `chaturbate/chaturbate.go`:
   - Replace FlareSolverr-first logic with API-first approach
   - Add retry logic with multiple strategies
   - Add fallback to scraping on Cloudflare block
4. ✅ Update `internal/internal_req.go`:
   - Add random User-Agent selection
   - Add cookie sanitization
5. ✅ Update `chaturbate/chaturbate.go` playlist selection:
   - Always prefer highest quality
6. ✅ Add stream staleness detection to segment watchers

## Testing Plan

1. Test with known online channel
2. Test with offline channel
3. Test with private show
4. Test Cloudflare blocking scenario
5. Test multi-channel recording in GitHub Actions
