package gpx

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

var Isink int
var Bsink bool
var E error
var trkpSlice = []byte("<trkpt lat=\"37.942557\" lon=\"-5.760211\"><ele>615.25</ele></trkpt>")

func initData() []byte {
	gpxFileName := "./gpx/cazalla.gpx"
	gpxbytes, _ := os.ReadFile(gpxFileName)
	_, l := trkpCountEstimate(gpxbytes)
	trkpLen = l
	startSearch = trkpLen - (len(closetag) + 2)
	d := bytes.Index(gpxbytes, opentag)
	gpxbytes = gpxbytes[d:]
	return gpxbytes
}

// *****************************************************************
// Benchmark_XML_Unmarshal-8
//     144	   9023921 ns/op	 1765741 B/op	   51142 allocs/op
//
// Benchmark_ParseGPX-8 with indexTag and indexByte
//     4966	    236186 ns/op	   57608 B/op	       1 allocs/op
//
// ParseGPX = 9023921 / 236186 = 30.6 x faster

func Benchmark_ParseGPX(b *testing.B) {
	gpx := &GPX{}
	gpxbytes, _ := os.ReadFile("./gpx/cazalla.gpx")
	for range b.N {
		ParseGPX(gpxbytes, gpx, false) // 278784 ns/op. 35 x faster than xml.Unmarshal
	}
}
func Benchmark_XML_Unmarshal(b *testing.B) {
	gpx := &GPX{}
	gpxbytes, _ := os.ReadFile("./gpx/cazalla.gpx")
	for range b.N {
		xml.Unmarshal(gpxbytes, gpx) // 9774621 ns/op
	}
}

func Benchmark_indexTag_Long(b *testing.B) {
	gpxbytes := initData()
	i := 0
	for range b.N {
		i = indexTag(gpxbytes, []byte("</trkseg>"))
	}
	Isink = i
}
func Benchmark_Bytes_Index_long(b *testing.B) {
	gpxbytes := initData()
	i := 0
	for range b.N {
		i = bytes.Index(gpxbytes, []byte("</trkseg>"))
	}
	Isink = i
}

func Benchmark_indexTag_short(b *testing.B) {
	s := trkpSlice[30:]
	for range b.N {
		indexTag(s, eletag)
	}
}
func Benchmark_Bytes_Index_short(b *testing.B) {
	s := trkpSlice[30:]
	i := 0
	for range b.N {
		i = bytes.Index(s, eletag)
	}
	Isink = i
}

func Benchmark_Bytes_IndexByte_short(b *testing.B) {
	for range b.N {
		bytes.IndexByte(trkpSlice, 'k')
	}
}
func Benchmark_indexByte_short(b *testing.B) {
	for range b.N {
		indexByte(trkpSlice, 'k')
	}
}

func Benchmark_ParseTrkpt(b *testing.B) {
	s := trkpSlice
	var e error
	for range b.N {
		_, e = parseTrkpt(s)
	}
	E = e
}

func Benchmark_ParseElevation(b *testing.B) {
	s := trkpSlice
	var e error
	for range b.N {
		_, e = parseElevation(s)
	}
	E = e
}

// BenchmarkNextBytes-8 with bytes.Index and bytes.IndexByte
//	33912	     35217 ns/op	       4 B/op	       0 allocs/op
// BenchmarkNext-8 with indexTag and indexByte
//	40431	     25733 ns/op	       3 B/op	       0 allocs/op

func Benchmark_NextTrkpt(b *testing.B) {
	s := initData()
	var q []byte
	for range b.N {
		g := s
		for {
			q, g = nextTrkpt(g)
			if q == nil {
				break
			}
		}
	}
}

// Put use_std_library true to use bytes.Index and bytes.IndexByte

// BenchmarkParseAll-8 with bytes.Index and bytes.IndexByte
//     4292	    259424 ns/op	      32 B/op	       0 allocs/op
// BenchmarkParseAll-8 with indexTag and indexByte
//     5330	    189868 ns/op	      26 B/op	       0 allocs/op

func BenchmarkParseAll(b *testing.B) {
	s := initData()
	var q []byte
	for range b.N {
		g := s
		for {
			q, g = nextTrkpt(g)
			if q == nil {
				break
			}
			parseTrkpt(q)
		}
	}
}
