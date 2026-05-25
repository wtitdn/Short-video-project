package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	endpoint  = "39.104.69.250:9000"
	accessKey = "admin"
	secretKey = "admin123456"
	bucket    = "videosys"
	useSSL    = false
)

var (
	minioClient *minio.Client
	minioCore   *minio.Core
)

type initUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

type initUploadResponse struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	UploadID  string `json:"upload_id"`
}

type partURLRequest struct {
	ObjectKey  string `json:"object_key"`
	UploadID   string `json:"upload_id"`
	PartNumber int    `json:"part_number"`
}

type partURLResponse struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

type completeUploadRequest struct {
	ObjectKey string         `json:"object_key"`
	UploadID  string         `json:"upload_id"`
	Parts     []completePart `json:"parts"`
}

type completePart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type abortUploadRequest struct {
	ObjectKey string `json:"object_key"`
	UploadID  string `json:"upload_id"`
}

func main() {
	var err error

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	}
	minioClient, err = minio.New(endpoint, opts)
	if err != nil {
		log.Fatal(err)
	}
	minioCore, err = minio.NewCore(endpoint, opts)
	if err != nil {
		log.Fatal(err)
	}
	if err := ensureBucket(context.Background()); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", indexPage)
	http.HandleFunc("/upload/init", withCORS(initMultipartUpload))
	http.HandleFunc("/upload/part-url", withCORS(createPartURL))
	http.HandleFunc("/upload/complete", withCORS(completeMultipartUpload))
	http.HandleFunc("/upload/abort", withCORS(abortMultipartUpload))

	log.Println("server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ensureBucket(ctx context.Context) error {
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func initMultipartUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req initUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.FileName) == "" {
		http.Error(w, "file_name is required", http.StatusBadRequest)
		return
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}

	ext := filepath.Ext(req.FileName)
	objectKey := fmt.Sprintf("videos/%s%s", uuid.NewString(), ext)
	uploadID, err := minioCore.NewMultipartUpload(
		r.Context(),
		bucket,
		objectKey,
		minio.PutObjectOptions{ContentType: req.ContentType},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, initUploadResponse{
		Bucket:    bucket,
		ObjectKey: objectKey,
		UploadID:  uploadID,
	})
}

func createPartURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req partURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectKey == "" || req.UploadID == "" || req.PartNumber <= 0 {
		http.Error(w, "object_key, upload_id and part_number are required", http.StatusBadRequest)
		return
	}

	query := url.Values{}
	query.Set("partNumber", fmt.Sprintf("%d", req.PartNumber))
	query.Set("uploadId", req.UploadID)

	presignedURL, err := minioClient.Presign(
		r.Context(),
		http.MethodPut,
		bucket,
		req.ObjectKey,
		24*time.Hour,
		query,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, partURLResponse{
		PartNumber: req.PartNumber,
		URL:        presignedURL.String(),
	})
}

func completeMultipartUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req completeUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectKey == "" || req.UploadID == "" || len(req.Parts) == 0 {
		http.Error(w, "object_key, upload_id and parts are required", http.StatusBadRequest)
		return
	}

	sort.Slice(req.Parts, func(i, j int) bool {
		return req.Parts[i].PartNumber < req.Parts[j].PartNumber
	})

	parts := make([]minio.CompletePart, 0, len(req.Parts))
	for _, p := range req.Parts {
		if p.PartNumber <= 0 || p.ETag == "" {
			http.Error(w, "invalid part", http.StatusBadRequest)
			return
		}
		parts = append(parts, minio.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       strings.Trim(p.ETag, `"`),
		})
	}

	info, err := minioCore.CompleteMultipartUpload(
		r.Context(),
		bucket,
		req.ObjectKey,
		req.UploadID,
		parts,
		minio.PutObjectOptions{},
	)
	fmt.Println(info, err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"bucket":     info.Bucket,
		"object_key": info.Key,
		"etag":       info.ETag,
		"location":   info.Location,
	})
}

func abortMultipartUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req abortUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ObjectKey == "" || req.UploadID == "" {
		http.Error(w, "object_key and upload_id are required", http.StatusBadRequest)
		return
	}

	if err := minioCore.AbortMultipartUpload(r.Context(), bucket, req.ObjectKey, req.UploadID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "aborted"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>MinIO Multipart Upload Demo</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 760px; margin: 40px auto; line-height: 1.5; }
    button { padding: 8px 14px; }
    progress { width: 100%; height: 18px; }
    pre { background: #111; color: #eee; padding: 12px; overflow: auto; }
  </style>
</head>
<body>
  <h2>MinIO Multipart Upload Demo</h2>
  <input id="file" type="file">
  <button id="upload">Upload</button>
  <p><progress id="progress" value="0" max="100"></progress></p>
  <pre id="log"></pre>

  <script>
    const chunkSize = 5 * 1024 * 1024;
    const logEl = document.querySelector("#log");
    const progress = document.querySelector("#progress");

    function log(message) {
      logEl.textContent += message + "\n";
    }

    async function postJSON(url, body) {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    }

    document.querySelector("#upload").onclick = async () => {
      const file = document.querySelector("#file").files[0];
      if (!file) {
        alert("choose a file first");
        return;
      }

      logEl.textContent = "";
      progress.value = 0;

      let init;
      try {
        init = await postJSON("/upload/init", {
          file_name: file.name,
          content_type: file.type || "application/octet-stream",
        });
        log("upload_id: " + init.upload_id);
        log("object_key: " + init.object_key);

        const partCount = Math.ceil(file.size / chunkSize);
        const parts = [];

        for (let i = 0; i < partCount; i++) {
          const partNumber = i + 1;
          const start = i * chunkSize;
          const end = Math.min(start + chunkSize, file.size);
          const chunk = file.slice(start, end);

          const partURL = await postJSON("/upload/part-url", {
            object_key: init.object_key,
            upload_id: init.upload_id,
            part_number: partNumber,
          });

          const putRes = await fetch(partURL.url, {
            method: "PUT",
            body: chunk,
          });
          if (!putRes.ok) throw new Error(await putRes.text());

          const etag = putRes.headers.get("ETag");
          if (!etag) throw new Error("missing ETag for part " + partNumber);

          parts.push({ part_number: partNumber, etag });
          progress.value = Math.round((partNumber / partCount) * 100);
          log("part " + partNumber + "/" + partCount + " uploaded, etag=" + etag);
        }

        const done = await postJSON("/upload/complete", {
          object_key: init.object_key,
          upload_id: init.upload_id,
          parts,
        });
        log("complete:");
        log(JSON.stringify(done, null, 2));
      } catch (err) {
        log("error: " + err.message);
        if (init) {
          await postJSON("/upload/abort", {
            object_key: init.object_key,
            upload_id: init.upload_id,
          }).catch(() => {});
          log("aborted multipart upload");
        }
      }
    };
  </script>
</body>
</html>`
