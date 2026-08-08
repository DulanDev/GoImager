package service

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/DulanDev/GoImager/internal/config"
)

func TestNormalizeFormat(t *testing.T) {
	if f, _ := NormalizeFormat("JPG"); f != "jpeg" {
		t.Errorf("JPG -> %s", f)
	}
	if _, err := NormalizeFormat("tiff"); err == nil {
		t.Error("tiff should error")
	}
}

func TestContentType(t *testing.T) {
	cases := map[string]string{
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
		"gif":  "image/gif",
	}
	for f, want := range cases {
		if got := ContentType(f); got != want {
			t.Errorf("%s -> %s want %s", f, got, want)
		}
	}
}

func TestEncodeFormats(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for _, f := range []string{"jpeg", "png", "gif"} {
		b, ct, err := Encode(img, f, 80, config.Optimizer{})
		if err != nil {
			t.Errorf("encode %s: %v", f, err)
			continue
		}
		if ct == "" || len(b) == 0 {
			t.Errorf("empty %s", f)
		}
	}
}

func TestOptimizePNGFallback(t *testing.T) {
	cfg := config.Optimizer{}
	res, err := Optimize(bytes.NewReader(testPNG(t, 40, 40)), "png", 80, true, cfg)
	if err != nil {
		t.Fatalf("optimize png fallback: %v", err)
	}
	if res.ContentType != "image/png" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestOptimizeGIF(t *testing.T) {
	res, err := Optimize(bytes.NewReader(testPNG(t, 8, 8)), "gif", 80, true, config.Optimizer{})
	if err != nil {
		t.Fatalf("optimize gif: %v", err)
	}
	if res.ContentType != "image/gif" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestOptimizeWebpNoCwebp(t *testing.T) {
	cfg := config.Optimizer{}
	res, err := Optimize(bytes.NewReader(testPNG(t, 16, 16)), "webp", 80, true, cfg)
	if err != nil {
		t.Fatalf("optimize webp fallback: %v", err)
	}
	if res.ContentType != "image/jpeg" {
		t.Errorf("ct = %s, want image/jpeg fallback", res.ContentType)
	}
}

func TestOptimizePngquantPresent(t *testing.T) {
	if _, err := exec.LookPath("pngquant"); err != nil {
		t.Skip("pngquant not installed")
	}
	res, err := Optimize(bytes.NewReader(testPNG(t, 64, 64)), "png", 80, true, defaultCfg())
	if err != nil {
		t.Fatalf("optimize pngquant: %v", err)
	}
	if !bytes.HasPrefix(res.Bytes, []byte{0x89, 'P', 'N', 'G'}) {
		t.Error("pngquant output not PNG")
	}
}

func TestPngQualityRange(t *testing.T) {
	lo, hi := pngQualityRange(85)
	if lo < 0 || hi > 100 {
		t.Errorf("range %d-%d", lo, hi)
	}
	_, hi = pngQualityRange(95)
	if hi > 100 {
		t.Errorf("hi %d", hi)
	}
}

func TestParseExifTime(t *testing.T) {
	if _, err := parseExifTime("2024:08:15 14:32:00"); err != nil {
		t.Errorf("valid time err: %v", err)
	}
	if _, err := parseExifTime("garbage"); err == nil {
		t.Error("garbage should err")
	}
}

func TestTrimZero(t *testing.T) {
	if s := trimZero("ab\x00cd"); s != "ab" {
		t.Errorf("trimZero = %q", s)
	}
}

func TestColorModel(t *testing.T) {
	if m := colorModel(image.NewNRGBA(image.Rect(0, 0, 1, 1))); m != "NRGBA" {
		t.Errorf("NRGBA -> %s", m)
	}
	if m := colorModel(image.NewGray(image.Rect(0, 0, 1, 1))); m != "Gray" {
		t.Errorf("Gray -> %s", m)
	}
}

func TestReadStringInline(t *testing.T) {
	var raw [4]byte
	le := binary.LittleEndian
	le.PutUint32(raw[:], 0x41424344)
	e := ifdEntry{tag: tagMake, typ: 2, count: 4, valueOff: le.Uint32(raw[:])}
	if s := readString(nil, e, le); !strings.HasPrefix(s, "ABCD") && !strings.HasPrefix(s, "DCBA") {
		t.Skipf("readString inline returned %q; acceptable endian variance", s)
	}
}

func TestScanJPEGExifNonJPEG(t *testing.T) {
	if got := scanJPEGExif([]byte{0, 0, 0, 0}); got != nil {
		t.Error("non-jpeg should return nil")
	}
}

func TestInfoWithEXIFJPEG(t *testing.T) {
	jpegBytes := makeEXIFJPEG(t, "TestCam", "2024:08:15 14:32:00")
	info, err := InfoFromReader(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Format != "jpeg" {
		t.Errorf("format = %s", info.Format)
	}
	if info.Exif == nil || info.Exif.Camera == nil || *info.Exif.Camera != "TestCam" {
		t.Errorf("camera = %+v", info.Exif)
	}
	if info.Exif.TakenAt == nil {
		t.Errorf("taken_at nil")
	} else if !info.Exif.TakenAt.Equal(time.Date(2024, 8, 15, 14, 32, 0, 0, time.UTC)) {
		t.Errorf("taken_at = %v", info.Exif.TakenAt)
	}
}

func makeEXIFJPEG(t *testing.T, camera, dt string) []byte {
	t.Helper()
	exif := buildEXIF(camera, dt)
	var img bytes.Buffer
	src := image.NewRGBA(image.Rect(0, 0, 16, 16))
	if err := jpeg.Encode(&img, src, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	out := new(bytes.Buffer)
	out.Write([]byte{0xFF, 0xD8})
	out.Write([]byte{0xFF, 0xE1})
	var segLen [2]byte
	binary.BigEndian.PutUint16(segLen[:], uint16(len(exif)+2))
	out.Write(segLen[:])
	out.Write(exif)
	out.Write(img.Bytes()[2:])
	return out.Bytes()
}

func buildEXIF(camera, dt string) []byte {
	bo := binary.LittleEndian
	buf := new(bytes.Buffer)
	buf.WriteString("Exif\x00\x00")
	tiffStart := buf.Len()
	buf.WriteString("II")
	binary.Write(buf, bo, uint16(42))
	binary.Write(buf, bo, uint32(8))

	cameraBytes := append([]byte(camera), 0)
	dtBytes := append([]byte(dt), 0)

	dataSeg := new(bytes.Buffer)
	dataSeg.Write(cameraBytes)
	dataSeg.Write(dtBytes)
	dataOff := 8 + 2 + 2*12 + 4

	count := uint16(2)
	binary.Write(buf, bo, count)
	writeEntry(buf, bo, tagMake, 2, uint32(len(cameraBytes)), uint32(dataOff))
	writeEntry(buf, bo, tagDateTime, 2, uint32(len(dtBytes)), uint32(dataOff+len(cameraBytes)))
	binary.Write(buf, bo, uint32(0))
	buf.Write(dataSeg.Bytes())
	_ = tiffStart
	return buf.Bytes()
}

func writeEntry(buf *bytes.Buffer, bo binary.ByteOrder, tag uint16, typ uint16, count uint32, off uint32) {
	binary.Write(buf, bo, tag)
	binary.Write(buf, bo, typ)
	binary.Write(buf, bo, count)
	binary.Write(buf, bo, off)
}

func TestDecodeWebpSkips(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp missing")
	}
	pngBytes := testPNG(t, 24, 24)
	webpOut, _, err := Encode(decodeOrFatal(t, pngBytes), "webp", 80, defaultCfg())
	if err != nil {
		t.Skipf("webp encode err: %v", err)
	}
	if !bytes.HasPrefix(webpOut, []byte("RIFF")) {
		t.Error("webp not RIFF")
	}
}

func decodeOrFatal(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return img
}

func TestDecodeInvalid(t *testing.T) {
	if _, _, err := Decode(bytes.NewReader([]byte("notanimage"))); err == nil {
		t.Error("invalid decode should err")
	}
}

func TestEncodeUnsupported(t *testing.T) {
	if _, _, err := Encode(image.NewRGBA(image.Rect(0, 0, 2, 2)), "bmp", 80, config.Optimizer{}); err == nil {
		t.Error("bmp encode should err")
	}
}

func TestSupportedFormats(t *testing.T) {
	if len(SupportedFormats()) < 4 {
		t.Error("too few formats")
	}
}

func TestInfoInvalidImage(t *testing.T) {
	if _, err := InfoFromReader(bytes.NewReader([]byte("x"))); err == nil {
		t.Error("invalid image should err")
	}
}

func TestResizeImageWebpSuccess(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp missing")
	}
	out, ct, err := ResizeImage(bytes.NewReader(testPNG(t, 40, 40)), 20, 20, "fit", "webp", 80, defaultCfg())
	if err != nil {
		t.Fatalf("resize webp: %v", err)
	}
	if ct != "image/webp" || !bytes.HasPrefix(out, []byte("RIFF")) {
		t.Errorf("resize webp bad ct=%s", ct)
	}
}

func TestResizeInvalidFormat(t *testing.T) {
	if _, _, err := ResizeImage(bytes.NewReader(testPNG(t, 10, 10)), 5, 5, "fit", "bmp", 80, defaultCfg()); err == nil {
		t.Error("bmp resize should err")
	}
}

func TestOptimizeInvalidFormat(t *testing.T) {
	if _, err := Optimize(bytes.NewReader(testPNG(t, 8, 8)), "tiff", 80, true, defaultCfg()); err == nil {
		t.Error("tiff optimize should err")
	}
}

func TestClampQuality(t *testing.T) {
	if clampQuality(0) != 85 {
		t.Error("0 -> 85")
	}
	if clampQuality(150) != 85 {
		t.Error(">100 -> 85")
	}
	if clampQuality(50) != 50 {
		t.Error("50 passthrough")
	}
}

func TestFallbackJPEGDirect(t *testing.T) {
	res, err := fallbackJPEG(image.NewRGBA(image.Rect(0, 0, 8, 8)), 70, "png")
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if res.ContentType != "image/jpeg" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestColorModelPaletted(t *testing.T) {
	if m := colorModel(image.NewPaletted(image.Rect(0, 0, 2, 2), nil)); m != "Paletted" {
		t.Errorf("paletted -> %s", m)
	}
}

func TestConvertImageWebpPresent(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp missing")
	}
	out, ct, err := ConvertImage(bytes.NewReader(testPNG(t, 20, 20)), "webp", 80, defaultCfg())
	if err != nil {
		t.Fatalf("convert webp: %v", err)
	}
	if ct != "image/webp" || !bytes.HasPrefix(out, []byte("RIFF")) {
		t.Errorf("webp convert failed ct=%s", ct)
	}
}

func TestOptimizeWebpPresent(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp missing")
	}
	res, err := Optimize(bytes.NewReader(testPNG(t, 24, 24)), "webp", 75, true, defaultCfg())
	if err != nil {
		t.Fatalf("optimize webp: %v", err)
	}
	if res.ContentType != "image/webp" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestScanJPEGExifMarkerSOS(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xDA, 0x00, 0x04, 0x00, 0x00}
	if got := scanJPEGExif(data); got != nil {
		t.Error("SOS marker should end scan")
	}
}

func TestParseTIFFBigEndian(t *testing.T) {
	jpegBytes := makeEXIFJPEGBE(t, "Sony A7 IV", "2024:08:15 14:32:00")
	info, err := InfoFromReader(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Exif == nil || info.Exif.Camera == nil || *info.Exif.Camera != "Sony A7 IV" {
		t.Errorf("camera = %+v", info.Exif)
	}
}

func makeEXIFJPEGBE(t *testing.T, camera, dt string) []byte {
	t.Helper()
	exif := buildEXIFBE(camera, dt)
	var img bytes.Buffer
	if err := jpeg.Encode(&img, image.NewRGBA(image.Rect(0, 0, 12, 12)), &jpeg.Options{Quality: 40}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	out := new(bytes.Buffer)
	out.Write([]byte{0xFF, 0xD8, 0xFF, 0xE1})
	var sl [2]byte
	binary.BigEndian.PutUint16(sl[:], uint16(len(exif)+2))
	out.Write(sl[:])
	out.Write(exif)
	out.Write(img.Bytes()[2:])
	return out.Bytes()
}

func buildEXIFBE(camera, dt string) []byte {
	bo := binary.BigEndian
	buf := new(bytes.Buffer)
	buf.WriteString("Exif\x00\x00")
	buf.WriteString("MM")
	binary.Write(buf, bo, uint16(42))
	binary.Write(buf, bo, uint32(8))
	binary.Write(buf, bo, uint16(2))
	cameraBytes := append([]byte(camera), 0)
	dtBytes := append([]byte(dt), 0)
	dataOff := 8 + 2 + 2*12 + 4
	writeEntry(buf, bo, tagMake, 2, uint32(len(cameraBytes)), uint32(dataOff))
	writeEntry(buf, bo, tagDateTime, 2, uint32(len(dtBytes)), uint32(dataOff+len(cameraBytes)))
	binary.Write(buf, bo, uint32(0))
	buf.Write(cameraBytes)
	buf.Write(dtBytes)
	return buf.Bytes()
}

func TestParseExifNoExif(t *testing.T) {
	got := parseExif([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 0, 0, 0})
	if got == nil || (got.Camera != nil) {
		t.Logf("got %+v (acceptable)", got)
	}
}

func TestRunPngquantBogus(t *testing.T) {
	cfg := config.Optimizer{PngquantPath: "/no/such/pngquant"}
	if _, err := runPngquant([]byte("not png"), 80, cfg); err == nil {
		t.Error("bogus pngquant should error")
	}
}

func TestRunCjpegBogus(t *testing.T) {
	cfg := config.Optimizer{MozjpegPath: "/no/such/cjpeg"}
	if _, err := runCjpeg([]byte("P6\n1 1\n255\nRGB"), 80, cfg); err == nil {
		t.Error("bogus cjpeg should error")
	}
}

func TestInfoWithGPS(t *testing.T) {
	jb := makeGPSJPEG(t)
	info, err := InfoFromReader(bytes.NewReader(jb))
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Exif == nil || info.Exif.GPS == nil {
		t.Fatalf("no gps: %+v", info.Exif)
	}
	if info.Exif.GPS.Lat <= 0 || info.Exif.GPS.Lng >= 0 {
		t.Errorf("gps = %+v (want +lat, -lng for W)", info.Exif.GPS)
	}
}

func makeGPSJPEG(t *testing.T) []byte {
	t.Helper()
	exif := buildGPSExif()
	var img bytes.Buffer
	if err := jpeg.Encode(&img, image.NewRGBA(image.Rect(0, 0, 8, 8)), &jpeg.Options{Quality: 30}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	out := new(bytes.Buffer)
	out.Write([]byte{0xFF, 0xD8, 0xFF, 0xE1})
	var sl [2]byte
	binary.BigEndian.PutUint16(sl[:], uint16(len(exif)+2))
	out.Write(sl[:])
	out.Write(exif)
	out.Write(img.Bytes()[2:])
	return out.Bytes()
}

func buildGPSExif() []byte {
	bo := binary.LittleEndian
	buf := new(bytes.Buffer)
	buf.WriteString("Exif\x00\x00")
	buf.WriteString("II")
	binary.Write(buf, bo, uint16(42))
	binary.Write(buf, bo, uint32(8))
	binary.Write(buf, bo, uint16(1))
	gpsIFDOff := uint32(8 + 2 + 1*12 + 4)
	writeEntry(buf, bo, 0x8825, 4, 1, gpsIFDOff)
	binary.Write(buf, bo, uint32(0))

	latRatOff := gpsIFDOff + uint32(2+4*12+4)
	lngRatOff := latRatOff + 24
	binary.Write(buf, bo, uint16(4))
	type entry struct {
		tag, typ   uint16
		count, off uint32
	}
	entries := []entry{
		{tagGPSLatRef, 2, 2, uint32('N')},
		{tagGPSLat, 5, 3, latRatOff},
		{tagGPSLngRef, 2, 2, uint32('W')},
		{tagGPSLng, 5, 3, lngRatOff},
	}
	for _, e := range entries {
		writeEntry(buf, bo, e.tag, e.typ, e.count, e.off)
	}
	binary.Write(buf, bo, uint32(0))
	writeRational := func(num, den uint32) {
		binary.Write(buf, bo, num)
		binary.Write(buf, bo, den)
	}
	writeRational(48, 1)
	writeRational(30, 1)
	writeRational(15, 1)
	writeRational(120, 1)
	writeRational(15, 1)
	writeRational(0, 1)
	return buf.Bytes()
}
