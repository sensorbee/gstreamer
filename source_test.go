package gstreamer

import (
	"fmt"
	"testing"

	"gopkg.in/sensorbee/sensorbee.v0/bql"
	"gopkg.in/sensorbee/sensorbee.v0/core"
	"gopkg.in/sensorbee/sensorbee.v0/data"
)

func TestCreateRawSource(t *testing.T) {
	t.Run("valid cases", func(t *testing.T) {
		cs := []map[string]interface{}{
			{
				"pipeline": "videotestsrc ! appsink param=1",
				"format":   "jpeg",
			},
			{
				"pipeline": "videotestsrc ! appsink",
				"format":   "jpeg",
				"width":    640,
				"height":   480,
			},
			{
				"pipeline":    "videotestsrc ! appsink",
				"format":      "raw",
				"width":       640,
				"height":      480,
				"color_model": "bgr",
			},
		}

		for i, c := range cs {
			t.Run(fmt.Sprint("case", i), func(t *testing.T) {
				m, err := data.NewMap(c)
				if err != nil {
					t.Fatal("can't convert a map:", c)
				}

				src, err := CreateRawSource(core.NewContext(nil), &bql.IOParams{}, m)
				if err != nil {
					t.Fatal("creating a source failed:", err)
				}

				s, ok := src.(*Source)
				if !ok {
					t.Fatal("can't convert src")
				}

				if s.pipeline != c["pipeline"].(string) {
					t.Error("pipeline isn't set")
				}
				if s.format != c["format"].(string) {
					t.Error("format isn't set")
				}

				switch s.format {
				case "raw":
					if s.width != c["width"].(int) {
						t.Error("width isn't set")
					}
					if s.height != c["height"].(int) {
						t.Error("height isn't set")
					}
					if s.format == "raw" && s.colorModel != c["color_model"].(string) {
						t.Error("color_model isn't set")
					}
				}
			})
		}
	})

	t.Run("invalid cases", func(t *testing.T) {
		t.Run("general", func(t *testing.T) {
			cs := []struct {
				n string
				m map[string]interface{}
			}{
				{
					n: "missing pipeline",
					m: map[string]interface{}{
						"format": "jpeg",
					},
				},
				{
					n: "missing format",
					m: map[string]interface{}{
						"pipeline": "videotestsrc ! appsink",
					},
				},
				{
					n: "unsupported format",
					m: map[string]interface{}{
						"pipeline": "videotestsrc ! appsink",
						"format":   "png",
					},
				},
				{ // this is already tested in data.Decode. So, only this is ok.
					n: "invalid value type",
					m: map[string]interface{}{
						"pipeline": "videotestsrc ! appsink",
						"format":   1,
					},
				},
			}

			for _, c := range cs {
				t.Run(c.n, func(t *testing.T) {
					m, err := data.NewMap(c.m)
					if err != nil {
						t.Fatal("can't convert a map:", c.m)
					}

					_, err = CreateRawSource(core.NewContext(nil), &bql.IOParams{}, m)
					if err == nil {
						t.Error("CreateRawSource didn't fail")
					}
				})
			}
		})

		t.Run("raw", func(t *testing.T) {
			cs := []struct {
				n       string
				missing string
			}{
				{
					n:       "missing width",
					missing: "width",
				},
				{
					n:       "missing height",
					missing: "height",
				},
				{
					n:       "unsupported format",
					missing: "color_model",
				},
			}

			for _, c := range cs {
				m := data.Map{
					"pipeline":    data.String("videotestsrc ! appsink"),
					"format":      data.String("raw"),
					"width":       data.Int(640),
					"height":      data.Int(480),
					"color_model": data.String("bgr"),
				}
				delete(m, c.missing)
				_, err := CreateRawSource(core.NewContext(nil), &bql.IOParams{}, m)
				if err == nil {
					t.Error("CreateRawSource didn't fail:", c.n)
				}
			}
		})
	})
}
