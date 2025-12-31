package utils

import (
	"context"
	"os"
	"testing"
)

// TestChunkFile 测试文件分块功能（ChunkFile 在 file.go 中定义）
func TestChunkFile_File(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		chunkSize  int64
		wantChunks int
	}{
		{
			name:       "empty data",
			data:       []byte{},
			chunkSize:  10,
			wantChunks: 0,
		},
		{
			name:       "small data, single chunk",
			data:       []byte{1, 2, 3, 4, 5},
			chunkSize:  10,
			wantChunks: 1,
		},
		{
			name:       "exact chunk size",
			data:       make([]byte, 10),
			chunkSize:  10,
			wantChunks: 1,
		},
		{
			name:       "multiple chunks",
			data:       make([]byte, 25),
			chunkSize:  10,
			wantChunks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestProcessFileInChunks(t *testing.T) {
	ctx := context.Background()
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	tests := []struct {
		name    string
		config  *ChunkConfig
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultChunkConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &ChunkConfig{
				ChunkSize:   20,
				Concurrency: 2,
			},
			wantErr: false,
		},
		{
			name: "with progress callback",
			config: &ChunkConfig{
				ChunkSize:   20,
				Concurrency: 2,
				OnProgress: func(progress FileProgress) {
					// 验证进度信息
					if progress.Total != int64(len(data)) {
						t.Errorf("OnProgress: Total = %d, want %d", progress.Total, len(data))
					}
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ProcessFileInChunks(ctx, data, func(chunk []byte, index int) (int, error) {
				return len(chunk), nil
			}, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessFileInChunks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(results) == 0 {
					t.Error("ProcessFileInChunks() got 0 results")
				}
			}
		})
	}
}

func TestReadFileAsStream(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test_file_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 写入测试数据
	testData := make([]byte, 1000)
	for i := range testData {
		testData[i] = byte(i)
	}
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("Write test data failed: %v", err)
	}
	tmpFile.Close()

	// 测试流式读取
	var progressCount int
	data, err := ReadFileAsStream(tmpFile.Name(), func(progress FileProgress) {
		progressCount++
		if progress.Total != int64(len(testData)) {
			t.Errorf("ReadFileAsStream progress Total = %d, want %d", progress.Total, len(testData))
		}
	})

	if err != nil {
		t.Errorf("ReadFileAsStream() error = %v", err)
		return
	}

	if len(data) != len(testData) {
		t.Errorf("ReadFileAsStream() got %d bytes, want %d", len(data), len(testData))
	}

	// 验证数据内容
	for i := range data {
		if data[i] != testData[i] {
			t.Errorf("ReadFileAsStream() data[%d] = %d, want %d", i, data[i], testData[i])
		}
	}

	if progressCount == 0 {
		t.Error("ReadFileAsStream() progress callback not called")
	}
}

func TestReadFileInChunks(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test_file_*.bin")
	if err != nil {
		t.Fatalf("Create temp file failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 写入测试数据
	testData := make([]byte, 100)
	for i := range testData {
		testData[i] = byte(i)
	}
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("Write test data failed: %v", err)
	}
	tmpFile.Close()

	// 测试分块读取
	var chunks []int
	err = ReadFileInChunks(tmpFile.Name(), func(chunk []byte, index int) error {
		chunks = append(chunks, len(chunk))
		return nil
	}, &ChunkConfig{
		ChunkSize: 30,
	})

	if err != nil {
		t.Errorf("ReadFileInChunks() error = %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("ReadFileInChunks() got 0 chunks")
	}

	// 验证总长度
	totalLen := 0
	for _, chunkLen := range chunks {
		totalLen += chunkLen
	}
	if totalLen != len(testData) {
		t.Errorf("ReadFileInChunks() total length = %d, want %d", totalLen, len(testData))
	}
}

func TestEstimateProcessingTime(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		chunkSize       int64
		processingSpeed int64
		wantPositive    bool
	}{
		{
			name:            "small file",
			fileSize:        1024,
			chunkSize:       512,
			processingSpeed: 1024,
			wantPositive:    true,
		},
		{
			name:            "large file",
			fileSize:        100 * 1024 * 1024, // 100MB
			chunkSize:       5 * 1024 * 1024,   // 5MB
			processingSpeed: 10 * 1024 * 1024,  // 10MB/s
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

func TestDefaultChunkConfig(t *testing.T) {
	config := DefaultChunkConfig()
	if config.ChunkSize != 1024*1024 {
		t.Errorf("DefaultChunkConfig() ChunkSize = %d, want %d", config.ChunkSize, 1024*1024)
	}
	if config.Concurrency != 3 {
		t.Errorf("DefaultChunkConfig() Concurrency = %d, want 3", config.Concurrency)
	}
}
