package lib

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/html"
)

const mockMainPageHTML = `<html><body>
<table class="table table-bordered">
<tr class="table-primary"><th>Plan name</th><th>Actions</th></tr>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts CNFD-B</th></tr>
<tr>
  <th>env1 (<b>B-0</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env1'">Info</button>
    <button type="button" onclick="location.href='/exposeform/env1'">Reprovision</button>
    <button type="button" onclick="location.href='/sno/env1'">SNO</button>
  </th>
</tr>
<tr>
  <th>env2 (<b>B-1</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env2'">Info</button>
    <button type="button" onclick="location.href='/exposeform/env2'">Reprovision</button>
  </th>
</tr>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts CNFD-C</th></tr>
<tr>
  <th>env3 (<b>C-0</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env3'">Info</button>
  </th>
</tr>
</table></body></html>`

func TestListEnvironments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Expected path /, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockMainPageHTML))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	envs, err := client.ListEnvironments(context.Background())
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 3 {
		t.Fatalf("Expected 3 environments, got %d", len(envs))
	}

	if envs[0].Name != "env1" {
		t.Errorf("Expected first env name env1, got %s", envs[0].Name)
	}

	if envs[0].Group != "CNFD-B" {
		t.Errorf("Expected first env group CNFD-B, got %s", envs[0].Group)
	}

	if envs[2].Name != "env3" {
		t.Errorf("Expected third env name env3, got %s", envs[2].Name)
	}

	if envs[2].Group != "CNFD-C" {
		t.Errorf("Expected third env group CNFD-C, got %s", envs[2].Group)
	}
}

func TestListEnvironmentsNoDuplicates(t *testing.T) {
	html := `<html><body><table>
<tr><th>env1</th><th>
  <button onclick="location.href='/infoplan/env1'">Info</button>
  <button onclick="location.href='/infoplan/env1'">Info2</button>
</th></tr>
</table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	envs, err := client.ListEnvironments(context.Background())
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 1 {
		t.Errorf("Expected 1 environment (no duplicates), got %d", len(envs))
	}
}

func TestListEnvironmentsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><p>No environments</p></body></html>"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	envs, err := client.ListEnvironments(context.Background())
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 0 {
		t.Errorf("Expected 0 environments, got %d", len(envs))
	}
}

func TestListEnvironmentsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.ListEnvironments(context.Background())
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestGetAttrNotFound(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "button",
		Attr: []html.Attribute{
			{Key: "type", Val: "button"},
		},
	}

	result := getAttr(node, "onclick")
	if result != "" {
		t.Errorf("Expected empty string for missing attr, got %s", result)
	}
}

func TestHasAttrTrue(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "th",
		Attr: []html.Attribute{
			{Key: "colspan", Val: "3"},
		},
	}

	if !hasAttr(node, "colspan") {
		t.Error("Expected hasAttr to return true for colspan")
	}
}

func TestHasAttrFalse(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "th",
	}

	if hasAttr(node, "colspan") {
		t.Error("Expected hasAttr to return false for missing attr")
	}
}

func TestListEnvironmentsWithInfo(t *testing.T) {
	mainPage := `<html><body><table>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts TEST</th></tr>
<tr><th>env1</th><th>
  <button onclick="location.href='/infoplan/env1'">Info</button>
</th></tr>
<tr><th>env2</th><th>
  <button onclick="location.href='/infoplan/env2'">Info</button>
</th></tr>
</table></body></html>`

	env1Info := `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>env1</td><td>client1</td><td>2026-05-27 12:00 testowner</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>env1-installer</td><td>up</td><td>192.168.1.100</td></tr>
<tr><td>env1-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Write([]byte(mainPage))
		case "/infoplan/env1":
			w.Write([]byte(env1Info))
		case "/infoplan/env2":
			w.Write([]byte("<html><body><table></table></body></html>"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	details, err := client.ListEnvironmentsWithInfo(context.Background(), 5, nil, io.Discard)
	if err != nil {
		t.Fatalf("ListEnvironmentsWithInfo failed: %v", err)
	}

	if len(details) != 2 {
		t.Fatalf("Expected 2 environments, got %d", len(details))
	}

	if details[0].Status != "active" {
		t.Errorf("Expected env1 status active, got %s", details[0].Status)
	}

	if details[0].NodesUp != 2 {
		t.Errorf("Expected env1 nodes_up 2, got %d", details[0].NodesUp)
	}

	if details[0].Owner != "testowner" {
		t.Errorf("Expected env1 owner testowner, got %s", details[0].Owner)
	}

	if details[0].InstallerIP != "192.168.1.100" {
		t.Errorf("Expected env1 installer IP 192.168.1.100, got %s", details[0].InstallerIP)
	}

	if details[1].Status != "empty" {
		t.Errorf("Expected env2 status empty, got %s", details[1].Status)
	}

	if details[1].NodeCount != 0 {
		t.Errorf("Expected env2 node_count 0, got %d", details[1].NodeCount)
	}
}

func TestListEnvironmentsBodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, oversizedBody())
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.ListEnvironments(context.Background())
	assertTruncated(t, err)
}

