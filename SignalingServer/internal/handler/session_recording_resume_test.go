package handler

import (
	"errors"
	"testing"

	"quickdesk/signaling/internal/service"
)

func TestParseArtifactUploadHeaders(t *testing.T) {
	cases := []struct {
		name        string
		offset      string
		length      string
		complete    string
		probe       string
		wantOffset  int64
		wantLength  int64
		wantDone    bool
		wantProbe   bool
		wantFailure bool
	}{
		{"offset alone is a chunk with no declared total", "0", "", "", "", 0, 0, false, false, false},
		{"a declared total is kept", "0", "4096", "", "", 0, 4096, false, false, false},
		{"star means the total is unknown", "1024", "*", "", "", 1024, 0, false, false, false},
		{"the complete flag is honoured", "1024", "", "1", "", 1024, 0, true, false, false},
		{"true and yes mean complete too", "0", "", "TRUE", "", 0, 0, true, false, false},
		{"an explicit zero is not complete", "0", "", "0", "", 0, 0, false, false, false},
		{"the probe flag is honoured", "1024", "4096", "", "1", 1024, 4096, false, true, false},
		{"a probe can be spelled true", "1024", "", "", "true", 1024, 0, false, true, false},
		{"a missing offset is refused", "", "10", "", "", 0, 0, false, false, true},
		{"a negative offset is refused", "-1", "", "", "", 0, 0, false, false, true},
		{"a non-numeric offset is refused", "abc", "", "", "", 0, 0, false, false, true},
		{"a zero total is refused", "0", "0", "", "", 0, 0, false, false, true},
		{"a total behind the offset is refused", "2048", "1024", "", "", 0, 0, false, false, true},
		{"a nonsense complete flag is refused", "0", "", "maybe", "", 0, 0, false, false, true},
		{"a nonsense probe flag is refused", "0", "", "", "maybe", 0, 0, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArtifactUploadHeaders(tc.offset, tc.length, tc.complete, tc.probe)
			if tc.wantFailure {
				if err == nil {
					t.Fatalf("parsed %+v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Offset != tc.wantOffset || got.Length != tc.wantLength ||
				got.Complete != tc.wantDone || got.Probe != tc.wantProbe {
				t.Fatalf("parsed %+v, want offset=%d length=%d complete=%v probe=%v",
					got, tc.wantOffset, tc.wantLength, tc.wantDone, tc.wantProbe)
			}
		})
	}
}

func TestPlanArtifactChunk(t *testing.T) {
	cases := []struct {
		name        string
		req         artifactUploadRequest
		staged      int64
		chunk       int64
		maxBytes    int64
		wantWrite   bool
		wantFinish  bool
		wantFailure error
	}{
		{
			name:   "a middle chunk is written and the upload stays open",
			req:    artifactUploadRequest{Offset: 100, Length: 300},
			staged: 100, chunk: 100, wantWrite: true,
		},
		{
			name:   "reaching the declared total finishes",
			req:    artifactUploadRequest{Offset: 200, Length: 300},
			staged: 200, chunk: 100, wantWrite: true, wantFinish: true,
		},
		{
			name:   "the complete flag finishes without a declared total",
			req:    artifactUploadRequest{Offset: 5, Complete: true},
			staged: 5, chunk: 1, wantWrite: true, wantFinish: true,
		},
		{
			name:   "an empty probe writes nothing and stays open",
			req:    artifactUploadRequest{Offset: 42, Length: 300},
			staged: 42, chunk: 0,
		},
		{
			name:   "an empty probe finishing an already-complete upload finalises it",
			req:    artifactUploadRequest{Offset: 300, Length: 300},
			staged: 300, chunk: 0, wantFinish: true,
		},
		{
			name:   "a chunk that starts elsewhere is a conflict",
			req:    artifactUploadRequest{Offset: 99, Length: 300},
			staged: 100, chunk: 10, wantFailure: errArtifactOffsetMismatch,
		},
		{
			name:   "a chunk past the declared total is refused",
			req:    artifactUploadRequest{Offset: 200, Length: 300},
			staged: 200, chunk: 101, wantFailure: errArtifactLengthExceeded,
		},
		{
			name:   "a chunk past the configured cap is refused",
			req:    artifactUploadRequest{Offset: 0},
			staged: 0, chunk: 11, maxBytes: 10, wantFailure: errArtifactUploadTooLarge,
		},
		{
			name:   "complete before the declared total is refused",
			req:    artifactUploadRequest{Offset: 200, Length: 300, Complete: true},
			staged: 200, chunk: 10, wantFailure: errArtifactUploadIncomplete,
		},
		{
			name:   "finishing with nothing written is an empty artifact",
			req:    artifactUploadRequest{Offset: 0, Complete: true},
			staged: 0, chunk: 0, wantFailure: service.ErrArtifactEmpty,
		},
		{
			name:   "the cap does not bite when it is disabled",
			req:    artifactUploadRequest{Offset: 0},
			staged: 0, chunk: 5 << 20, wantWrite: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			write, finish, err := planArtifactChunk(tc.req, tc.staged, tc.chunk, tc.maxBytes)
			if tc.wantFailure != nil {
				if !errors.Is(err, tc.wantFailure) {
					t.Fatalf("err = %v, want %v", err, tc.wantFailure)
				}
				if write || finish {
					t.Fatalf("a refused chunk reported write=%v finish=%v", write, finish)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if write != tc.wantWrite || finish != tc.wantFinish {
				t.Fatalf("write=%v finish=%v, want write=%v finish=%v",
					write, finish, tc.wantWrite, tc.wantFinish)
			}
		})
	}
}
