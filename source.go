// This package contains video data sources of SensorBee using GStreamer-1.0.
package gstreamer

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "source.h"
#include <stdlib.h>

const char* ErrorMessage(GError *err) {
    return err->message;
}
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
	"strings"
	"time"
	"unsafe"

	"gopkg.in/sensorbee/sensorbee.v0/bql"
	"gopkg.in/sensorbee/sensorbee.v0/core"
	"gopkg.in/sensorbee/sensorbee.v0/data"
)

func init() {
	C.gst_init(nil, nil)
}

// Source is a video source using GStreamer.
type Source struct {
	ioParams      *bql.IOParams
	pipeline      string
	width, height int
	format        string
	colorModel    string
}

func (s *Source) GenerateStream(ctx *core.Context, w core.Writer) error {
	var src *C.Source
	err := func() error {
		pipeline := C.CString(s.pipeline)
		defer C.free(unsafe.Pointer(pipeline))
		if e := C.CreateAndStartSource(pipeline, &src); e != nil {
			err := errors.New(C.GoString(C.ErrorMessage(e)))
			C.g_error_free(e)
			ctx.ErrLog(err).WithField("pipeline", s.pipeline).Error("Cannot create a pipeline")
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}
	defer func() {
		C.DestroySource(src)
	}()

	ctx.Log().WithFields(map[string]interface{}{
		"pipeline":  s.pipeline,
		"node_type": s.ioParams.TypeName,
		"node_name": s.ioParams.Name,
	}).Info("Start streaming")

	// It seems pulling frames before the main loop starts ends up with a dead-lock.
	// This could sometimes fail but should be sufficient for most cases.
	time.Sleep(time.Second)

	for {
		var (
			buf  *C.GstBuffer
			info C.GstMapInfo
		)
		if e := C.GrabFrame(src, &buf, &info); e != nil {
			err := errors.New(C.GoString(C.ErrorMessage(e)))
			C.g_error_free(e)
			return err
		}
		img := make(data.Blob, int(C.GetFrameSize(&info)))
		C.CopyFrame(unsafe.Pointer(&img[0]), &info)
		C.ReleaseFrame(buf, &info)

		// When the format is jpeg, this source automatically detects its width
		// and hights. It assumes that frames from the camera have the same size.
		if s.format == "jpeg" && (s.width == 0 || s.height == 0) {
			conf, err := jpeg.DecodeConfig(bytes.NewReader([]byte(img)))
			if err != nil {
				return err
			}
			s.width = conf.Width
			s.height = conf.Height
		}

		t := core.NewTuple(data.Map{
			"image":  img,
			"width":  data.Int(s.width),
			"height": data.Int(s.height),
			"format": data.String(s.format),
		})
		if s.format == "raw" {
			t.Data["color_model"] = data.String(s.colorModel)
		}
		if err := w.Write(ctx, t); err != nil {
			return err
		}
	}
}

func (s *Source) Stop(ctx *core.Context) error {
	return nil
}

// RawSourceOptions is options for gst_raw_video source.
type RawSourceOptions struct {
	// Pipeline contains a pipeline that passed to gst-launch-1.0. For example,
	//
	//  videotestsrc ! video/x-raw,format=BGR,width=640,height=480,framerate=30/1 ! appsink
	//
	// The pipeline must end with appsink.
	Pipeline string `bql:",required"`

	// Width is the width of frames captured by the pipeline. When the format is
	// jpeg, the source will automatically detect the size of frames. Otherwise,
	// this parameter is required.
	Width int

	// Height is the height of frames captured by the pipeline. When the format
	// is jpeg, the source will automatically detect the size of frames.
	// Otherwise, this parameter is required.
	Height int

	// Format specifies the format of frame images retrieved from the video
	// source. It supports raw or jpeg. The default value is jpeg.
	Format string `bql:",required"`

	// ColorModel is the color model (e.g. RGB, BGR, ARGB, etc) of frames. This
	// is only required when the format is raw.
	ColorModel string
}

type SourceOptions struct {
	Width  int
	Height int

	// Format specifies the format of frame images retrieved from the video
	// source. It supports raw or jpeg. The default value is jpeg.
	Format string

	// ColorModel represents the RGB layout of raw format. This option is only
	// referred when format is raw.
	ColorModel string

	// Framerate is the f
	Framerate string
}

func CreateRawSource(ctx *core.Context, ioParams *bql.IOParams, params data.Map) (core.Source, error) {
	opt := RawSourceOptions{}
	if err := data.Decode(params, &opt); err != nil {
		return nil, err
	}

	// TODO: report these validation errors at once.
	if ps := strings.Split(opt.Pipeline, "!"); len(ps) == 0 ||
		!strings.HasPrefix(strings.TrimSpace(ps[len(ps)-1]), "appsink") {
		return nil, errors.New("pipeline must end with appsink")
	}

	switch opt.Format {
	case "raw":
		if opt.Width == 0 || opt.Height == 0 {
			return nil, errors.New("width and heights must bw specified when the format is raw")
		}
		if opt.ColorModel == "" {
			return nil, errors.New("color_model is required when the format is raw")
		}
	case "jpeg":
	default:
		return nil, fmt.Errorf("unsupported format: %v", opt.Format)
	}

	s := &Source{
		ioParams:   ioParams,
		pipeline:   opt.Pipeline,
		width:      opt.Width,
		height:     opt.Height,
		format:     opt.Format,
		colorModel: opt.ColorModel,
	}
	return s, nil
}
