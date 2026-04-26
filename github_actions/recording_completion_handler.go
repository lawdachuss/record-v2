package github_actions

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// RecordingCompletionHandler handles the workflow when a recording completes.
// It coordinates uploads to external storage, database updates, notifications,
// and local file cleanup.
//
// This handler is designed to be called from the channel's finalizeRecording
// method after the recording file has been finalized (remuxed/transcoded).
//
// Requirements: 3.1, 3.7, 6.7, 14.1, 15.3
type RecordingCompletionHandler struct {
	storageUploader *StorageUploader
	databaseManager *DatabaseManager
	supabaseManager *SupabaseManager
	healthMonitor   *HealthMonitor
	sessionID       string
	matrixJobID     string
}

// NewRecordingCompletionHandler creates a new handler for recording completion events.
func NewRecordingCompletionHandler(
	storageUploader *StorageUploader,
	databaseManager *DatabaseManager,
	supabaseManager *SupabaseManager,
	healthMonitor *HealthMonitor,
	sessionID string,
	matrixJobID string,
) *RecordingCompletionHandler {
	return &RecordingCompletionHandler{
		storageUploader: storageUploader,
		databaseManager: databaseManager,
		supabaseManager: supabaseManager,
		healthMonitor:   healthMonitor,
		sessionID:       sessionID,
		matrixJobID:     matrixJobID,
	}
}

