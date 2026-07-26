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
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-segment_format_options" && args[i+1] == "movflags=+faststart" {
			return
		}
	}
	t.Fatalf("buildFFmpegArgs() does not enable MP4 faststart: %v", args)
}
