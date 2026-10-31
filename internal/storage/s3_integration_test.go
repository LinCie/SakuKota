//go:build integration

package storage_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"

	"rantau-go/internal/config"
	"rantau-go/internal/storage"
)

const (
	testBucket          = "rantau-images"
	testAccessKeyID     = "GK0123456789abcdef01234567"
	testSecretAccessKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	// testPublicEndpoint is a host the test never connects to. Requests are
	// sent to the container with this Host header instead.
	testPublicEndpoint = "http://images.example.test:3900"
)

// testEndpoint is the S3 API of the Garage container started by TestMain.
var testEndpoint string

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Same image and config as compose.yaml.
	container, err := testcontainers.Run(ctx, "dxflrs/garage:v2.5.0",
		testcontainers.WithEnv(map[string]string{
			"GARAGE_RPC_SECRET": "0000000000000000000000000000000000000000000000000000000000000000",
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      "../../docker/garage/garage.toml",
			ContainerFilePath: "/etc/garage.toml",
			FileMode:          0o644,
		}),
		testcontainers.WithExposedPorts("3900/tcp"),
		testcontainers.WithWaitStrategy(wait.ForLog("S3 API server listening")),
	)
	if err != nil {
		log.Fatalf("start garage container: %v", err)
	}

	if err := initGarage(ctx, container); err != nil {
		log.Fatalf("init garage: %v", err)
	}

	testEndpoint, err = container.PortEndpoint(ctx, "3900/tcp", "http")
	if err != nil {
		log.Fatalf("garage endpoint: %v", err)
	}

	code := m.Run()

	if err := container.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

// nodeID matches a full Garage node ID in the CLI's output.
var nodeID = regexp.MustCompile(`[0-9a-f]{64}`)

// initGarage sets up the node like docker/garage/init.sh: a layout, the test
// key, and the bucket.
func initGarage(ctx context.Context, c *testcontainers.DockerContainer) error {
	out, err := garage(ctx, c, "node", "id", "-q")
	if err != nil {
		return err
	}
	id := nodeID.FindString(out)
	if id == "" {
		return fmt.Errorf("no node id in %q", out)
	}

	for _, args := range [][]string{
		{"layout", "assign", "-z", "dc1", "-c", "1G", id},
		{"layout", "apply", "--version", "1"},
		{"key", "import", "-n", "test", "--yes", testAccessKeyID, testSecretAccessKey},
		{"bucket", "create", testBucket},
		{"bucket", "allow", "--read", "--write", "--owner", "--key", testAccessKeyID, testBucket},
	} {
		if _, err := garage(ctx, c, args...); err != nil {
			return err
		}
	}

	return nil
}

// garage runs the Garage CLI in the container and returns its output.
func garage(ctx context.Context, c *testcontainers.DockerContainer, args ...string) (string, error) {
	code, r, err := c.Exec(ctx, append([]string{"/garage"}, args...), tcexec.Multiplexed())
	if err != nil {
		return "", err
	}

	out, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("garage %s: exit %d: %s", strings.Join(args, " "), code, out)
	}

	return string(out), nil
}

// -----------------------------------------------------------------------------
// Fixtures and helpers
// -----------------------------------------------------------------------------

func newS3(t *testing.T) *storage.S3 {
	t.Helper()

	return storage.NewS3(config.StorageConfig{
		Endpoint:        testEndpoint,
		PublicEndpoint:  testPublicEndpoint,
		Region:          "garage",
		Bucket:          testBucket,
		AccessKeyID:     testAccessKeyID,
		SecretAccessKey: testSecretAccessKey,
	})
}

// testKey returns a key no other test uses.
func testKey(t *testing.T) string {
	t.Helper()

	return "test/" + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()) + ".png"
}

