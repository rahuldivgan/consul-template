package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func testConfig(contents string, t *testing.T) *Config {
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte(contents))
	if err != nil {
		t.Fatal(err)
	}

	config, err := ParseConfig(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestMerge_emptyConfig(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{})

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_topLevel(t *testing.T) {
	config1 := testConfig(`
		consul = "consul-1"
		token = "token-1"
		max_stale = "1s"
		retry = "1s"
		wait = "1s"
		log_level = "log_level-1"
	`, t)
	config2 := testConfig(`
		consul = "consul-2"
		token = "token-2"
		max_stale = "2s"
		retry = "2s"
		wait = "2s"
		log_level = "log_level-2"
	`, t)
	config1.Merge(config2)

	if !reflect.DeepEqual(config1, config2) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config1, config2)
	}
}

func TestMerge_vault(t *testing.T) {
	config := testConfig(`
		vault {
			address = "1.1.1.1"
			token = "1"
		}
	`, t)
	config.Merge(testConfig(`
		vault {
			address = "2.2.2.2"
		}
	`, t))

	expected := &VaultConfig{
		Address: "2.2.2.2",
		Token:   "1",
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  true,
			Cert:    "",
			CaCert:  "",
		},
	}

	if !reflect.DeepEqual(config.Vault, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault, expected)
	}
}

func TestMerge_vaultSSL(t *testing.T) {
	config := testConfig(`
		vault {
			ssl {
				enabled = true
				verify = true
				cert = "1.pem"
				ca_cert = "ca-1.pem"
			}
		}
	`, t)
	config.Merge(testConfig(`
		vault {
			ssl {
				enabled = false
			}
		}
	`, t))

	expected := &VaultConfig{
		SSL: &SSLConfig{
			Enabled: false,
			Verify:  true,
			Cert:    "1.pem",
			CaCert:  "ca-1.pem",
		},
	}

	if !reflect.DeepEqual(config.Vault, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Vault, expected)
	}
}

func TestMerge_auth(t *testing.T) {
	config := testConfig(`
		auth {
			enabled = true
			username = "1"
			password = "1"
		}
	`, t)
	config.Merge(testConfig(`
		auth {
			password = "2"
		}
	`, t))

	expected := &AuthConfig{
		Enabled:  true,
		Username: "1",
		Password: "2",
	}

	if !reflect.DeepEqual(config.Auth, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Auth, expected)
	}
}

func TestMerge_SSL(t *testing.T) {
	config := testConfig(`
		ssl {
			enabled = true
			verify = true
			cert = "1.pem"
			ca_cert = "ca-1.pem"
		}
	`, t)
	config.Merge(testConfig(`
		ssl {
			enabled = false
		}
	`, t))

	expected := &SSLConfig{
		Enabled: false,
		Verify:  true,
		Cert:    "1.pem",
		CaCert:  "ca-1.pem",
	}

	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestMerge_syslog(t *testing.T) {
	config := testConfig(`
		syslog {
			enabled = true
			facility = "1"
		}
	`, t)
	config.Merge(testConfig(`
		syslog {
			facility = "2"
		}
	`, t))

	expected := &SyslogConfig{
		Enabled:  true,
		Facility: "2",
	}

	if !reflect.DeepEqual(config.Syslog, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Syslog, expected)
	}
}

func TestMerge_configTemplates(t *testing.T) {
	config := testConfig(`
		template {
			source = "1"
			destination = "1"
			command = "1"
		}
	`, t)
	config.Merge(testConfig(`
		template {
			source = "2"
			destination = "2"
			command = "2"
		}
	`, t))

	expected := []*ConfigTemplate{
		&ConfigTemplate{
			Source:      "1",
			Destination: "1",
			Command:     "1",
		},
		&ConfigTemplate{
			Source:      "2",
			Destination: "2",
			Command:     "2",
		},
	}

	if !reflect.DeepEqual(config.ConfigTemplates, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.ConfigTemplates, expected)
	}
}

func TestMerge_wait(t *testing.T) {
	config := testConfig(`
		wait = "1s:1s"
	`, t)
	config.Merge(testConfig(`
		wait = "2s:2s"
	`, t))

	expected := &watch.Wait{
		Min: 2 * time.Second,
		Max: 2 * time.Second,
	}

	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Wait, expected)
	}
}

