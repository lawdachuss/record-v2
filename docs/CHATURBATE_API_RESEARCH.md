# Chaturbate API Research - April 2026

## Problem Statement

Live Chaturbate channels are showing as offline even though they're actually streaming. The `initialRoomDossier` JavaScript variable is not found in the HTML returned by FlareSolverr.

## Research Findings

### 1. Recent Changes (2025-2026)

**Evidence from yt-dlp Issue #12580 (February 2025):**
- yt-dlp users are experiencing the same issue
- Error: "Failed to download m3u8 information: HTTP Error 404: Not Found"
- The m3u8 stream URLs are returning 404 errors
- This suggests Chaturbate made significant changes to their streaming infrastructure

### 2. Current Status of `initialRoomDossier`

**From our logs:**
```
[DEBUG] kittengirlxo: Page title: Chaturbate - Free Adult Webcams, Live Sex, Free Sex Chat...
[DEBUG] kittengirlxo: Found 77 script tags in HTML
[DEBUG] kittengirlxo: Has room_page markers: false, Has chaturbate markers: true
[DEBUG] kittengirlxo: String 'initialRoomDossier' not found anywhere in HTML
```

**Analysis:**
- We're successfully getting the Chaturbate page (2.5MB HTML)
- The page has 77 script tags (JavaScript is present)
- But `initialRoomDossier` variable is completely missing
- This is NOT a Cloudflare issue (we're bypassing it successfully)

### 3. Possible Reasons for Missing Data

1. **Dynamic Loading**: Room data may now be loaded via AJAX after page load
2. **Variable Renamed**: Chaturbate may have renamed `initialRoomDossier` to something else
3. **Obfuscation**: The variable name may be obfuscated/minified
4. **API-Only Approach**: They may have moved to API-only data delivery
5. **Age Gate**: The data might only appear after accepting age verification

### 4. Alternative Methods

**Legacy API Endpoint:**
- URL: `https://chaturbate.com/api/chatvideocontext/{username}/`
- Status: Already implemented as fallback in our code
- May also be affected by recent changes

**Other Observed Patterns:**
- Some recorders use browser extensions (client-side only)
- Some use Selenium/Puppeteer to execute JavaScript
- Some scrape the affiliate XML feed (limited data)

## Current Implementation

### What We've Done

1. ✅ **Multiple Pattern Matching**: Try 5 different patterns for `initialRoomDossier`
2. ✅ **Enhanced Debugging**: Show page title, script count, HTML snippets
3. ✅ **Legacy API Fallback**: Automatically try API endpoint when HTML parsing fails
4. ✅ **FlareSolverr Integration**: Successfully bypass Cloudflare

### Code Flow

```
1. FlareSolverr fetches room page with real Chrome
2. Try to parse initialRoomDossier from HTML
3. If not found → Try legacy API endpoint
4. If API fails → Mark as offline
```

## Recommendations

### Short-term Solutions

1. **Wait for Legacy API Response** (CURRENT)
   - The fallback to legacy API is already implemented
   - Next workflow run will show if this works
   - Expected log: `[INFO] username: ✅ Legacy API fallback successful!`

2. **Check for Alternative JavaScript Variables**
   - Search for other variables that might contain stream data
   - Look for: `roomData`, `streamInfo`, `videoContext`, etc.
   - Add patterns to search for these alternatives

3. **Execute JavaScript in FlareSolverr**
   - FlareSolverr can execute JavaScript on the page
   - We could try to trigger any AJAX calls that load room data
   - Wait for dynamic content to load before extracting HTML

### Medium-term Solutions

1. **Reverse Engineer Current Method**
   - Inspect a live Chaturbate page in browser DevTools
   - Find how the player actually gets the stream URL
   - Look at Network tab for API calls
   - Replicate the exact method they use

2. **Use Selenium/Puppeteer**
   - Run a real browser that executes all JavaScript
   - Wait for stream player to initialize
   - Extract stream URL from video element or network requests
   - More resource-intensive but more reliable

3. **Monitor for Pattern Changes**
   - Set up alerts when parsing fails
   - Regularly check if Chaturbate changes their structure
   - Keep fallback methods updated

### Long-term Solutions

1. **Multi-Site Support**
   - Don't rely solely on Chaturbate
   - Stripchat support is already implemented
   - Add more platforms for redundancy

2. **Community Collaboration**
   - Monitor yt-dlp issues for solutions
   - Check other recorder projects for updates
   - Share findings with the community

## Next Steps

1. **Test Legacy API Fallback**
   - Wait for next workflow run
   - Check if `fetchStreamLegacy` successfully gets HLS URLs
   - If yes: Problem solved!
   - If no: Need to investigate further

2. **Manual Browser Inspection**
   - Open a live Chaturbate room in browser
   - Open DevTools → Network tab
   - Find the actual API call that gets stream URL
   - Replicate that exact call in our code

3. **Add More Debug Output**
   - Log all network requests made by FlareSolverr
   - Show any API calls visible in the HTML
   - Search for alternative variable names

## Technical Details

### Current Parsing Logic

```go
// Try multiple patterns
patterns := []string{
    "window.initialRoomDossier = \"",
    "initialRoomDossier = \"",
    "window.initialRoomDossier=\"",
    "initialRoomDossier=\"",
    "\"initialRoomDossier\":\"", // JSON format
}

// If all fail, try legacy API
legacyStream, err := fetchStreamLegacy(ctx, client, username)
```

### Legacy API Structure

```go
apiURL := "https://chaturbate.com/api/chatvideocontext/{username}/"

// Expected response:
{
    "room_status": "public",
    "hls_source": "https://...",
    "num_viewers": 123,
    ...
}
```

## References

- [yt-dlp Issue #12580](https://github.com/yt-dlp/yt-dlp/issues/12580) - Chaturbate m3u8 download failures (Feb 2025)
- [yt-dlp Issue #9594](https://github.com/yt-dlp/yt-dlp/issues/9594) - Chaturbate no longer downloads (Mar 2024)
- FlareSolverr Documentation: https://github.com/FlareSolverr/FlareSolverr

## Conclusion

The `initialRoomDossier` variable has been removed or changed by Chaturbate. Our legacy API fallback should handle this, but we need to wait for the next workflow run to confirm. If the API also fails, we'll need to reverse engineer the current method Chaturbate uses to deliver stream URLs.

**Status**: ⏳ Waiting for legacy API fallback test results
**Priority**: 🔴 High - Affects core functionality
**Impact**: Multiple channels showing as offline when they're live