// HandleRecordingCompletion processes a completed recording file.
// It performs the following operations in sequence:
// 1. Upload the file to Gofile and Filester in parallel
// 2. Add recording metadata to the database
// 3. Send notification via Health Monitor
// 4. Delete the local file after successful upload
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - filePath: Full path to the completed recording file
//   - site: The streaming site name (e.g., "chaturbate", "stripchat")
//   - channel: The channel username
//   - startTime: When the recording started
//   - duration: Recording duration in seconds
//
// Returns an error if any critical step fails. The method logs all operations
// for monitoring and debugging.
//
// Requirements: 3.1, 3.7, 6.7, 14.1, 15.3
func (rch *RecordingCompletionHandler) HandleRecordingCompletion(
	ctx context.Context,
	filePath string,
	site string,
	channel string,
	startTime time.Time,
	duration float64,
) error {
	log.Printf("[RecordingCompletionHandler] Processing completed recording: %s", filePath)
	log.Printf("[RecordingCompletionHandler] Site: %s, Channel: %s, Duration: %.2fs", site, channel, duration)

	// Step 1: Get file information
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat recording file: %w", err)
	}
	fileSizeBytes := fileInfo.Size()
	log.Printf("[RecordingCompletionHandler] File size: %d bytes", fileSizeBytes)

	// Step 2: Upload to Gofile and Filester in parallel (Requirement 3.1, 14.1)
	log.Printf("[RecordingCompletionHandler] Starting dual upload to Gofile and Filester...")
	uploadResult, err := rch.storageUploader.UploadRecording(ctx, filePath)
	if err != nil {
		// Upload failed - log error but don't fail the entire operation
		// The file has been preserved in artifacts as fallback
		log.Printf("[RecordingCompletionHandler] Upload failed: %v", err)
		
		// Send detailed notification about upload failure and fallback usage (Requirement 8.3)
		notificationTitle := fmt.Sprintf("Upload Failed - Fallback to Artifacts - %s", channel)
		notificationMessage := fmt.Sprintf(
			"Failed to upload recording for channel %s to both Gofile and Filester.\n\n"+
			"File Details:\n"+
			"  - Channel: %s\n"+
			"  - Site: %s\n"+
			"  - File: %s\n"+
			"  - Size: %d bytes (%.2f MB)\n"+
			"  - Duration: %.2fs\n"+
			"  - Session: %s\n"+
			"  - Matrix Job: %s\n\n"+
			"Fallback Action:\n"+
			"  The file has been preserved for GitHub Artifacts upload.\n"+
			"  It will be available in the workflow artifacts section.\n"+
			"  Retention: 7 days\n\n"+
			"Error: %v",
			channel,
			channel,
			site,
			filePath,
			fileSizeBytes,
			float64(fileSizeBytes)/(1024*1024),
			duration,
			rch.sessionID,
			rch.matrixJobID,
			err,
		)
		
		notificationErr := rch.healthMonitor.SendNotification(notificationTitle, notificationMessage)
		if notificationErr != nil {
			log.Printf("[RecordingCompletionHandler] Failed to send upload failure notification: %v", notificationErr)
		}
		
		// Don't delete the local file if upload failed - it needs to be preserved for artifacts
		log.Printf("[RecordingCompletionHandler] Local file preserved for artifact upload: %s", filePath)
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("[RecordingCompletionHandler] Upload completed successfully")
	log.Printf("[RecordingCompletionHandler] Gofile URL: %s", uploadResult.GofileURL)
	log.Printf("[RecordingCompletionHandler] Filester URL: %s", uploadResult.FilesterURL)
	if len(uploadResult.FilesterChunks) > 0 {
		log.Printf("[RecordingCompletionHandler] Filester chunks: %d", len(uploadResult.FilesterChunks))
	}

	// Step 3: Determine quality string from file name or use default
	// Quality format: "{resolution}p{framerate}" (e.g., "2160p60", "1080p60")
	quality := extractQualityFromFilename(filePath)
	if quality == "" {
		quality = "unknown"
	}
	log.Printf("[RecordingCompletionHandler] Recording quality: %s", quality)

	// Step 4: Add recording metadata to database (Requirement 15.3)
	log.Printf("[RecordingCompletionHandler] Adding recording to database...")
	date := rch.databaseManager.FormatDate(startTime)
	metadata := RecordingMetadata{
		Timestamp:      rch.databaseManager.FormatTimestamp(startTime),
		DurationSec:    int(duration),
		FileSizeBytes:  fileSizeBytes,
		Quality:        quality,
		GofileURL:      uploadResult.GofileURL,
		FilesterURL:    uploadResult.FilesterURL,
		FilesterChunks: uploadResult.FilesterChunks,
		SessionID:      rch.sessionID,
		MatrixJob:      rch.matrixJobID,
	}

	err = rch.databaseManager.AddRecording(site, channel, date, metadata)
	if err != nil {
		log.Printf("[RecordingCompletionHandler] Failed to add recording to database: %v", err)
		
		// Send notification about database failure
		notificationErr := rch.healthMonitor.SendNotification(
			"Database Update Failed",
			fmt.Sprintf("Failed to add recording metadata to database for channel %s: %v. Recording is uploaded but not indexed.", channel, err),
		)
		if notificationErr != nil {
			log.Printf("[RecordingCompletionHandler] Failed to send database failure notification: %v", notificationErr)
		}
		
		// Continue with notification and cleanup even if database update fails
		// The recording is safely uploaded, just not indexed
	} else {
		log.Printf("[RecordingCompletionHandler] Recording added to database successfully")
	}

	// Step 4.5: Add recording to Supabase (if configured)
	if rch.supabaseManager != nil {
		log.Printf("[RecordingCompletionHandler] Adding recording to Supabase...")
		
		supabaseRecording := SupabaseRecording{
			Site:           site,
			Channel:        channel,
			Timestamp:      startTime,
			Date:           date,
			DurationSec:    int(duration),
			FileSizeBytes:  fileSizeBytes,
			Quality:        quality,
			GofileURL:      uploadResult.GofileURL,
			FilesterURL:    uploadResult.FilesterURL,
			FilesterChunks: uploadResult.FilesterChunks,
			SessionID:      rch.sessionID,
			MatrixJob:      rch.matrixJobID,
		}
		
		insertedRecord, err := rch.supabaseManager.InsertRecording(supabaseRecording)
		if err != nil {
			log.Printf("[RecordingCompletionHandler] Failed to add recording to Supabase: %v", err)
			
			// Send notification about Supabase failure
			notificationErr := rch.healthMonitor.SendNotification(
				"Supabase Update Failed",
				fmt.Sprintf("Failed to add recording metadata to Supabase for channel %s: %v. Recording is uploaded and in JSON database.", channel, err),
			)
			if notificationErr != nil {
				log.Printf("[RecordingCompletionHandler] Failed to send Supabase failure notification: %v", notificationErr)
			}
			
			// Continue even if Supabase insert fails - JSON database is primary
		} else {
			log.Printf("[RecordingCompletionHandler] Recording added to Supabase successfully (ID: %s)", insertedRecord.ID)
		}
	} else {
		log.Printf("[RecordingCompletionHandler] Supabase not configured, skipping Supabase insert")
	}

	// Step 5: Send notification via Health Monitor (Requirement 6.7)
	log.Printf("[RecordingCompletionHandler] Sending completion notification...")
	notificationTitle := fmt.Sprintf("Recording Completed - %s", channel)
	notificationMessage := fmt.Sprintf(
		"Channel: %s\nDuration: %ds\nSize: %d bytes\nQuality: %s\nGofile: %s\nFilester: %s\nSession: %s\nMatrix Job: %s",
		channel,
		int(duration),
		fileSizeBytes,
		quality,
		uploadResult.GofileURL,
		uploadResult.FilesterURL,
		rch.sessionID,
		rch.matrixJobID,
	)
	
	err = rch.healthMonitor.SendNotification(notificationTitle, notificationMessage)
	if err != nil {
		log.Printf("[RecordingCompletionHandler] Failed to send completion notification: %v", err)
		// Continue with cleanup even if notification fails
	} else {
		log.Printf("[RecordingCompletionHandler] Completion notification sent successfully")
	}

	// Step 6: Delete local file after successful upload (Requirement 3.7)
	// Note: The StorageUploader.UploadRecording method already deletes the file
	// after successful dual upload, so we only need to verify it was deleted
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[RecordingCompletionHandler] Local file deleted successfully: %s", filePath)
	} else if err == nil {
		// File still exists - this shouldn't happen if upload succeeded
		log.Printf("[RecordingCompletionHandler] WARNING: Local file still exists after upload: %s", filePath)
		log.Printf("[RecordingCompletionHandler] Attempting to delete local file...")
		if err := os.Remove(filePath); err != nil {
			log.Printf("[RecordingCompletionHandler] Failed to delete local file: %v", err)
			// Don't fail the operation - file is uploaded and indexed
		} else {
			log.Printf("[RecordingCompletionHandler] Local file deleted successfully")
		}
	} else {
		log.Printf("[RecordingCompletionHandler] Error checking file status: %v", err)
	}

	log.Printf("[RecordingCompletionHandler] Recording completion handling finished successfully")
	return nil
}

