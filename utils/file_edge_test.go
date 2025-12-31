package utils

import (
	"context"
	"os"
	"testing"
)

// TestChunkFile_EdgeCases 测试文件分块的边界情况
func TestChunkFile_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		chunkSize  int64
		wantChunks int
	}{
		{
			name:       "nil data",
			data:       nil,
			chunkSize:  10,
			wantChunks: 0,
		},
		{
			name:       "chunk size zero (should use default)",
			data:       make([]byte, 100),
			chunkSize:  0,
			wantChunks: 1, // 默认 1MB，100字节应该只有1块
		},
		{
			name:       "chunk size negative (should use default)",
			data:       make([]byte, 100),
			chunkSize:  -1,
			wantChunks: 1,
		},
		{
			name:       "chunk size 1",
			data:       make([]byte, 100),
			chunkSize:  1,
			wantChunks: 100,
		},
		{
			name:       "very large chunk size",
			data:       make([]byte, 100),
			chunkSize:  1000000,
			wantChunks: 1,
		},
		{
			name:       "chunk size exactly half",
			data:       make([]byte, 20),
			chunkSize:  10,
			wantChunks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.data == nil {
				tt.data = []byte{}
			}
			chunks := ChunkFile(tt.data, tt.chunkSize)
			if len(chunks) != tt.wantChunks {
				t.Errorf("ChunkFile() got %d chunks, want %d", len(chunks), tt.wantChunks)
			}

			// 验证所有数据都被包含
			totalLen := 0
			for _, chunk := range chunks {
				totalLen += len(chunk)
			}
			if totalLen != len(tt.data) {
				t.Errorf("ChunkFile() total length = %d, want %d", totalLen, len(tt.data))
			}
		})
	}
}

// TestProcessFileInChunks_EdgeCases 测试分块处理的边界情况
func TestProcessFileInChunks_EdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		data    []byte
		config  *ChunkConfig
		wantErr bool
	}{
		{
			name:    "nil data",
			data:    nil,
			config:  DefaultChunkConfig(),
			wantErr: false,
		},
		{
			name:    "concurrency zero (should use default)",
			data:    make([]byte, 100),
			config:  &ChunkConfig{ChunkSize: 10, Concurrency: 0},
			wantErr: false,
		},
		{
			name:    "chunk size zero (should use default)",
			data:    make([]byte, 100),
			config:  &ChunkConfig{ChunkSize: 0, Concurrency: 3},
			wantErr: false,
		},
		{
			name:    "very high concurrency",
			data:    make([]byte, 1000),
			config:  &ChunkConfig{ChunkSize: 10, Concurrency: 1000},
			wantErr: false,
		},
		{
			name:    "single concurrency",
			data:    make([]byte, 100),
			config:  &ChunkConfig{ChunkSize: 10, Concurrency: 1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.data == nil {
				tt.data = []byte{}
			}
			for i := range tt.data {
				tt.data[i] = byte(i)
			}

			results, err := ProcessFileInChunks(ctx, tt.data, func(chunk []byte, index int) (int, error) {
				return len(chunk), nil
			}, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessFileInChunks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(results) == 0 && len(tt.data) > 0 {
				t.Error("ProcessFileInChunks() got 0 results for non-empty data")
			}
		})
	}
}

// TestReadFileAsStream_EdgeCases 测试流式读取的边界情况
func TestReadFileAsStream_EdgeCases(t *testing.T) {
	// 创建空文件
	emptyFile, err := os.CreateTemp("", "test_empty_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(emptyFile.Name())
	emptyFile.Close()

	// 测试空文件
	var progressCount int
	data, err := ReadFileAsStream(emptyFile.Name(), func(progress FileProgress) {
		progressCount++
	})

	if err != nil {
		t.Errorf("ReadFileAsStream() error = %v", err)
	}
	if len(data) != 0 {
		t.Errorf("ReadFileAsStream() got %d bytes, want 0", len(data))
	}

	// 创建单字节文件
	singleByteFile, err := os.CreateTemp("", "test_single_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(singleByteFile.Name())
	singleByteFile.Write([]byte{0x01})
	singleByteFile.Close()

	data, err = ReadFileAsStream(singleByteFile.Name(), nil)
	if err != nil {
		t.Errorf("ReadFileAsStream() error = %v", err)
	}
	if len(data) != 1 || data[0] != 0x01 {
		t.Errorf("ReadFileAsStream() got %v, want [0x01]", data)
	}
}

// TestReadFileInChunks_EdgeCases 测试分块读取的边界情况
func TestReadFileInChunks_EdgeCases(t *testing.T) {
	// 创建空文件
	emptyFile, err := os.CreateTemp("", "test_empty_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(emptyFile.Name())
	emptyFile.Close()

	// 测试空文件
	var chunks []int
	err = ReadFileInChunks(emptyFile.Name(), func(chunk []byte, index int) error {
		chunks = append(chunks, len(chunk))
		return nil
	}, &ChunkConfig{ChunkSize: 10})

	if err != nil {
		t.Errorf("ReadFileInChunks() error = %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("ReadFileInChunks() got %d chunks, want 0", len(chunks))
	}

	// 创建小于分块大小的文件
	smallFile, err := os.CreateTemp("", "test_small_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(smallFile.Name())
	smallFile.Write([]byte{1, 2, 3})
	smallFile.Close()

	chunks = []int{}
	err = ReadFileInChunks(smallFile.Name(), func(chunk []byte, index int) error {
		chunks = append(chunks, len(chunk))
		return nil
	}, &ChunkConfig{ChunkSize: 10})

	if err != nil {
		t.Errorf("ReadFileInChunks() error = %v", err)
	}
	if len(chunks) != 1 || chunks[0] != 3 {
		t.Errorf("ReadFileInChunks() got %v, want [3]", chunks)
	}
}

// TestEstimateProcessingTime_EdgeCases 测试处理时间估算的边界情况
func TestEstimateProcessingTime_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		chunkSize       int64
		processingSpeed int64
		wantPositive    bool
	}{
		{
			name:            "zero file size",
			fileSize:        0,
			chunkSize:       1024,
			processingSpeed: 1024,
			wantPositive:    false,
		},
		{
			name:            "zero chunk size",
			fileSize:        1024,
			chunkSize:       0,
			processingSpeed: 1024,
			wantPositive:    false,
		},
		{
			name:            "zero processing speed",
			fileSize:        1024,
			chunkSize:       1024,
			processingSpeed: 0,
			wantPositive:    false,
		},
		{
			name:            "chunk size larger than file",
			fileSize:        100,
			chunkSize:       1000,
			processingSpeed: 1000,
			wantPositive:    true,
		},
		{
			name:            "very slow processing",
			fileSize:        1024,
			chunkSize:       1024,
			processingSpeed: 1,
			wantPositive:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := EstimateProcessingTime(tt.fileSize, tt.chunkSize, tt.processingSpeed)
			if tt.wantPositive && duration <= 0 {
				t.Errorf("EstimateProcessingTime() duration = %v, want positive", duration)
			}
		})
	}
}
