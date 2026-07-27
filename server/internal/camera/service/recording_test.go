package service

import (
	"slices"
	"testing"
)

func TestCompletedSegmentPathsSkipsNewestActiveFile(t *testing.T) {
	files := []string{
		"cam_1_20260727_010010.mp4",
		"cam_1_20260727_010000.mp4",
		"cam_1_20260727_010020.mp4",
	}

	got := completedSegmentPaths(files, false)
	want := []string{
		"cam_1_20260727_010000.mp4",
		"cam_1_20260727_010010.mp4",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("completedSegmentPaths() = %v, want %v", got, want)
	}
}

func TestCompletedSegmentPathsIncludesNewestAfterFFmpegStops(t *testing.T) {
	files := []string{
		"cam_1_20260727_010010.mp4",
		"cam_1_20260727_010000.mp4",
	}

	got := completedSegmentPaths(files, true)
	want := []string{
		"cam_1_20260727_010000.mp4",
		"cam_1_20260727_010010.mp4",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("completedSegmentPaths() = %v, want %v", got, want)
	}
}

func TestRecordingSegmentsUseFastStart(t *testing.T) {
	args := buildFFmpegArgs("rtsp://camera/live", "/tmp/segment_%s.mp4", 300)
	if !hasAdjacentArgs(args, "-segment_format_options", "movflags=+faststart") {
		t.Fatalf("buildFFmpegArgs() does not enable MP4 faststart: %v", args)
	}
}

func TestRecordingSegmentsBreakAtConfiguredDuration(t *testing.T) {
	args := buildFFmpegArgs("rtsp://camera/live", "/tmp/segment_%s.mp4", 300)
	if !hasAdjacentArgs(args, "-break_non_keyframes", "1") {
		t.Fatalf("buildFFmpegArgs() does not enable precise segment boundaries: %v", args)
	}
}

func hasAdjacentArgs(args []string, key, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
