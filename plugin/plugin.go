package plugin

import (
	"gopkg.in/sensorbee/gstreamer.v0"
	"gopkg.in/sensorbee/sensorbee.v0/bql"
)

func init() {
	bql.RegisterGlobalSourceCreator("gst_raw_video",
		bql.SourceCreatorFunc(gstreamer.CreateRawSource))
	bql.RegisterGlobalSourceCreator("gst_nvcamera",
		bql.SourceCreatorFunc(gstreamer.CreateNVCameraSource))
}
