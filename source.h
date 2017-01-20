#ifndef SENSORBEE_GSTREAMER_SOURCE_H_
#define SENSORBEE_GSTREAMER_SOURCE_H_

#include <gst/gst.h>

struct Source_;
typedef struct Source_ Source;

// CreateAndStartSource creates a new source from the given pipeline.
// The pipeline should have the valid format that gst-launch-1.0
// can correctly executes. The pipeline also has to end with appsink.
GError *CreateAndStartSource(const char *pipeline_str, Source **src);

// DestroySource releases all resources the source has.
// This function cannot be called concurrently when Grab
// is being called.
void DestroySource(Source *s);

// GrabFrame returns a new frame in buf and m. buf and m returned from
// this function have to be released by ReleaseFrame.
GError *GrabFrame(Source *s, GstBuffer **buf, GstMapInfo *m);

// GetBufferSize returns the size of the buffer.
gsize GetFrameSize(GstMapInfo *m);

// CopyFrame copies the data in GstMapInfo to Go's slice.
void CopyFrame(void *dst, GstMapInfo *m);

// ReleaseFrame releases the buffer returned from GrabFrame.
void ReleaseFrame(GstBuffer *buf, GstMapInfo *m);

#endif