// extractQualityFromFilename attempts to extract quality information from the filename.
// This is a helper function that looks for common quality patterns in filenames.
// Returns an empty string if quality cannot be determined.
func extractQualityFromFilename(filePath string) string {
	// This is a placeholder implementation
	// In a real implementation, this would parse the filename or use metadata
	// from the recording process to determine the actual quality
	
	// For now, return empty string to indicate unknown quality
	// The actual quality should be passed from the recording engine
	return ""
}

// GetStorageUploader returns the storage uploader instance.
func (rch *RecordingCompletionHandler) GetStorageUploader() *StorageUploader {
	return rch.storageUploader
}

// GetDatabaseManager returns the database manager instance.
func (rch *RecordingCompletionHandler) GetDatabaseManager() *DatabaseManager {
	return rch.databaseManager
}

// GetSupabaseManager returns the supabase manager instance.
func (rch *RecordingCompletionHandler) GetSupabaseManager() *SupabaseManager {
	return rch.supabaseManager
}

// GetHealthMonitor returns the health monitor instance.
func (rch *RecordingCompletionHandler) GetHealthMonitor() *HealthMonitor {
	return rch.healthMonitor
}

// GetSessionID returns the session ID.
func (rch *RecordingCompletionHandler) GetSessionID() string {
	return rch.sessionID
}

// GetMatrixJobID returns the matrix job ID.
func (rch *RecordingCompletionHandler) GetMatrixJobID() string {
	return rch.matrixJobID
}