func envListHTML(names ...string) string {
	var b strings.Builder
	b.WriteString(`<html><body><table>`)
	b.WriteString(`<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts TEST</th></tr>`)

	for _, name := range names {
		fmt.Fprintf(&b, `<tr><th>%s</th><th><button onclick="location.href='/infoplan/%s'">Info</button></th></tr>`, name, name)
	}

	b.WriteString(`</table></body></html>`)

	return b.String()
}

func numberedEnvNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("env%d", i+1)
	}

	return names
}

func TestListEnvironmentsWithInfoConcurrentProgress(t *testing.T) {
	names := numberedEnvNames(12)
	infoHTML := `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>plan</td><td>client</td><td>2026-05-27 12:00 owner</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>installer</td><td>up</td><td>192.168.1.100</td></tr>
</table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte(envListHTML(names...)))

			return
		}

		w.Write([]byte(infoHTML))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	var buf bytes.Buffer
	details, err := client.ListEnvironmentsWithInfo(context.Background(), 10, nil, &buf)
	if err != nil {
		t.Fatalf("ListEnvironmentsWithInfo failed: %v", err)
	}

	if len(details) != 12 {
		t.Fatalf("Expected 12 environments, got %d", len(details))
	}
}

func TestListEnvironmentsWithInfoCancelsLeftoverFetches(t *testing.T) {
	names := numberedEnvNames(8)
	var infoplanHits atomic.Int32
	started := make(chan struct{}, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte(envListHTML(names...)))

			return
		}

		infoplanHits.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}

		time.Sleep(300 * time.Millisecond)
		w.Write([]byte("<html><body><table></table></body></html>"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		_, err := client.ListEnvironmentsWithInfo(ctx, 2, nil, io.Discard)
		errCh <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Until(deadline)):
			t.Fatal("timed out waiting for infoplan workers to start")
		}
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ListEnvironmentsWithInfo returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListEnvironmentsWithInfo did not return after cancel")
	}

	if hits := infoplanHits.Load(); hits >= int32(len(names)) {
		t.Fatalf("expected leftover GetInfoPlan calls to skip, got %d hits", hits)
	}
}

type panicWriter struct{}

func (panicWriter) Write(p []byte) (int, error) {
	if string(p) == "\n" {
		return 1, nil
	}

	panic("boom")
}

func TestListEnvironmentsWithInfoRecoversWriterPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte(envListHTML("env1")))

			return
		}

		w.Write([]byte("<html><body><table></table></body></html>"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	done := make(chan error, 1)

	go func() {
		_, err := client.ListEnvironmentsWithInfo(context.Background(), 1, nil, panicWriter{})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ListEnvironmentsWithInfo returned error after panic: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ListEnvironmentsWithInfo hung after worker panic")
	}
}
