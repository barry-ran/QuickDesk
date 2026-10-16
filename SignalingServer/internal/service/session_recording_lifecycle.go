package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// RecordingArtifactUsage is the storage summary surfaced to admins.
type RecordingArtifactUsage struct {
	Count      int64
	TotalBytes int64
}

// SweepStaleRecordings flips recordings that have been stuck in "recording"
// since before the cutoff to "failed".
//
// Without this a client that crashes mid-recording leaves a row that claims a
// capture is still in progress forever, and the UI offers neither a result nor
// a retry.
func (s *SessionRecordingService) SweepStaleRecordings(ctx context.Context, cutoff time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("status = ? AND updated_at < ?", "recording", cutoff.UTC()).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ArtifactExpired decides whether a row's media file is past its retention.
//
// A row is only eligible when it still points at a file; the index row itself
// is deliberately kept so history survives the cleanup.
func ArtifactExpired(row models.SessionRecording, cutoff time.Time) bool {
	if row.ArtifactPath == "" && row.ArtifactURL == "" && row.ThumbnailPath == "" {
		return false
	}
	stamp := row.UpdatedAt
	if row.EndedAt != nil {
		stamp = *row.EndedAt
	}
	return stamp.Before(cutoff.UTC())
}

// ExpiredArtifacts returns the rows whose media files are past retention.
func (s *SessionRecordingService) ExpiredArtifacts(
	ctx context.Context,
	cutoff time.Time,
	limit int,
) ([]models.SessionRecording, error) {
	q := s.db.WithContext(ctx).
		Where("artifact_path <> '' OR artifact_url <> '' OR thumbnail_path <> ''").
		Where("COALESCE(ended_at, updated_at) < ?", cutoff.UTC()).
		Order("id ASC")
	if limit <= 0 {
		limit = 200
	}
	var rows []models.SessionRecording
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ClearArtifact forgets the stored media for a row while keeping the row.
func (s *SessionRecordingService) ClearArtifact(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"artifact_path":  "",
			"artifact_url":   "",
			"thumbnail_path": "",
			"thumbnail_url":  "",
			"size_bytes":     0,
			"updated_at":     time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ArtifactUsage reports how many recordings still hold media and how much disk
// they occupy according to the index.
func (s *SessionRecordingService) ArtifactUsage(ctx context.Context) (RecordingArtifactUsage, error) {
	var usage RecordingArtifactUsage
	row := struct {
		Count      int64
		TotalBytes int64
	}{}
	err := s.db.WithContext(ctx).
		Model(&models.SessionRecording{}).
		Select("COUNT(*) AS count, COALESCE(SUM(size_bytes), 0) AS total_bytes").
		Where("artifact_path <> '' OR artifact_url <> '' OR thumbnail_path <> ''").
		Scan(&row).Error
	if err != nil {
		return usage, err
	}
	usage.Count = row.Count
	usage.TotalBytes = row.TotalBytes
	return usage, nil
}

// RecordingLifecycle periodically reclaims storage and unsticks recordings.
type RecordingLifecycle struct {
	recordings *SessionRecordingService
	dir        string
	staleAfter time.Duration
	retention  time.Duration
	interval   time.Duration
	removeFile func(string) error
	purgeBatch int
}

// SweepResult is what one lifecycle pass changed.
type SweepResult struct {
	StaleMarkedFailed int64
	ArtifactsPurged   int
	BytesFreed        int64
	OrphansRemoved    int
	// UploadsReaped counts abandoned resumable-upload staging files that were
	// deleted. They are separate from orphans because their rule is age, not
	// "no row points at me" - see uploadTempMinAge.
	UploadsReaped int
}

// orphanMinAge keeps the orphan pass away from a file that an upload may still
// be writing. A row is always created before its bytes, so a file this old with
// no row is genuinely abandoned.
const orphanMinAge = time.Hour

// uploadTempMinAge is how long a `.part` staging file may sit untouched before
// the lifecycle deletes it. Resumable uploads make this file bigger and more
// likely to be abandoned than the single-shot path ever did, and nothing else
// reclaims it: the orphan pass deliberately ignores `.part` names so a pass can
// never delete a file that is still being written.
//
// Six hours is a trade-off, and the trade-off is visible to clients: a paused
// upload older than this must restart from zero, which is exactly the operation
// a resumable upload exists to make cheap. A shorter window would be more
// aggressive about disk, a longer one would leave abandoned captures lying
// around for a day.
const uploadTempMinAge = 6 * time.Hour

func NewRecordingLifecycle(
	recordings *SessionRecordingService,
	dir string,
	staleAfter time.Duration,
	retention time.Duration,
	interval time.Duration,
) *RecordingLifecycle {
	return &RecordingLifecycle{
		recordings: recordings,
		dir:        dir,
		staleAfter: staleAfter,
		retention:  retention,
		interval:   interval,
		removeFile: os.Remove,
		purgeBatch: 200,
	}
}

// Run blocks until ctx is cancelled, sweeping on every tick. Retention of zero
// means "keep media forever", so only the stale sweep runs in that case.
func (l *RecordingLifecycle) Run(ctx context.Context) {
	if l == nil || l.recordings == nil || l.interval <= 0 {
		return
	}
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = l.SweepOnce(ctx)
		}
	}
}

// RecordFileID extracts the recording id a stored file belongs to, or false when
// the name is not one this service writes (`rec-<id>.<ext>` / `thumb-<id>.<ext>`).
// Upload temporaries (`...mp4.part`) deliberately do not parse, so a pass can
// never delete a file that is still being written.
func RecordFileID(name string) (uint, bool) {
	rest := ""
	switch {
	case strings.HasPrefix(name, "rec-"):
		rest = strings.TrimPrefix(name, "rec-")
	case strings.HasPrefix(name, "thumb-"):
		rest = strings.TrimPrefix(name, "thumb-")
	default:
		return 0, false
	}
	digits := strings.TrimSuffix(rest, filepath.Ext(rest))
	if digits == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// UploadTempRecordingID extracts the recording id from a staging file
// (`rec-<id>.<ext>.part` / `thumb-<id>.<ext>.part`). The reaper treats these
// separately from stored media: age decides, not whether a row still exists,
// because a half-finished upload legitimately belongs to a row that is very
// much alive.
func UploadTempRecordingID(name string) (uint, bool) {
	if !strings.HasSuffix(name, ".part") {
		return 0, false
	}
	return RecordFileID(strings.TrimSuffix(name, ".part"))
}

// ReapOrphanFiles deletes two kinds of leftover file and reports the two counts
// separately, because they are cleaned for different reasons.
//
// Stored media whose index row no longer exists: deleting a row already deletes
// its files, but an unlink can fail (a locked or permission-protected file), and
// without this pass such a file would never be reclaimed, because every other
// cleanup path starts from a row.
//
// Staging files (`...part`) that have not been written for uploadTempMinAge: a
// resumable upload the client abandoned, or never came back to.
func (l *RecordingLifecycle) ReapOrphanFiles(ctx context.Context) (orphans int, uploads int, err error) {
	if l == nil || l.recordings == nil || l.dir == "" {
		return 0, 0, nil
	}
	userDirs, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	now := time.Now().UTC()
	for _, userDir := range userDirs {
		if !userDir.IsDir() {
			continue
		}
		dir := filepath.Join(l.dir, userDir.Name())
		entries, err := os.ReadDir(dir)
		if err != nil {
			return orphans, uploads, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// Staging files are decided by age alone.
			if _, isTemp := UploadTempRecordingID(entry.Name()); isTemp {
				info, err := entry.Info()
				if err != nil || now.Sub(info.ModTime().UTC()) < uploadTempMinAge {
					continue
				}
				if removeErr := l.removeFile(filepath.Join(dir, entry.Name())); removeErr != nil &&
					!os.IsNotExist(removeErr) {
					continue
				}
				uploads++
				continue
			}
			id, ok := RecordFileID(entry.Name())
			if !ok {
				continue
			}
			info, err := entry.Info()
			if err != nil || now.Sub(info.ModTime().UTC()) < orphanMinAge {
				continue
			}
			indexed, err := l.recordings.Exists(ctx, id)
			if err != nil {
				return orphans, uploads, err
			}
			if indexed {
				continue
			}
			if removeErr := l.removeFile(filepath.Join(dir, entry.Name())); removeErr != nil &&
				!os.IsNotExist(removeErr) {
				continue
			}
			orphans++
		}
	}
	return orphans, uploads, nil
}

// SweepOnce performs one stale sweep, one retention pass and one orphan pass,
// and reports what changed. It is also the implementation behind the admin
// trigger.
func (l *RecordingLifecycle) SweepOnce(ctx context.Context) (SweepResult, error) {
	var result SweepResult
	if l == nil || l.recordings == nil {
		return result, errors.New("recording lifecycle is not configured")
	}
	now := time.Now().UTC()
	if l.staleAfter > 0 {
		stale, err := l.recordings.SweepStaleRecordings(ctx, now.Add(-l.staleAfter))
		if err != nil {
			return result, err
		}
		result.StaleMarkedFailed = stale
	}
	if l.retention > 0 {
		if err := l.purgeExpired(ctx, now, &result); err != nil {
			return result, err
		}
	}
	orphans, uploads, err := l.ReapOrphanFiles(ctx)
	if err != nil {
		return result, err
	}
	result.OrphansRemoved = orphans
	result.UploadsReaped = uploads
	return result, nil
}

func (l *RecordingLifecycle) purgeExpired(ctx context.Context, now time.Time, result *SweepResult) error {
	cutoff := now.Add(-l.retention)
	rows, err := l.recordings.ExpiredArtifacts(ctx, cutoff, l.purgeBatch)
	if err != nil {
		return err
	}
	for i := range rows {
		row := rows[i]
		// A row can hold both a media file and a preview; delete every stored
		// file, and if any deletion fails keep the row untouched so a later pass
		// can retry it rather than orphaning the file.
		retry := false
		for _, path := range []string{row.ArtifactPath, row.ThumbnailPath} {
			if path == "" {
				continue
			}
			if removeErr := l.removeFile(path); removeErr != nil && !os.IsNotExist(removeErr) {
				retry = true
				break
			}
		}
		if retry {
			continue
		}
		if err = l.recordings.ClearArtifact(ctx, row.ID); err != nil {
			return err
		}
		result.ArtifactsPurged++
		result.BytesFreed += row.SizeBytes
	}
	return nil
}
