//go:build integration

package integration

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goforj/notification"
	"github.com/goforj/notification/driver/gotify"
	"github.com/goforj/notification/driver/ntfy"
	"github.com/goforj/notification/notificationtest"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// pinned binds integration claims to immutable upstream images.
//
//go:embed versions.json
var pinned []byte

// fixturePool owns one disposable server per provider for the whole package run.
var fixturePool = struct {
	sync.Mutex
	servers    map[string]string
	containers []testcontainers.Container
}{servers: map[string]string{}}

// TestMain cleans up every container, including partially configured fixtures.
func TestMain(m *testing.M) {
	code := m.Run()
	for i := len(fixturePool.containers) - 1; i >= 0; i-- {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := fixturePool.containers[i].Terminate(ctx)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "integration container cleanup failed")
			code = 1
		}
	}
	os.Exit(code)
}

// server starts each pinned engine once and waits for its real health/version endpoint.
func server(t *testing.T, name string) string {
	t.Helper()
	fixturePool.Lock()
	defer fixturePool.Unlock()
	if endpoint := fixturePool.servers[name]; endpoint != "" {
		return endpoint
	}
	var versions map[string]struct{ Version, Image string }
	if err := json.Unmarshal(pinned, &versions); err != nil {
		t.Fatal(err)
	}
	request := testcontainers.ContainerRequest{
		Image:        versions[name].Image,
		ExposedPorts: []string{"80/tcp"},
		WaitingFor:   wait.ForHTTP("/version").WithPort("80/tcp").WithStartupTimeout(60 * time.Second),
	}
	if name == "ntfy" {
		request.Cmd = []string{"serve"}
		request.WaitingFor = wait.ForHTTP("/v1/health").WithPort("80/tcp").WithStartupTimeout(60 * time.Second)
	} else {
		request.Env = map[string]string{"GOTIFY_DEFAULTUSER_NAME": "fixture", "GOTIFY_DEFAULTUSER_PASS": "fixture-password-disposable"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: request, Started: true})
	if container != nil {
		fixturePool.containers = append(fixturePool.containers, container)
	}
	if err != nil {
		t.Fatal("native fixture startup failed:", err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "80/tcp")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + host + ":" + port.Port()
	if name == "ntfy" {
		exit, output, err := container.Exec(ctx, []string{"ntfy", "--version"})
		if err != nil || exit != 0 {
			t.Fatal("ntfy version check failed")
		}
		data, err := io.ReadAll(io.LimitReader(output, 4096))
		if err != nil || !strings.Contains(string(data), versions[name].Version) {
			t.Fatal("unexpected ntfy version")
		}
	} else {
		var version struct {
			Version string `json:"version"`
		}
		api(t, http.MethodGet, endpoint+"/version", nil, "", &version)
		if strings.TrimPrefix(version.Version, "v") != versions[name].Version {
			t.Fatal("unexpected Gotify version")
		}
	}
	fixturePool.servers[name] = endpoint
	return endpoint
}

// api performs bounded fixture setup/inspection outside the driver's own request path.
func api(t *testing.T, method, url string, body io.Reader, token string, out any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("X-Gotify-Key", token)
	} else {
		request.SetBasicAuth("fixture", "fixture-password-disposable")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatal("fixture API rejected request:", response.StatusCode)
	}
	if out != nil && json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(out) != nil {
		t.Fatal("invalid fixture response")
	}
}

// TestNtfy verifies native acknowledgements, integer priorities and persisted subscription data.
func TestNtfy(t *testing.T) {
	endpoint := server(t, "ntfy")
	topic := fmt.Sprintf("fixture_%d", time.Now().UnixNano())
	driver, err := ntfy.New(ntfy.Config{URL: endpoint, Topic: topic, Priority: 4, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	notificationtest.Sender(t, func() notification.Sender { return driver })
	message := notification.Message{Title: "Native verification", Body: "integer priority"}
	if err := notification.New(driver).Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	var found bool
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(endpoint + "/" + topic + "/json?poll=1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(response.Body, 65536))
	for decoder.More() {
		var stored struct {
			Message  string `json:"message"`
			Priority int    `json:"priority"`
		}
		if err := decoder.Decode(&stored); err != nil {
			t.Fatal(err)
		}
		if stored.Message == message.Body {
			found = true
			if stored.Priority != 4 {
				t.Fatal("native priority mismatch")
			}
		}
	}
	if !found {
		t.Fatal("message missing from native subscription history")
	}
}

// TestGotify verifies native application authentication, storage and recipient isolation.
func TestGotify(t *testing.T) {
	endpoint := server(t, "gotify")
	var application struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	api(t, http.MethodPost, endpoint+"/application", strings.NewReader(`{"name":"notification-fixture"}`), "", &application)
	t.Cleanup(func() {
		api(t, http.MethodDelete, fmt.Sprintf("%s/application/%d", endpoint, application.ID), nil, "", nil)
	})
	driver, err := gotify.New(gotify.Config{URL: endpoint, Token: application.Token, Priority: 8, HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	notificationtest.Sender(t, func() notification.Sender { return driver })
	var history struct {
		Messages []struct {
			AppID    int    `json:"appid"`
			Priority int    `json:"priority"`
			Message  string `json:"message"`
		} `json:"messages"`
	}
	api(t, http.MethodGet, fmt.Sprintf("%s/application/%d/message", endpoint, application.ID), nil, "", &history)
	if len(history.Messages) == 0 {
		t.Fatal("native messages missing")
	}
	for _, message := range history.Messages {
		if message.AppID != application.ID || message.Priority != 8 || message.Message != "T-42 changed" {
			t.Fatal("native message projection mismatch")
		}
	}
	wrong, err := gotify.New(gotify.Config{URL: endpoint, Token: "invalid-disposable-token", HTTP: notification.HTTPConfig{AllowPrivate: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer wrong.Close()
	if err := wrong.Send(context.Background(), notification.Message{Body: "must reject"}); !errors.Is(err, notification.ErrUnauthorized) {
		t.Fatal("invalid application token was accepted")
	}
}
