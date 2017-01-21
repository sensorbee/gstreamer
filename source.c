// This code is based on https://github.com/dkorobkov/gstreamer-appsrc-appsink-example/blob/master/JpegGstEncoder_TegraTX1.cpp

#include "source.h"

#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <gst/app/gstappsink.h>

struct Source_ {
    pthread_t streaming_thread;
    pthread_mutex_t mtx; // mainly for loop_error
    gboolean thread_started;

    GstElement *pipeline;
    GstAppSink *sink;
    GMainLoop *loop;
    guint bus_watch_id;
    GError *loop_error;
};

gboolean busHandler(GstBus *bus, GstMessage *msg, gpointer data) {
    Source *s = (Source*) data;
    switch (GST_MESSAGE_TYPE(msg)) {
    case GST_MESSAGE_EOS: {
        g_main_loop_quit(s->loop);
    } break;
    case GST_MESSAGE_ERROR: {
        gchar *debug;
        GError *err;
        gst_message_parse_error(msg, &err, &debug);
        g_free(debug);
        pthread_mutex_lock(&s->mtx);
        if (!s->loop_error) { // just in case
            s->loop_error = err;
        } else {
            g_error_free(err);
        }
        pthread_mutex_unlock(&s->mtx);
        g_main_loop_quit(s->loop);
    } break;
    }
    return TRUE;
}

void *streamingThread(void *data) {
    Source *s = (Source*) data;
    gst_element_set_state(s->pipeline, GST_STATE_PLAYING);
    g_main_loop_run(s->loop);
    return NULL;
}

GError *startPipeline(Source *s) {
    GstBus *bus;
    // assuming all "g" things don't cause any error because there's no way
    // to report an error in the case of out-of-memory anyway.
    s->loop = g_main_loop_new(NULL, FALSE);
    bus = gst_pipeline_get_bus(GST_PIPELINE(s->pipeline));
    s->bus_watch_id = gst_bus_add_watch(bus, busHandler, s);
    gst_object_unref(bus);

    if (pthread_create(&s->streaming_thread, NULL, streamingThread, s) != 0) {
        return g_error_new_literal(1, 1, "cannot start a new thread");
    }
    s->thread_started = TRUE;
    return NULL;
}

GError *CreateAndStartSource(const char *pipeline_str, Source **src) {
    GError *err = NULL;
    Source *s = (Source*) malloc(sizeof(Source));
    if (!s) {
        return g_error_new_literal(1, 1, "cannot allocate memory"); // this is likely to fail, too.
    }
    memset(s, 0, sizeof(s));

    s->pipeline = gst_parse_launch(pipeline_str, &err);
    if (err) {
        DestroySource(s);
        return err;
    }

    // FIXME: creating multiple sources will create appsink1 or more.
    s->sink = (GstAppSink*) gst_bin_get_by_name(GST_BIN(s->pipeline), "appsink0");
    if (!s->sink) {
        DestroySource(s);
        return g_error_new_literal(1, 1, "pipeline doesn't have an appsink");
    }

    err = startPipeline(s);
    if (err) {
        DestroySource(s);
        return err;
    }

    *src = s;
    return NULL;
}

void DestroySource(Source *s) {
    // This implementation is only used by source.go and it doesn't call this
    // function concurrently. So, this function doesn't acquire locks.
    if (s->thread_started) {
        g_main_loop_quit(s->loop);
        pthread_join(s->streaming_thread, NULL);
        gst_element_set_state(s->pipeline, GST_STATE_NULL);
    }

    if (s->sink) gst_object_unref(s->sink);
    if (s->pipeline) gst_object_unref(s->pipeline);
    if (s->bus_watch_id) g_source_remove(s->bus_watch_id);
    if (s->loop) g_main_loop_unref(s->loop);
    if (s->loop_error) g_error_free(s->loop_error);
    free(s);
}

GError *GrabFrame(Source *s, GstBuffer **buf, GstMapInfo *m) {
    GstSample *sample;
    GError *err;
    pthread_mutex_lock(&s->mtx);
    err = s->loop_error;
    if (err) err = g_error_copy(err);
    pthread_mutex_unlock(&s->mtx);
    if (err) return err;

    sample = gst_app_sink_pull_sample(s->sink);
    if (!sample) {
        return g_error_new_literal(1, 1, "cannot grab next frame");
    }
    *buf = gst_sample_get_buffer(sample);
    gst_buffer_map(*buf, m, GST_MAP_READ);
    gst_buffer_ref(*buf); // keep buffer

    gst_sample_unref(sample);
    return NULL;
}

gsize GetFrameSize(GstMapInfo *m) {
    return m->size;
}

void CopyFrame(void *dst, GstMapInfo *m) {
    memcpy(dst, m->data, m->size);
}

void ReleaseFrame(GstBuffer *buf, GstMapInfo *m) {
    gst_buffer_unmap(buf, m);
    gst_buffer_unref(buf);
}
