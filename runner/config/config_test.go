package config

import "testing"

func TestControllerBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		runner  RunnerIE
		want    string
		wantErr bool
	}{
		{"ip and port", RunnerIE{ControllerIP: "10.8.0.3", ControllerPort: 8888}, "http://10.8.0.3:8888", false},
		{"ipv6", RunnerIE{ControllerIP: "fd00::3", ControllerPort: 8888}, "http://[fd00::3]:8888", false},
		{"url wins", RunnerIE{ControllerIP: "10.8.0.3", ControllerPort: 8888, ControllerURL: "https://it.example.org"}, "https://it.example.org", false},
		{"url with port and slash", RunnerIE{ControllerURL: "https://it.example.org:8443/"}, "https://it.example.org:8443", false},
		{"plain http url", RunnerIE{ControllerURL: "http://10.8.0.3:8888"}, "http://10.8.0.3:8888", false},
		{"no scheme", RunnerIE{ControllerURL: "it.example.org"}, "", true},
		{"other scheme", RunnerIE{ControllerURL: "ftp://it.example.org"}, "", true},
		{"path", RunnerIE{ControllerURL: "https://it.example.org/api"}, "", true},
		{"credentials", RunnerIE{ControllerURL: "https://user:pass@it.example.org"}, "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.runner.ControllerBaseURL()
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
