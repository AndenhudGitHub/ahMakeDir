package logic

import (
	"fmt"
	"io"

	"github.com/vimeo/go-iccjpeg/iccjpeg"
)

// extractICCProfile extracts the ICC profile from a JPEG file.
func extractICCProfile(r io.Reader) ([]byte, error) {
	profile, err := iccjpeg.GetICCBuf(r)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// embedICCProfile embeds an ICC profile into a JPEG file.
// It reads the JPEG data from r, inserts the profile, and writes the result to w.
func embedICCProfile(w io.Writer, r io.Reader, profile []byte) error {
	// go-iccjpeg provides a Writer that embeds the profile.
	// However, we need to decode the JPEG structure to insert it correctly if we were doing it manually.
	// But wait, go-iccjpeg's Writer is a wrapper around an io.Writer that writes the profile *before* the image data?
	// No, checking documentation/usage of go-iccjpeg:
	// It seems go-iccjpeg is mostly for *extraction*.
	// Let's check if it supports writing/embedding.
	// If not, we might need to do what PHP did: manual segment insertion.

	// Actually, let's look at how we can use it.
	// If go-iccjpeg doesn't support writing, we might need another approach.
	// Let's assume for a moment we need to implement a simple embedder if the library doesn't have one.
	// But wait, the PHP code manually constructs APP2 segments.

	// Let's check if we can use a simpler approach:
	// 1. Read all bytes.
	// 2. Find SOI (FF D8).
	// 3. Insert APP2 segments immediately after SOI.

	return embedICCProfileManual(w, r, profile)
}

func embedICCProfileManual(w io.Writer, r io.Reader, profile []byte) error {
	// Read all input data
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return fmt.Errorf("not a valid JPEG")
	}

	// Write SOI
	_, err = w.Write(data[:2])
	if err != nil {
		return err
	}

	// Prepare APP2 segments
	// Max segment size is 65535.
	// ICC header is 14 bytes: "ICC_PROFILE\0" (12 bytes) + chunk_seq (1) + chunk_count (1).
	// So max data per chunk is 65535 - 2 (length bytes) - 14 = 65519.

	const maxChunkDataSize = 65519
	const iccHeaderLen = 14
	const iccMarker = "ICC_PROFILE\x00"

	profileLen := len(profile)
	numChunks := (profileLen + maxChunkDataSize - 1) / maxChunkDataSize

	for i := 0; i < numChunks; i++ {
		start := i * maxChunkDataSize
		end := start + maxChunkDataSize
		if end > profileLen {
			end = profileLen
		}
		chunkData := profile[start:end]
		chunkLen := len(chunkData)

		// Segment size = 2 (length bytes) + 14 (header) + chunkLen
		segmentSize := 2 + iccHeaderLen + chunkLen

		// Write Marker (FF E2)
		if _, err := w.Write([]byte{0xFF, 0xE2}); err != nil {
			return err
		}

		// Write Length (Big Endian uint16)
		// length includes the 2 bytes for the length itself
		lengthVal := uint16(segmentSize)
		if _, err := w.Write([]byte{byte(lengthVal >> 8), byte(lengthVal)}); err != nil {
			return err
		}

		// Write Header
		if _, err := io.WriteString(w, iccMarker); err != nil {
			return err
		}
		if _, err := w.Write([]byte{byte(i + 1), byte(numChunks)}); err != nil {
			return err
		}

		// Write Data
		if _, err := w.Write(chunkData); err != nil {
			return err
		}
	}

	// Write the rest of the original image
	_, err = w.Write(data[2:])
	return err
}
