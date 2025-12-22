package util

import (
	"bytes"
	"compress/gzip"
	"io"
)

func GzipCompress(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(input)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GzipDecompress(compressedInput []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedInput))
	if err != nil {
		return nil, err
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = reader.Close()
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
