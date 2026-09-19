package pluginsystem

import "testing"

func TestDefaultWorkerLimit(t *testing.T) {
	for _, test := range []struct {
		name     string
		cpuCount int
		want     int
	}{
		{name: "covers routing lane budget", cpuCount: 1, want: 8},
		{name: "uses larger cpu count", cpuCount: 16, want: 16},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := defaultWorkerLimit(test.cpuCount); got != test.want {
				t.Fatalf("default worker limit = %d, want %d", got, test.want)
			}
		})
	}
}
