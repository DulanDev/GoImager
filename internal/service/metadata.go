package service

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"strings"
	"time"
)

type exifInfo struct {
	Camera   *string    `json:"camera"`
	TakenAt  *time.Time `json:"taken_at"`
	GPS      *gpsPoint  `json:"gps"`
}

type gpsPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Info struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Format     string  `json:"format"`
	SizeBytes  int64   `json:"size_bytes"`
	ColorModel string  `json:"color_model"`
	Exif       *exifInfo `json:"exif"`
}

const (
	tagMake              = 271
	tagModel             = 272
	tagDateTime          = 306
	tagDateTimeOriginal = 36867
	tagGPSLat            = 2
	tagGPSLatRef         = 1
	tagGPSLng            = 4
	tagGPSLngRef         = 3
)

func InfoFromReader(src io.Reader) (*Info, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, src); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, &ErrInvalid{Code: "INVALID_IMAGE", Message: fmt.Sprintf("could not decode image: %v", err)}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, &ErrInvalid{Code: "INVALID_IMAGE", Message: fmt.Sprintf("could not decode image: %v", err)}
	}
	info := &Info{
		Width:      cfg.Width,
		Height:     cfg.Height,
		Format:     format,
		SizeBytes:  int64(len(data)),
		ColorModel: colorModel(img),
		Exif:       parseExif(data),
	}
	return info, nil
}

func colorModel(img image.Image) string {
	switch img.(type) {
	case *image.NRGBA:
		return "NRGBA"
	case *image.RGBA:
		return "RGBA"
	case *image.Gray:
		return "Gray"
	case *image.Gray16:
		return "Gray16"
	case *image.YCbCr:
		return "YCbCr"
	case *image.Paletted:
		return "Paletted"
	default:
		return "RGBA"
	}
}

func parseExif(data []byte) *exifInfo {
	exif := scanJPEGExif(data)
	if exif == nil {
		return &exifInfo{}
	}
	return exif
}

func scanJPEGExif(data []byte) *exifInfo {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return nil
		}
		marker := data[i+1]
		if marker == 0xDA {
			return nil
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(data) {
			return nil
		}
		seg := data[i+4 : i+2+segLen]
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			tiff := seg[6:]
			return parseTIFF(tiff)
		}
		i += 2 + segLen
	}
	return nil
}

func parseTIFF(data []byte) *exifInfo {
	if len(data) < 8 {
		return nil
	}
	var bo binary.ByteOrder
	switch {
	case bytes.HasPrefix(data, []byte("II")):
		bo = binary.LittleEndian
	case bytes.HasPrefix(data, []byte("MM")):
		bo = binary.BigEndian
	default:
		return nil
	}
	offsetIFD0 := bo.Uint32(data[4:8])
	if int(offsetIFD0)+2 > len(data) {
		return nil
	}
	info := &exifInfo{}
	var makeStr, modelStr string
	ifd0 := readIFD(data, int(offsetIFD0), bo)
	for _, e := range ifd0 {
		switch e.tag {
		case tagMake:
			makeStr = readString(data, e, bo)
		case tagModel:
			modelStr = readString(data, e, bo)
		case tagDateTime:
			if t, err := parseExifTime(readString(data, e, bo)); err == nil {
				info.TakenAt = &t
			}
		}
	}
	camera := strings.TrimSpace(strings.TrimRight(makeStr+" "+modelStr, " "))
	if camera != "" {
		info.Camera = &camera
	}
	if exifOff := findExifIFD(data, int(offsetIFD0), bo); exifOff > 0 && exifOff+2 <= len(data) {
		for _, e := range readIFD(data, exifOff, bo) {
			if e.tag == tagDateTimeOriginal && info.TakenAt == nil {
				if t, err := parseExifTime(readString(data, e, bo)); err == nil {
					info.TakenAt = &t
				}
			}
		}
	}
	if gps := readGPS(data, int(offsetIFD0), bo); gps != nil {
		info.GPS = gps
	}
	return info
}

