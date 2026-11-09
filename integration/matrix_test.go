//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/matrix"
	"github.com/goforj/notification/notificationtest"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// matrixServer starts one non-federating SQLite homeserver with disposable local registration.
func matrixServer(t *testing.T) string {
	t.Helper()
	fixturePool.Lock()
	defer fixturePool.Unlock()
	if endpoint := fixturePool.servers["matrix"]; endpoint != "" {
		return endpoint
	}
	var versions map[string]struct{ Version, Image string }
	if err := json.Unmarshal(pinned, &versions); err != nil {
		t.Fatal(err)
	}
	config := `server_name: notification.test
pid_file: /tmp/notification-fixture.pid
signing_key_path: /tmp/notification-fixture.signing.key
media_store_path: /tmp/notification-media
report_stats: false
macaroon_secret_key: disposable-notification-fixture
registration_shared_secret: disposable-registration-fixture
enable_registration: true
enable_registration_without_verification: true
trusted_key_servers: []
suppress_key_server_warning: true
listeners:
  - port: 8008
    type: http
    tls: false
    bind_addresses: ['0.0.0.0']
    resources:
      - names: [client]
database:
  name: sqlite3
  args:
    database: /tmp/notification-fixture.db
rc_message:
  per_second: 1000
  burst_count: 1000
rc_registration:
  per_second: 1000
  burst_count: 1000
`
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	request := testcontainers.ContainerRequest{Image: versions["matrix"].Image, ExposedPorts: []string{"8008/tcp"}, Env: map[string]string{"SYNAPSE_CONFIG_PATH": "/data/homeserver.yaml", "SYNAPSE_REPORT_STATS": "no"}, Files: []testcontainers.ContainerFile{{Reader: strings.NewReader(config), ContainerFilePath: "/data/homeserver.yaml", FileMode: 0644}}, WaitingFor: wait.ForHTTP("/_matrix/client/versions").WithPort("8008/tcp").WithStartupTimeout(60 * time.Second)}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: request, Started: true})
	if container != nil {
		fixturePool.containers = append(fixturePool.containers, container)
	}
	if err != nil {
		t.Fatal("Synapse startup failed:", err)
	}
	code, output, err := container.Exec(ctx, []string{"python", "-c", "import synapse; print(synapse.__version__)"})
	if err != nil || code != 0 {
		t.Fatal("Synapse version check failed")
	}
	data, err := io.ReadAll(io.LimitReader(output, 4096))
	if err != nil || !strings.Contains(string(data), versions["matrix"].Version) {
		t.Fatal("unexpected Synapse version")
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "8008/tcp")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + host + ":" + port.Port()
	fixturePool.servers["matrix"] = endpoint
	return endpoint
}

// matrixAPI performs bounded fixture setup and inspection independently of the notification driver.
func matrixAPI(t *testing.T, method, endpoint, token, body string, result any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatal("Matrix fixture setup rejected:", response.StatusCode)
	}
	if result != nil && json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(result) != nil {
		t.Fatal("invalid Matrix fixture response")
	}
}

// TestMatrix verifies modern opaque room IDs, native authentication and stored plaintext events.
func TestMatrix(t *testing.T) {
	endpoint := matrixServer(t)
	var account struct {
		Token string `json:"access_token"`
	}
	user := fmt.Sprintf("fixture%d", time.Now().UnixNano())
	matrixAPI(t, http.MethodPost, endpoint+"/_matrix/client/v3/register", "", fmt.Sprintf(`{"username":%q,"password":"disposable-fixture-password","auth":{"type":"m.login.dummy"}}`, user), &account)
	if account.Token == "" {
		t.Fatal("missing disposable account token")
	}
	var room struct {
		ID string `json:"room_id"`
	}
	matrixAPI(t, http.MethodPost, endpoint+"/_matrix/client/v3/createRoom", account.Token, `{"preset":"private_chat","room_version":"12","creation_content":{"m.federate":false}}`, &room)
	if room.ID == "" || strings.Contains(room.ID, ":") {
		t.Fatal("expected a modern version-12 opaque room ID")
	}
	driver, err := matrix.New(matrix.Config{URL: endpoint, Token: account.Token, RoomID: room.ID, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	notificationtest.Sender(t, func() notification.Sender { return driver })
	var history struct {
		Chunk []struct {
			Type    string `json:"type"`
			Content struct {
				Type string `json:"msgtype"`
				Body string `json:"body"`
			} `json:"content"`
		} `json:"chunk"`
	}
	matrixAPI(t, http.MethodGet, endpoint+"/_matrix/client/v3/rooms/"+url.PathEscape(room.ID)+"/messages?dir=b&limit=20", account.Token, "", &history)
	found := 0
	for _, event := range history.Chunk {
		if event.Type == "m.room.message" {
			found++
			if event.Content.Type != "m.text" || event.Content.Body != "Ticket\nT-42 changed" {
				t.Fatal("stored Matrix message differs")
			}
		}
	}
	if found != 9 {
		t.Fatal("native Matrix message count:", found)
	}
	rejected, err := matrix.New(matrix.Config{URL: endpoint, Token: "invalid-disposable-token", RoomID: room.ID, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer rejected.Close()
	if err := rejected.Send(context.Background(), notification.Message{Body: "must reject"}); !errors.Is(err, notification.ErrUnauthorized) {
		t.Fatal("invalid Matrix access token accepted", err)
	}
}
