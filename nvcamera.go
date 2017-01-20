package gstreamer

import (
	"fmt"
	"strings"

	"gopkg.in/sensorbee/sensorbee.v0/bql"
	"gopkg.in/sensorbee/sensorbee.v0/core"
	"gopkg.in/sensorbee/sensorbee.v0/data"
)

// NVCameraSourceOptions defines options for gst_nvcamera. This source itself
// doesn't deeply validate values of parameters and just lets GStreamer does it.
// Therefore, users should use gst-launch-1.0 command to validate parameters
// before creating a source to ease debugging the configuration. The pipeline
// this source creates depends on the format. When the format is raw, the
// pipeline will be:
//
//  nvcamerasrc !
//  video/x-raw(memory:NVMM),format=I420,width={{width}},height={{height}},framerate={{framerate}} !
//  nvvidconv flip-method={{flip_method}} !
//  video/x-raw !
//  videoconvert !
//  video/x-raw,format={{color_model}} !
//  appsink
//
// color_model will be capitalized. On the other hand, when the format is jpeg,
// the pipeline will be
//
//  nvcamerasrc !
//  video/x-raw(memory:NVMM),format=I420,width={{width}},height={{height}},framerate={{framerate}} !
//  nvvidconv flip-method={{flip_method}} !
//  nvjpegenc !
//  appsink
type NVCameraSourceOptions struct {
	// Width is the width of frames. The default value is 1280.
	Width int

	// Height is the height of frames. The default value is 720.
	Height int

	// Format specifies the format of frame images retrieved from the video
	// source. It supports raw or jpeg. The default value is jpeg.
	Format string

	// ColorModel represents the RGB layout of raw format. This option is only
	// referred when format is raw. The default value is "bgr". The value can
	// be either "bgr" or "rgb" at the moment.
	ColorModel string

	// Framerate is the framerate of the video in the format that GStreamer
	// accepts. For example, "30/1" (30FPS), "10/1" (10FPS), and so on. The
	// default value is "10/1".
	Framerate string

	// FilpMethod is the parameter for nvvidconv flip-method. The default value
	// is 2 (good for the default installation of the camera module).
	FlipMethod int
}

// CreateNVCameraSource creates a new gst_nvcamera source.
func CreateNVCameraSource(ctx *core.Context, ioParams *bql.IOParams, params data.Map) (core.Source, error) {
	s, err := createNVCameraSource(ioParams, params)
	if err != nil {
		return nil, err
	}
	return core.ImplementSourceStop(s), nil
}

func createNVCameraSource(ioParams *bql.IOParams, params data.Map) (*Source, error) {
	opt := NVCameraSourceOptions{
		Width:      1280,
		Height:     720,
		Format:     "jpeg",
		ColorModel: "bgr",
		Framerate:  "10/1",
		FlipMethod: 2,
	}
	if err := data.Decode(params, &opt); err != nil {
		return nil, err
	}

	pipeline := []string{
		"nvcamerasrc",
		"video/x-raw(memory:NVMM),format=I420,width=%v,height=%v,framerate=%v",
		"nvvidconv flip-method=%v",
	}
	values := []interface{}{opt.Width, opt.Height, opt.Framerate, opt.FlipMethod}

	switch opt.Format {
	case "jpeg":
		pipeline = append(pipeline, "nvjpegenc")
	case "raw":
		switch opt.ColorModel {
		case "rgb", "bgr":
		default:
			return nil, fmt.Errorf("unsupported color_model: %v", opt.ColorModel)
		}
		pipeline = append(pipeline,
			"video/x-raw",
			"videoconvert",
			"video/x-raw,format=%v")
		values = append(values, strings.ToUpper(opt.ColorModel))
	default:
		return nil, fmt.Errorf("unsupported format: %v", opt.Format)
	}
	pipeline = append(pipeline, "appsink")

	s := &Source{
		ioParams:   ioParams,
		pipeline:   fmt.Sprintf(strings.Join(pipeline, " ! "), values...),
		width:      opt.Width,
		height:     opt.Height,
		format:     opt.Format,
		colorModel: opt.ColorModel,
	}
	return s, nil
}
