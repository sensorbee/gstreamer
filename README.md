# GStreamer video source plugin for SensorBee

This plugin is developed to use cameras on NVIDIA Jetson TX1. Because OpenCV2
pre-installed on JetPack doesn't seem to support GStreamer pipelines at the
moment [^1], I decided to write a plugin that directly uses GStreamer library to
obtain video frames from the default camera module.

## Prerequisites

The system needs to have GStreamer 1.0 and it has to be found by pkg-config.

## Usage

Add `gopkg.in/sensorbee/gstreamer.v0/plugin` to build.yaml's plugins section.

```
plugins:
  - gopkg.in/sensorbee/gstreamer.v0/plugin
```

## Sources

This plugin module has two source plugins:

* `gst_nvcamera`
* `gst_raw_video`

### `gst_nvcamera`

`gst_nvcamera` retrieves video frames using `nvcamerasrc` plugin of GStreamer.
This plugin doesn't strictly validate values of parameters and lets GStreamer
does it.

#### Parameters

`gst_nvcamera` only has optional parameters.

* `format`
* `color_model`
* `width`
* `height`
* `framerate`
* `flip_method`

##### `format`

`format` specifies the image format of retrieved frames. `"jpeg"` or `"raw"`
is supported. When the value is `"raw"`, the layout of RGB can be customized
by the `color_model` parameter. The default value of this parameter is `"jpeg"`.

##### `color_model`

`color_model` specifies the layout of RGB when `format` is `"raw"`. It only
accepts `"bgr"` or `"rgb"` at the moment. The default value is `"bgr"`,
which works well with OpenCV.

##### `width`

Width of a frame. The default value is 1280.

##### `height`

Height of a frame. The default value is 720.

##### `framerate`

Framerate of the video. The format of the value looks like "30/1" for 30FPS.
The default value is "10/1".

##### `flip_method`

`flip_method` has the value passed to `nvvidconv`'s `flip-method` parameter.
The default value is 2, which works well with the default camera module
mounted on the board.

#### Pipelines

Pipelines created by `gst_nvcamera` depends on `"format"`. When it's `"raw"`,
the pipeline will be:

```
nvcamerasrc !
video/x-raw(memory:NVMM),format=I420,width={{width}},height={{height}},framerate={{framerate}} !
nvvidconv flip-method={{flip_method}} !
video/x-raw !
videoconvert !
video/x-raw,format={{color_model}} !
appsink
```

`color_model` will be capitalized. For example, `"bgr"` will become `BGR`.

When `"format"` is `"jpeg"`,

```
nvcamerasrc !
video/x-raw(memory:NVMM),format=I420,width={{width}},height={{height}},framerate={{framerate}} !
nvvidconv flip-method={{flip_method}} !
nvjpegenc !
appsink
```

Run these plugins with `gst-launch-1.0` when there's a problem.

#### Examples

```
CREATE SOURCE video TYPE gst_nvcamera;
```

This create a `video` source with the default configuration, which is:

* `format` is `"jpeg"`
* `width`x`height` is 1280x720
* `framerate` is "10/1"

```
CREATE SOURCE video TYPE gst_nvcamera WITH
    format = "raw",
    width = 640,
    height = 480,
    framerate = "30/1";
```

This create a `video` source with a custom configuration as follows:

* `format` is `"raw"` and `color_model` is `"bgr"` by default
* `width`x`height` is 640x480
* `framerate` is "30/1"

#### Output

`"raw"`:

```
{
    "format": "raw",
    "color_model" "bgr" or "rgb",
    "width": width of the frame,
    "height": height of the frame,
    "image": (binary data of the image)
}
```

`"jpeg"`:

```
{
    "format": "jpeg",
    "width": width of the frame,
    "height": height of the frame,
    "image": (binary data of the image as a JPEG file)
}
```

### `gst_raw_video`

The `gst_raw_video` source allow a user to use custom GStreamer pipeline.

#### Parameters

`gst_raw_video` has following required parameters:

* `pipeline`
* `format`

When `format` is `raw`, it has additional requirement parameters:

* `color_model`
* `width`
* `height`

These parameters are same as ones defined in `gst_nvcamera`. When `format` is
`"jpeg"`, `width` and `height` are automatically detected.

#### Examples

```
CREATE SOURCE video TYPE gst_raw_video WITH
    pipeline = "videotestsrc ! video/x-raw,width=640,height=480 ! jpegenc ! appsink",
    format = "jpeg";
```

```
CREATE SOURCE video2 TYPE gst_raw_video WITH
    pipeline = "videotestsrc ! video/x-raw,format=BGR,width=640,height=480 ! appsink",
    format = "raw",
    color_model = "bgr",
    width = 640,
    height = 480;
```

#### Output

Same as `gst_nvcamera`.

[^1]: https://devtalk.nvidia.com/default/topic/904949/how-to-get-tx1-camera-in-opencv/
