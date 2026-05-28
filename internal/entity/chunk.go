package entity

const ChunkSize = 5 << 20 // 5 MB

type ChunkUploadSession struct {
	UploadID     string `json:"upload_id"`
	AccountID    uint   `json:"account_id"`
	Filename     string `json:"filename"`
	FileSize     int64  `json:"file_size"`
	ChunkSize    int64  `json:"chunk_size"`
	TotalChunks  int    `json:"total_chunks"`
	FileHash     string `json:"file_hash"`
	UploadedBits []bool `json:"uploaded_bits"`
}

func (s *ChunkUploadSession) UploadedChunks() []int {
	var indices []int
	for i, uploaded := range s.UploadedBits {
		if uploaded {
			indices = append(indices, i)
		}
	}
	return indices
}

func (s *ChunkUploadSession) IsComplete() bool {
	for _, b := range s.UploadedBits {
		if !b {
			return false
		}
	}
	return true
}

type InitChunkUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type"`
}

type InitChunkUploadResponse struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	UploadID  string `json:"upload_id"`
}

type ChunkPartURLRequest struct {
	ObjectKey  string `json:"object_key" binding:"required"`
	UploadID   string `json:"upload_id" binding:"required"`
	PartNumber int    `json:"part_number" binding:"required,min=1"`
}

type ChunkPartURLResponse struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

type CompleteChunkUploadRequest struct {
	ObjectKey string              `json:"object_key" binding:"required"`
	UploadID  string              `json:"upload_id" binding:"required"`
	Parts     []CompleteChunkPart `json:"parts" binding:"required"`
}

type CompleteChunkPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type AbortChunkUploadRequest struct {
	ObjectKey string `json:"object_key" binding:"required"`
	UploadID  string `json:"upload_id" binding:"required"`
}