// send sends the presigned upload of body to the container as a client would
// send it to the public endpoint, and returns the response status.
func send(t *testing.T, upload storage.PresignedUpload, contentType, body string) int {
	t.Helper()

	if !strings.HasPrefix(upload.URL, testPublicEndpoint+"/") {
		t.Fatalf("upload URL = %q, want it on the public endpoint %s", upload.URL, testPublicEndpoint)
	}

	req, err := http.NewRequest(upload.Method, testEndpoint+strings.TrimPrefix(upload.URL, testPublicEndpoint), strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Host = strings.TrimPrefix(testPublicEndpoint, "http://")
	for name, value := range upload.Headers {
		req.Header.Set(name, value)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send upload: %v", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

// put stores body under key through a presigned upload.
func put(t *testing.T, s3 *storage.S3, key, contentType, body string) {
	t.Helper()

	upload, err := s3.PresignPut(context.Background(), key, contentType, time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if status := send(t, upload, contentType, body); status != http.StatusOK {
		t.Fatalf("upload status = %d, want 200", status)
	}
}

// -----------------------------------------------------------------------------
// Presigned uploads
// -----------------------------------------------------------------------------

func TestS3_PresignPut(t *testing.T) {
	s3 := newS3(t)
	key := testKey(t)

	before := time.Now()
	upload, err := s3.PresignPut(context.Background(), key, "image/png", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upload.Method != http.MethodPut {
		t.Errorf("method = %q, want PUT", upload.Method)
	}
	if want := before.Add(5 * time.Minute); upload.ExpiresAt.Before(want.Add(-time.Second)) ||
		upload.ExpiresAt.After(want.Add(5*time.Second)) {
		t.Errorf("expires at %v, want about %v", upload.ExpiresAt, want)
	}
	if status := send(t, upload, "image/png", "png bytes"); status != http.StatusOK {
		t.Fatalf("upload status = %d, want 200", status)
	}

	info, err := s3.Stat(context.Background(), key)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info != (storage.ObjectInfo{Size: int64(len("png bytes")), ContentType: "image/png"}) {
		t.Errorf("stored object = %+v, want 9 bytes of image/png", info)
	}
}

func TestS3_PresignPut_ContentTypeIsSigned(t *testing.T) {
	s3 := newS3(t)
	key := testKey(t)

	upload, err := s3.PresignPut(context.Background(), key, "image/png", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status := send(t, upload, "text/html", "<script></script>"); status != http.StatusForbidden {
		t.Errorf("upload with another content type: status = %d, want 403", status)
	}
	if _, err := s3.Stat(context.Background(), key); !errors.Is(err, storage.ErrObjectNotFound) {
		t.Errorf("stat err = %v, want ErrObjectNotFound", err)
	}
}

func TestS3_PresignPut_Expired(t *testing.T) {
	s3 := newS3(t)
	key := testKey(t)

	upload, err := s3.PresignPut(context.Background(), key, "image/png", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(2 * time.Second)

	// AWS answers 403 and Garage 400 ("Date is too old").
	if status := send(t, upload, "image/png", "png bytes"); status != http.StatusForbidden && status != http.StatusBadRequest {
		t.Errorf("expired upload: status = %d, want 400 or 403", status)
	}
}

// -----------------------------------------------------------------------------
// Stat, copy, and delete
// -----------------------------------------------------------------------------

func TestS3_Stat_Missing(t *testing.T) {
	_, err := newS3(t).Stat(context.Background(), testKey(t))
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("err = %v, want ErrObjectNotFound", err)
	}
}

func TestS3_Copy(t *testing.T) {
	s3 := newS3(t)
	src, dst := testKey(t), testKey(t)+".copy"
	put(t, s3, src, "image/webp", "webp bytes")

	if err := s3.Copy(context.Background(), src, dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{src, dst} {
		info, err := s3.Stat(context.Background(), key)
		if err != nil {
			t.Fatalf("stat %s: %v", key, err)
		}
		if info != (storage.ObjectInfo{Size: int64(len("webp bytes")), ContentType: "image/webp"}) {
			t.Errorf("%s = %+v, want 10 bytes of image/webp", key, info)
		}
	}
}

func TestS3_Copy_Missing(t *testing.T) {
	s3 := newS3(t)
	key := testKey(t)

	if err := s3.Copy(context.Background(), key, key+".copy"); err == nil {
		t.Fatal("err = nil, want an error for a missing source")
	}
}

func TestS3_Delete(t *testing.T) {
	s3 := newS3(t)
	key := testKey(t)
	put(t, s3, key, "image/png", "png bytes")

	if err := s3.Delete(context.Background(), key); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s3.Stat(context.Background(), key); !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("stat after delete: err = %v, want ErrObjectNotFound", err)
	}

	if err := s3.Delete(context.Background(), key); err != nil {
		t.Errorf("deleting a missing object: err = %v, want nil", err)
	}
}
