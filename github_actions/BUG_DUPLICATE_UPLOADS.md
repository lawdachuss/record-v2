# BUG FIX: Duplicate Upload Issue

## Problem Description

The system was uploading recordings multiple times, creating duplicate entries in Supabase and wasting bandwidth/storage.

## Root Cause Analysis

### Issue 1: No Duplicate Detection in Emergency Upload

The `UploadCompletedRecordings()` method in `github_actions_mode.go` was designed to upload recordings during emergency shutdown (when workflow is cancelled). However, it had **no duplicate detection logic**:

1. It scanned the recordings directory for all video files
2. It uploaded **every file** it found without checking if already uploaded
3. No query to Supabase to check for existing recordings
4. This caused the same recording to be uploaded multiple times

### Issue 2: Missing Supabase Metadata After Emergency Upload

After successfully uploading files during emergency shutdown, the system did **not** add metadata to Supabase. This meant:

1. Recordings uploaded during emergency shutdown were not tracked in the database
2. No way to query or find these recordings later
3. Inconsistent database state

### Issue 3: Incorrect Comment About File Deletion

The code had a misleading comment stating "StorageUploader.UploadRecording already deletes the file after successful upload", but this was **false**. The `UploadRecording()` method explicitly leaves file deletion to the handler (see line 889 in storage_uploader.go: "DO NOT delete file here - let the handler delete it after Supabase insert succeeds").

## Solution Implemented

### Fix 1: Added Duplicate Detection

Added duplicate detection logic in `UploadCompletedRecordings()`:

1. Before uploading, check if recording already exists in Supabase
2. Use file size matching with 1% tolerance (accounts for encoding differences)
3. Query recordings by date (using file modification time)
4. Skip upload if duplicate found
5. Delete local file if it's a duplicate

### Fix 2: Added Supabase Metadata Insertion

After successful upload during emergency shutdown:

1. Create `SupabaseRecording` struct with available metadata
2. Insert recording into Supabase database
3. Log success/failure of database insertion
4. Continue even if Supabase insert fails (recording is already uploaded)

### Fix 3: Added Explicit File Deletion

After successful upload and Supabase insertion:

1. Explicitly delete the local file
2. Log success/failure of file deletion
3. This ensures local storage is cleaned up properly

### Fix 4: Added Helper Method to Supabase Manager

Added `CheckRecordingExists()` method to `supabase_manager.go`:

```go
func (sm *SupabaseManager) CheckRecordingExists(date string, fileSizeBytes int64, tolerancePercent int) (*SupabaseRecording, error)
```

This method:
- Queries recordings by date
- Checks if any recording matches the file size within tolerance
- Returns the existing recording if found, nil otherwise
- Provides clean API for duplicate detection

## Files Modified

1. `github_actions/github_actions_mode.go`
   - Added duplicate detection in `UploadCompletedRecordings()`
   - Added Supabase metadata insertion after upload
   - Added explicit file deletion after successful upload

2. `github_actions/supabase_manager.go`
   - Added `CheckRecordingExists()` method for duplicate detection

## Testing Recommendations

1. Test emergency shutdown scenario:
   - Start recording
   - Cancel workflow (SIGINT/SIGTERM)
   - Verify recording is uploaded once
   - Verify Supabase entry is created
   - Verify local file is deleted

2. Test duplicate detection:
   - Upload a recording normally
   - Trigger emergency shutdown with same file
   - Verify duplicate is detected and skipped
   - Verify no duplicate Supabase entry

3. Test file size tolerance:
   - Create files with slightly different sizes (within 1% tolerance)
   - Verify they are detected as duplicates
   - Create files with significantly different sizes
   - Verify they are uploaded as separate recordings

## Impact

- **Prevents duplicate uploads**: Saves bandwidth and storage costs
- **Ensures database consistency**: All uploads are tracked in Supabase
- **Proper cleanup**: Local files are deleted after successful upload
- **Graceful degradation**: If Supabase check fails, upload continues (better to have duplicates than lose recordings)

## Related Issues

This fix addresses the user report: "its uploading the already uploaded recordings"
