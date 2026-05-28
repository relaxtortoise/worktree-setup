package selfupdate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failReadCloser satisfies io.ReadCloser and always returns an error on Read.
type failReadCloser struct{}

func (failReadCloser) Read(p []byte) (int, error) { return 0, errors.New("read failed") }
func (failReadCloser) Close() error               { return nil }

// mockHTTPResponse creates an HTTP response for testing
func mockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

// TestMain dispatches subprocess tests for cases that need os.Exit or
// file-system side effects that should not touch the original test binary.
func TestMain(m *testing.M) {
	switch os.Getenv("WT_SUBCMD") {
	case "check-mode":
		DoHTTPGet = func(url string) (*http.Response, error) {
			return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
		}
		u := New("1.0.0", "owner", "repo")
		_ = u.Run(nil, true, true)
		return

	case "full-update":
		var fuCallCount int
		DoHTTPGet = func(url string) (*http.Response, error) {
			fuCallCount++
			if fuCallCount == 1 {
				return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
			}
			return mockHTTPResponse(200, "fake binary data for testing purposes"), nil
		}
		u := New("1.0.0", "owner", "repo")
		if err := u.Run([]string{"v2.0.0"}, true, false); err != nil {
			fmt.Println("FULL_UPDATE_ERROR:", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "download-http-error":
		callCount := 0
		DoHTTPGet = func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
			}
			return mockHTTPResponse(500, "internal server error"), nil
		}
		u := New("1.0.0", "owner", "repo")
		if err := u.Run([]string{"v2.0.0"}, true, false); err != nil {
			fmt.Println("DOWNLOAD_ERROR:", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "download-network-error":
		callCount := 0
		DoHTTPGet = func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
			}
			return nil, errors.New("connection refused")
		}
		u := New("1.0.0", "owner", "repo")
		if err := u.Run([]string{"v2.0.0"}, true, false); err != nil {
			fmt.Println("DOWNLOAD_ERROR:", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "download-read-error":
		// Body returns an error on Read, exercising the io.Copy error path.
		var dreCallCount int
		DoHTTPGet = func(url string) (*http.Response, error) {
			dreCallCount++
			if dreCallCount == 1 {
				return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
			}
			return &http.Response{StatusCode: 200, Body: failReadCloser{}}, nil
		}
		u := New("1.0.0", "owner", "repo")
		if err := u.Run([]string{"v2.0.0"}, true, false); err != nil {
			fmt.Println("DOWNLOAD_READ_ERROR:", err)
			os.Exit(0)
		}
		os.Exit(1)

	case "create-temp-error":
		// os.CreateTemp fails because TMPDIR points to a non-existent directory.
		var cteCallCount int
		DoHTTPGet = func(url string) (*http.Response, error) {
			cteCallCount++
			if cteCallCount == 1 {
				return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
			}
			return mockHTTPResponse(200, "fake binary"), nil
		}
		u := New("1.0.0", "owner", "repo")
		if err := u.Run([]string{"v2.0.0"}, true, false); err != nil {
			fmt.Println("CREATE_TEMP_ERROR:", err)
			os.Exit(0)
		}
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// filterEnv returns the environment without the given key prefix.
func filterEnv(prefix string) []string {
	var out []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// runSubprocess copies the test binary to a temp dir and runs a TestMain
// sub-command.  This keeps file-system side-effects (e.g. os.Rename) inside
// the temporary copy instead of touching the original test binary.
func runSubprocess(t *testing.T, subCmd, tempDir string) ([]byte, error) {
	t.Helper()

	testBin, err := os.Executable()
	require.NoError(t, err)

	tempBin := filepath.Join(tempDir, "selfupdate.test")
	src, err := os.ReadFile(testBin)
	require.NoError(t, err)
	err = os.WriteFile(tempBin, src, 0755)
	require.NoError(t, err)

	cmd := exec.Command(tempBin, "-test.run=^TestShouldNotBeCalled$")
	cmd.Env = filterEnv("TMPDIR=")
	cmd.Env = append(cmd.Env, "WT_SUBCMD="+subCmd)
	cmd.Dir = tempDir

	return cmd.CombinedOutput()
}

// runSubprocessWithEnv is like runSubprocess but allows setting extra
// environment variables.
func runSubprocessWithEnv(t *testing.T, subCmd, tempDir string, extraEnv ...string) ([]byte, error) {
	t.Helper()

	testBin, err := os.Executable()
	require.NoError(t, err)

	tempBin := filepath.Join(tempDir, "selfupdate.test")
	src, err := os.ReadFile(testBin)
	require.NoError(t, err)
	err = os.WriteFile(tempBin, src, 0755)
	require.NoError(t, err)

	cmd := exec.Command(tempBin, "-test.run=^TestShouldNotBeCalled$")
	cmd.Env = filterEnv("TMPDIR=")
	cmd.Env = append(cmd.Env, "WT_SUBCMD="+subCmd)
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.Dir = tempDir

	return cmd.CombinedOutput()
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	u := New("1.0.0", "owner", "repo")
	assert.Equal(t, "1.0.0", u.Version)
	assert.Equal(t, "owner", u.Owner)
	assert.Equal(t, "repo", u.Repo)
}

// ---------------------------------------------------------------------------
// Run – "already up to date" (same version)
// ---------------------------------------------------------------------------

func TestRun_AlreadyUpToDate(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, `{"tag_name":"v1.0.0"}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	err := u.Run(nil, true, false)
	require.NoError(t, err)
}

func TestRun_AlreadyUpToDateWithExplicitTag(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, `{"tag_name":"v1.0.0"}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	err := u.Run([]string{"v1.0.0"}, true, false)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Run – version not found
// ---------------------------------------------------------------------------

func TestRun_VersionNotFound(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	err := u.Run([]string{"v99.0.0"}, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRun_VersionNoVPrefix(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	err := u.Run([]string{"99.0.0"}, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Run – latest release not found
// ---------------------------------------------------------------------------

func TestRun_LatestReleaseNotFound(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	err := u.Run(nil, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "latest release")
}

// ---------------------------------------------------------------------------
// Run – check mode (--check) – via subprocess because of os.Exit(1)
// ---------------------------------------------------------------------------

func TestRun_CheckMode(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestShouldNotBeCalled$")
	cmd.Env = append(os.Environ(), "WT_SUBCMD=check-mode")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(out), "An update is available.")
	assert.Contains(t, string(out), "Current version: v1.0.0")
	assert.Contains(t, string(out), "Latest version:  v2.0.0")
}

// ---------------------------------------------------------------------------
// Run – interactive prompt (!yesFlag)
// ---------------------------------------------------------------------------

func TestRun_InteractiveCancel(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	err = u.Run([]string{"v2.0.0"}, false, false)
	require.NoError(t, err)
}

func TestRun_InteractiveYesThenDownloadFails(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	_, _ = w.WriteString("y\n")
	_ = w.Close()

	callCount := 0
	DoHTTPGet = func(url string) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
		}
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	err = u.Run([]string{"v2.0.0"}, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

// ---------------------------------------------------------------------------
// Run – download / file operations (via subprocess on a temp binary copy)
// ---------------------------------------------------------------------------

func TestRun_FullUpdate(t *testing.T) {
	out, err := runSubprocess(t, "full-update", t.TempDir())
	require.NoError(t, err, "subprocess failed: %s", string(out))
	assert.Contains(t, string(out), "Updated to v2.0.0")
}

func TestRun_DownloadHTTPError(t *testing.T) {
	out, err := runSubprocess(t, "download-http-error", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, string(out), "DOWNLOAD_ERROR:")
	assert.Contains(t, string(out), "HTTP 500")
}

func TestRun_DownloadNetworkError(t *testing.T) {
	out, err := runSubprocess(t, "download-network-error", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, string(out), "DOWNLOAD_ERROR:")
	assert.Contains(t, string(out), "check your network")
}

func TestRun_DownloadReadError(t *testing.T) {
	out, err := runSubprocess(t, "download-read-error", t.TempDir())
	require.NoError(t, err)
	assert.Contains(t, string(out), "DOWNLOAD_READ_ERROR:")
	assert.Contains(t, string(out), "read failed")
}

func TestRun_CreateTempError(t *testing.T) {
	out, err := runSubprocessWithEnv(t, "create-temp-error", t.TempDir(),
		"TMPDIR=/nonexistent-path-for-testing")
	require.NoError(t, err)
	assert.Contains(t, string(out), "CREATE_TEMP_ERROR:")
	assert.Contains(t, string(out), "temp file")
}

// ---------------------------------------------------------------------------
// getLatestRelease
// ---------------------------------------------------------------------------

func TestGetLatestRelease_Success(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		assert.Contains(t, url, "releases/latest")
		return mockHTTPResponse(200, `{"tag_name":"v3.0.0"}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	release, err := u.getLatestRelease()
	require.NoError(t, err)
	assert.Equal(t, "v3.0.0", release.TagName)
}

func TestGetLatestRelease_NotOK(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getLatestRelease()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no releases found")
}

func TestGetLatestRelease_NetworkError(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getLatestRelease()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestGetLatestRelease_JSONDecodeError(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, `{invalid json}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getLatestRelease()
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// getReleaseByTag
// ---------------------------------------------------------------------------

func TestGetReleaseByTag_Success(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		assert.Contains(t, url, "releases/tags/v2.0.0")
		return mockHTTPResponse(200, `{"tag_name":"v2.0.0"}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	release, err := u.getReleaseByTag("v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", release.TagName)
}

func TestGetReleaseByTag_NotFound(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(404, ""), nil
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getReleaseByTag("v99.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release not found for tag v99.0.0")
}

func TestGetReleaseByTag_NetworkError(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return nil, errors.New("timeout")
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getReleaseByTag("v2.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestGetReleaseByTag_JSONDecodeError(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, `{bad}`), nil
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getReleaseByTag("v2.0.0")
	require.Error(t, err)
}

func TestGetReleaseByTag_JSONDecodeErrorEmpty(t *testing.T) {
	oldHTTP := DoHTTPGet
	defer func() { DoHTTPGet = oldHTTP }()

	DoHTTPGet = func(url string) (*http.Response, error) {
		return mockHTTPResponse(200, ``), nil
	}

	u := New("1.0.0", "owner", "repo")
	_, err := u.getReleaseByTag("v2.0.0")
	require.Error(t, err)
}