type ifdEntry struct {
	tag      uint16
	typ      uint16
	count    uint32
	valueOff uint32
}

var typeSizes = map[uint16]int{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8}

func readIFD(data []byte, off int, bo binary.ByteOrder) []ifdEntry {
	if off+2 > len(data) {
		return nil
	}
	count := int(bo.Uint16(data[off : off+2]))
	if off+2+count*12+4 > len(data) {
		return nil
	}
	var entries []ifdEntry
	for i := 0; i < count; i++ {
		base := off + 2 + i*12
		e := ifdEntry{
			tag:      bo.Uint16(data[base : base+2]),
			typ:      bo.Uint16(data[base+2 : base+4]),
			count:    uint32(bo.Uint32(data[base+4 : base+8])),
			valueOff: bo.Uint32(data[base+8 : base+12]),
		}
		entries = append(entries, e)
	}
	return entries
}

func readString(data []byte, e ifdEntry, bo binary.ByteOrder) string {
	size, ok := typeSizes[e.typ]
	if !ok {
		return ""
	}
	total := int(e.count) * size
	if total <= 4 {
		out := make([]byte, total)
		var raw [4]byte
		bo.PutUint32(raw[:], e.valueOff)
		copy(out, raw[:total])
		return trimZero(string(out))
	}
	off := int(e.valueOff)
	if off+total > len(data) {
		return ""
	}
	return trimZero(string(data[off : off+total]))
}

func trimZero(s string) string {
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return s
}

func findExifIFD(data []byte, off int, bo binary.ByteOrder) int {
	entries := readIFD(data, off, bo)
	for _, e := range entries {
		if e.tag == 0x8769 {
			return int(e.valueOff)
		}
	}
	return 0
}

func parseExifTime(s string) (time.Time, error) {
	for _, layout := range []string{"2006:01:02 15:04:05", "2006:01:02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unparseable")
}

func readGPS(data []byte, off int, bo binary.ByteOrder) *gpsPoint {
	gpsIFDOff := 0
	for _, e := range readIFD(data, off, bo) {
		if e.tag == 0x8825 {
			gpsIFDOff = int(e.valueOff)
			break
		}
	}
	if gpsIFDOff == 0 {
		return nil
	}
	entries := readIFD(data, gpsIFDOff, bo)
	var latRef, lngRef string
	var lat, lng [3]float64
	haveLat, haveLng := false, false
	for _, e := range entries {
		switch e.tag {
		case tagGPSLatRef:
			latRef = readString(data, e, bo)
		case tagGPSLngRef:
			lngRef = readString(data, e, bo)
		case tagGPSLat:
			r := readRationalArray(data, e, bo)
			if r != nil && len(r) >= 3 {
				lat[0], lat[1], lat[2] = r[0], r[1], r[2]
				haveLat = true
			}
		case tagGPSLng:
			r := readRationalArray(data, e, bo)
			if r != nil && len(r) >= 3 {
				lng[0], lng[1], lng[2] = r[0], r[1], r[2]
				haveLng = true
			}
		}
	}
	if !haveLat || !haveLng {
		return nil
	}
	latVal := lat[0] + lat[1]/60 + lat[2]/3600
	lngVal := lng[0] + lng[1]/60 + lng[2]/3600
	if latRef == "S" {
		latVal = -latVal
	}
	if lngRef == "W" {
		lngVal = -lngVal
	}
	return &gpsPoint{Lat: latVal, Lng: lngVal}
}

func readRationalArray(data []byte, e ifdEntry, bo binary.ByteOrder) []float64 {
	if e.typ != 5 || int(e.count) != 3 {
		return nil
	}
	off := int(e.valueOff)
	if off+24 > len(data) {
		return nil
	}
	out := make([]float64, 3)
	for i := 0; i < 3; i++ {
		num := bo.Uint32(data[off+i*8 : off+i*8+4])
		den := bo.Uint32(data[off+i*8+4 : off+i*8+8])
		if den == 0 {
			continue
		}
		out[i] = float64(num) / float64(den)
	}
	return out
}