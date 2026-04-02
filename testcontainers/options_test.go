package httptape

import (
	"strings"
	"testing"
)

func TestValidate_MutuallyExclusiveConfig(t *testing.T) {
	o := options{
		image:       DefaultImage,
		port:        DefaultPort,
		mode:        ModeServe,
		fixturesDir: "/tmp/fixtures",
		configFile:  "/tmp/config.json",
		configJSON:  []byte(`{"version":"1"}`),
	}
	err := o.validate()
	if err == nil {
		t.Fatal("expected error for mutually exclusive config options")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_RecordModeRequiresTarget(t *testing.T) {
	o := options{
		image:       DefaultImage,
		port:        DefaultPort,
		mode:        ModeRecord,
		fixturesDir: "/tmp/fixtures",
	}
	err := o.validate()
	if err == nil {
		t.Fatal("expected error for record mode without target")
	}
	if !strings.Contains(err.Error(), "WithTarget") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_ServeModeRequiresFixtures(t *testing.T) {
	o := options{
		image: DefaultImage,
		port:  DefaultPort,
		mode:  ModeServe,
	}
	err := o.validate()
	if err == nil {
		t.Fatal("expected error for serve mode without fixtures")
	}
	if !strings.Contains(err.Error(), "WithFixturesDir") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_UnknownMode(t *testing.T) {
	o := options{
		image:       DefaultImage,
		port:        DefaultPort,
		mode:        "unknown",
		fixturesDir: "/tmp/fixtures",
	}
	err := o.validate()
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_ValidServeMode(t *testing.T) {
	o := options{
		image:       DefaultImage,
		port:        DefaultPort,
		mode:        ModeServe,
		fixturesDir: "/tmp/fixtures",
	}
	if err := o.validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_ValidRecordMode(t *testing.T) {
	o := options{
		image:       DefaultImage,
		port:        DefaultPort,
		mode:        ModeRecord,
		fixturesDir: "/tmp/fixtures",
		target:      "http://example.com",
	}
	if err := o.validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"8081/tcp", "8081"},
		{"9090/tcp", "9090"},
		{"443/tcp", "443"},
		{"8081", "8081"},
	}
	for _, tt := range tests {
		got := extractPort(tt.input)
		if got != tt.want {
			t.Errorf("extractPort(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildCmd_ServeMode(t *testing.T) {
	o := options{
		mode:        ModeServe,
		port:        "8081/tcp",
		fixturesDir: "/tmp/fixtures",
	}
	cmd := buildCmd(o)
	if cmd[0] != "serve" {
		t.Errorf("expected first arg to be 'serve', got %q", cmd[0])
	}
	assertContains(t, cmd, "--port", "8081")
	assertContains(t, cmd, "--fixtures", "/fixtures")
}

func TestBuildCmd_RecordMode(t *testing.T) {
	o := options{
		mode:        ModeRecord,
		port:        "8081/tcp",
		fixturesDir: "/tmp/fixtures",
		target:      "http://example.com",
		configFile:  "/tmp/config.json",
	}
	cmd := buildCmd(o)
	if cmd[0] != "record" {
		t.Errorf("expected first arg to be 'record', got %q", cmd[0])
	}
	assertContains(t, cmd, "--upstream", "http://example.com")
	assertContains(t, cmd, "--config", "/config/config.json")
}

func TestFunctionalOptions(t *testing.T) {
	o := options{}

	WithFixturesDir("/data")(&o)
	WithConfigFile("/cfg.json")(&o)
	WithPort("9090/tcp")(&o)
	WithImage("myimage:v1")(&o)
	WithMode(ModeRecord)(&o)
	WithTarget("http://upstream.local")(&o)

	if o.fixturesDir != "/data" {
		t.Errorf("fixturesDir = %q, want /data", o.fixturesDir)
	}
	if o.configFile != "/cfg.json" {
		t.Errorf("configFile = %q, want /cfg.json", o.configFile)
	}
	if o.port != "9090/tcp" {
		t.Errorf("port = %q, want 9090/tcp", o.port)
	}
	if o.image != "myimage:v1" {
		t.Errorf("image = %q, want myimage:v1", o.image)
	}
	if o.mode != ModeRecord {
		t.Errorf("mode = %q, want record", o.mode)
	}
	if o.target != "http://upstream.local" {
		t.Errorf("target = %q, want http://upstream.local", o.target)
	}
}

func TestWithConfig(t *testing.T) {
	o := options{}
	cfg := struct {
		Version string `json:"version"`
	}{Version: "1"}
	WithConfig(cfg)(&o)
	if len(o.configJSON) == 0 {
		t.Fatal("expected configJSON to be set")
	}
	if !strings.Contains(string(o.configJSON), `"version":"1"`) {
		t.Errorf("unexpected configJSON: %s", o.configJSON)
	}
}

// assertContains checks that the slice contains a consecutive pair of values.
func assertContains(t *testing.T, slice []string, key, value string) {
	t.Helper()
	for i := 0; i < len(slice)-1; i++ {
		if slice[i] == key && slice[i+1] == value {
			return
		}
	}
	t.Errorf("slice %v does not contain consecutive pair (%q, %q)", slice, key, value)
}