func TestParseConfig_readFileError(t *testing.T) {
	_, err := ParseConfig(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "no such file or directory"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_parseFileError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    invalid file in here
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "syntax error"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		consul = "nyc1.demo.consul.io"
		max_stale = "5s"
		token = "abcd1234"
		wait = "5s:10s"
		retry = "10s"
		log_level = "warn"

		vault {
			address = "vault.service.consul"
			token = "efgh5678"
			ssl {
				enabled = false
			}
		}

		auth {
			enabled = true
			username = "test"
			password = "test"
		}

		ssl {
			enabled = true
			verify = false
			cert = "c1.pem"
			ca_cert = "c2.pem"
		}

		syslog {
			enabled = true
			facility = "LOCAL5"
		}

		template {
			source = "nginx.conf.ctmpl"
			destination  = "/etc/nginx/nginx.conf"
		}

		template {
			source = "redis.conf.ctmpl"
			destination  = "/etc/redis/redis.conf"
			command = "service redis restart"
		}
  `), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:     configFile.Name(),
		Consul:   "nyc1.demo.consul.io",
		MaxStale: time.Second * 5,
		Vault: &VaultConfig{
			Address: "vault.service.consul",
			Token:   "efgh5678",
			SSL: &SSLConfig{
				Enabled: false,
				Verify:  true,
				Cert:    "",
				CaCert:  "",
			},
		},
		Auth: &AuthConfig{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  false,
			Cert:    "c1.pem",
			CaCert:  "c2.pem",
		},
		Syslog: &SyslogConfig{
			Enabled:  true,
			Facility: "LOCAL5",
		},
		Token: "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		Retry:    10 * time.Second,
		LogLevel: "warn",
		ConfigTemplates: []*ConfigTemplate{
			&ConfigTemplate{
				Source:      "nginx.conf.ctmpl",
				Destination: "/etc/nginx/nginx.conf",
			},
			&ConfigTemplate{
				Source:      "redis.conf.ctmpl",
				Destination: "/etc/redis/redis.conf",
				Command:     "service redis restart",
			},
		},
		setKeys: config.setKeys,
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	}
}

func TestParseConfig_mapstructureError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = true
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "unconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_extraKeys(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		fake_key = "nope"
		another_fake_key = "never"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error")
	}

	expected := "invalid keys: another_fake_key, fake_key"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseConfig_parseMaxStaleError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    max_stale = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_parseRetryError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    retry = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_parseWaitError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &AuthConfig{Enabled: false}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestParseConfigTemplate_emptyStringArgs(t *testing.T) {
	_, err := ParseConfigTemplate("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigTemplate_stringWithSpacesArgs(t *testing.T) {
	_, err := ParseConfigTemplate("  ")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "cannot specify empty template declaration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigurationTemplate_tooManyArgs(t *testing.T) {
	_, err := ParseConfigTemplate("foo:bar:blitz:baz")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid template declaration format"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfigurationTemplate_windowsDrives(t *testing.T) {
	ct, err := ParseConfigTemplate(`C:\abc\123:D:\xyz\789:some command`)
	if err != nil {
		t.Fatalf("failed parsing windows drive letters: %s", err)
	}

	expected := &ConfigTemplate{
		Source:      `C:\abc\123`,
		Destination: `D:\xyz\789`,
		Command:     "some command",
	}

	if !reflect.DeepEqual(ct, expected) {
		t.Fatalf("unexpected result parsing windows drives: %#v", ct)
	}
}

func TestParseConfigurationTemplate_source(t *testing.T) {
	source := "/tmp/config.ctmpl"
	template, err := ParseConfigTemplate(source)
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}
}

func TestParseConfigurationTemplate_destination(t *testing.T) {
	source, destination := "/tmp/config.ctmpl", "/tmp/out"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s", source, destination))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}
}

func TestParseConfigurationTemplate_command(t *testing.T) {
	source, destination, command := "/tmp/config.ctmpl", "/tmp/out", "reboot"
	template, err := ParseConfigTemplate(fmt.Sprintf("%s:%s:%s", source, destination, command))
	if err != nil {
		t.Fatal(err)
	}

	if template.Source != source {
		t.Errorf("expected %q to equal %q", template.Source, source)
	}

	if template.Destination != destination {
		t.Errorf("expected %q to equal %q", template.Destination, destination)
	}

	if template.Command != command {
		t.Errorf("expected %q to equal %q", template.Command, command)
	}
}
