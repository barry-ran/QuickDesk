package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// The update artifact store needs the same housekeeping the recording store
// has, and one reason of its own: installers are large, so an abandoned
// resumable upload leaves a lot of disk behind. Nothing else reclaims it - the
// upload only cleans up when it finishes or when the release row is deleted,
// and a client that simply walks away does neither.
//
// Two rules, reported separately because they clean different things:
//
//   - A staging file (`<name>.part`) older than staleUpload is an upload nobody
//     came back to. Age decides, not whether the row still exists: a
//     half-finished upload legitimately belongs to a live release.
//   - A stored artifact with no row pointing at it is an orphan. Every path that
//     writes one also removes it on failure, but a crash between the rename and
//     the database write leaves a file nothing will ever look at again.

// UpdateArtifactSweepResult is what one housekeeping pass changed.
type UpdateArtifactSweepResult struct {
	UploadsReaped  int
	BytesReaped    int64
	OrphansRemoved int
	OrphanBytes    int64
}

// UpdateArtifactLifecycle periodically reclaims the update artifact store.
type UpdateArtifactLifecycle struct {
	releases   *UpdateReleaseService
	dir        string
	staleAfter time.Duration
	interval   time.Duration
	removeFile func(string) error
}

func NewUpdateArtifactLifecycle(
	releases *UpdateReleaseService,
	dir string,
	staleAfter time.Duration,
	interval time.Duration,
) *UpdateArtifactLifecycle {
	return &UpdateArtifactLifecycle{
		releases:   releases,
		dir:        dir,
		staleAfter: staleAfter,
		interval:   interval,
		removeFile: os.Remove,
	}
}

// Run blocks until ctx is cancelled, sweeping on every tick. A zero interval
// disables the worker; the admin trigger still works.
func (l *UpdateArtifactLifecycle) Run(ctx context.Context) {
	if l == nil || l.releases == nil || l.interval <= 0 {
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

// SweepOnce performs one pass and reports what changed.
func (l *UpdateArtifactLifecycle) SweepOnce(ctx context.Context) (UpdateArtifactSweepResult, error) {
	var result UpdateArtifactSweepResult
	if l == nil || l.releases == nil {
		return result, errors.New("update artifact lifecycle is not configured")
	}
	if err := l.reap(ctx, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (l *UpdateArtifactLifecycle) reap(ctx context.Context, result *UpdateArtifactSweepResult) error {
	if strings.TrimSpace(l.dir) == "" {
		return nil
	}
	releaseDirs, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	now := time.Now().UTC()
	for _, releaseDir := range releaseDirs {
		if !releaseDir.IsDir() {
			continue
		}
		// A directory that is not a release id is not ours to clean up: a
		// misplaced file is an operator's problem, not a sweep's.
		id, err := strconv.ParseUint(releaseDir.Name(), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		dir := filepath.Join(l.dir, releaseDir.Name())
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".part") {
				if l.staleAfter <= 0 || now.Sub(info.ModTime().UTC()) < l.staleAfter {
					continue
				}
				if err := l.remove(path); err != nil {
					continue
				}
				result.UploadsReaped++
				result.BytesReaped += info.Size()
				continue
			}
			// A stored artifact has to be old enough that an upload cannot still
			// be finishing under it, and then has to be unclaimed to be an orphan.
			if now.Sub(info.ModTime().UTC()) < orphanMinAge {
				continue
			}
			claimed, err := l.claimedByRow(ctx, uint(id), path)
			if err != nil {
				return err
			}
			if claimed {
				continue
			}
			if err := l.remove(path); err != nil {
				continue
			}
			result.OrphansRemoved++
			result.OrphanBytes += info.Size()
		}
		if remaining, err := os.ReadDir(dir); err == nil && len(remaining) == 0 {
			_ = os.Remove(dir)
		}
	}
	return nil
}

// claimedByRow reports whether the release row still points at this exact path.
// Comparing the path rather than just the id matters: a release that was
// re-uploaded under another name has a row, but the old file is already an
// orphan.
func (l *UpdateArtifactLifecycle) claimedByRow(ctx context.Context, id uint, path string) (bool, error) {
	release, err := l.releases.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return release.ArtifactPath != "" && release.ArtifactPath == path, nil
}

// remove deletes a file, treating "already gone" as success so a concurrent
// delete is not reported as a failure.
func (l *UpdateArtifactLifecycle) remove(path string) error {
	err := l.removeFile(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
