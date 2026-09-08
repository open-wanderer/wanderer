package srch0

// TestingT is the small test-harness boundary shared by the Go adapters. The
// package is imported only from tests and does not depend on testing itself.
type TestingT interface {
	Helper()
	Fatal(...any)
	Fatalf(string, ...any)
	Error(...any)
	Log(...any)
	Logf(string, ...any)
	TempDir() string
	Cleanup(func())
	Setenv(string, string)
}
