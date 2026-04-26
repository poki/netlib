package features

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	cdpRuntime "github.com/chromedp/cdproto/runtime"
	messages "github.com/cucumber/messages/go/v21"
)

const (
	defaultActionTimeout = 30 * time.Second
	waitEventTimeout     = 20 * time.Second
)

type browserFeatureSuite struct {
	rootDir string
	tmpDir  string
	workDir string

	signalingBinary string
	testproxyBinary string

	harnessBundle string
	harnessServer *httptest.Server
	harnessURL    string

	allocCancel   context.CancelFunc
	browserCtx    context.Context
	browserCancel context.CancelFunc
}

func TestMain(m *testing.M) {
	if os.Getenv("SKIP_FEATURE_TESTS") == "true" {
		os.Exit(m.Run())
	}

	opts := godog.Options{
		Format:    "pretty",
		Paths:     []string{"."},
		Output:    colors.Colored(os.Stdout),
		Randomize: -1,
		Strict:    true,
	}
	godog.BindCommandLineFlags("godog.", &opts)
	flag.Parse()
	if len(flag.Args()) > 0 {
		opts.Paths = flag.Args()
	}

	suite, err := newBrowserFeatureSuite()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	world := newScenarioWorld(suite)
	testSuite := godog.TestSuite{
		Name:                "netlib",
		ScenarioInitializer: world.InitializeScenario,
		Options:             &opts,
	}

	status := testSuite.Run()
	suite.Close()
	os.Exit(status)
}

func newBrowserFeatureSuite() (*browserFeatureSuite, error) {
	rootDir, err := findRepoRoot()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "netlib-browser-features-*")
	if err != nil {
		return nil, err
	}

	suite := &browserFeatureSuite{
		rootDir: rootDir,
		tmpDir:  tmpDir,
	}

	if err := suite.buildBackendBinaries(); err != nil {
		suite.Close()
		return nil, err
	}
	if err := suite.buildHarness(); err != nil {
		suite.Close()
		return nil, err
	}
	if err := suite.startHarnessServer(); err != nil {
		suite.Close()
		return nil, err
	}
	if err := suite.startChrome(); err != nil {
		suite.Close()
		return nil, err
	}

	return suite, nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find go.mod")
		}
		dir = parent
	}
}

func (s *browserFeatureSuite) buildBackendBinaries() error {
	binaries := map[string]*string{
		"signaling": &s.signalingBinary,
		"testproxy": &s.testproxyBinary,
	}
	for backend, target := range binaries {
		output := filepath.Join(s.tmpDir, "netlib-cucumber-"+backend)
		cmd := exec.Command("go", "build", "-o", output, "./cmd/"+backend)
		cmd.Dir = s.rootDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build %s: %w\n%s", backend, err, string(out))
		}
		*target = output
	}
	return nil
}

func (s *browserFeatureSuite) buildHarness() error {
	distDir := filepath.Join(s.tmpDir, "harness")
	workDir, err := os.MkdirTemp(s.tmpDir, "work-")
	if err != nil {
		return err
	}
	s.workDir = workDir

	repoLink := filepath.Join(workDir, "repo")
	if err := os.Symlink(s.rootDir, repoLink); err != nil {
		return err
	}
	harnessSource := "./repo/features/browser-harness.ts"

	if err := os.WriteFile(filepath.Join(workDir, "package.json"), []byte(`{"name":"netlib-browser-features","private":true}`+"\n"), 0o644); err != nil {
		return err
	}
	indexHTML := `<!doctype html>
<body>
<script>
window.netlibTestLoadErrors = []
window.addEventListener('error', event => {
  window.netlibTestLoadErrors.push(event.message || String(event.error || event))
})
window.addEventListener('unhandledrejection', event => {
  const reason = event.reason
  window.netlibTestLoadErrors.push(String(reason && (reason.stack || reason.message) || reason))
})
</script>
<script type="module" src="./harness.ts"></script>
</body>
` + "\n"
	if err := os.WriteFile(filepath.Join(workDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		return err
	}
	harnessEntrypoint := "import { browserHarnessLoaded } from '" + harnessSource + "'\n" +
		";(window as any).netlibBrowserHarnessLoaded = browserHarnessLoaded\n"
	if err := os.WriteFile(filepath.Join(workDir, "harness.ts"), []byte(harnessEntrypoint), 0o644); err != nil {
		return err
	}

	cacheDir := filepath.Join(s.tmpDir, "parcel-cache")
	parcel := filepath.Join(s.rootDir, "node_modules", ".bin", "parcel")
	cmd := exec.Command(
		parcel, "build", "index.html",
		"--dist-dir", distDir,
		"--cache-dir", cacheDir,
		"--no-content-hash",
		"--no-source-maps",
		"--no-optimize",
		"--no-cache",
		"--log-level", "error",
	)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NODE_ENV=test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build browser harness: %w\n%s", err, string(out))
	}

	entries, err := os.ReadDir(distDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		if s.harnessBundle != "" {
			return fmt.Errorf("expected one harness bundle, found at least %s and %s", filepath.Base(s.harnessBundle), entry.Name())
		}
		s.harnessBundle = filepath.Join(distDir, entry.Name())
	}
	if s.harnessBundle == "" {
		return fmt.Errorf("browser harness build did not produce an HTML bundle in %s", distDir)
	}
	return nil
}

func (s *browserFeatureSuite) startHarnessServer() error {
	distDir := filepath.Dir(s.harnessBundle)
	files := http.FileServer(http.Dir(distDir))
	s.harnessServer = httptest.NewServer(files)
	s.harnessURL = s.harnessServer.URL + "/" + filepath.Base(s.harnessBundle)
	return nil
}

func (s *browserFeatureSuite) startChrome() error {
	chromePath, err := findChromeExecutable()
	if err != nil {
		return err
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromeExecAllocatorOptions(chromePath)...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	s.allocCancel = allocCancel
	s.browserCtx = browserCtx
	s.browserCancel = browserCancel

	return chromedp.Run(browserCtx)
}

func chromeExecAllocatorOptions(chromePath string) []chromedp.ExecAllocatorOption {
	disableFeatures := strings.Join([]string{
		"site-per-process",
		"Translate",
		"BlinkGenPropertyTrees",
		"WebRtcHideLocalIpsWithMdns",
	}, ",")

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.ExecPath(chromePath),
		chromedp.WindowSize(1280, 720),
		chromedp.Flag("disable-features", disableFeatures),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		chromedp.Flag("force-webrtc-ip-handling-policy", "default"),
		chromedp.Flag("remote-debugging-address", "127.0.0.1"),
	)
	return opts
}

func findChromeExecutable() (string, error) {
	if chromePath := strings.TrimSpace(os.Getenv("CHROME_PATH")); chromePath != "" {
		return resolveConfiguredChromePath("CHROME_PATH", chromePath)
	}

	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if executable, err := exec.LookPath(name); err == nil {
			return executable, nil
		}
	}

	if goruntime.GOOS == "darwin" {
		for _, root := range chromeApplicationRoots() {
			for _, appName := range []string{
				"Google Chrome.app",
				"Google Chrome for Testing.app",
				"Google Chrome Beta.app",
				"Google Chrome Dev.app",
				"Google Chrome Canary.app",
				"Chromium.app",
			} {
				if executable, ok := resolveChromePath(filepath.Join(root, appName)); ok {
					return executable, nil
				}
			}
		}
	}

	return "", errors.New("could not find Chrome; set CHROME_PATH to the Chrome executable or .app path")
}

func resolveConfiguredChromePath(envName, chromePath string) (string, error) {
	if executable, ok := resolveChromePath(chromePath); ok {
		return executable, nil
	}
	if !strings.ContainsRune(chromePath, filepath.Separator) {
		if resolved, err := exec.LookPath(chromePath); err == nil {
			if executable, ok := resolveChromePath(resolved); ok {
				return executable, nil
			}
		}
	}
	return "", fmt.Errorf("%s=%q does not point to an executable Chrome", envName, chromePath)
}

func chromeApplicationRoots() []string {
	roots := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, filepath.Join(home, "Applications"))
	}
	return roots
}

func resolveChromePath(chromePath string) (string, bool) {
	chromePath = strings.TrimSpace(chromePath)
	if chromePath == "" {
		return "", false
	}
	if strings.HasSuffix(chromePath, ".app") {
		return executableInAppBundle(chromePath)
	}

	info, err := os.Stat(chromePath)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return executableInAppBundle(chromePath)
	}
	if isExecutableFile(chromePath) {
		return chromePath, true
	}
	return "", false
}

func executableInAppBundle(appPath string) (string, bool) {
	macOSDir := filepath.Join(appPath, "Contents", "MacOS")
	baseName := strings.TrimSuffix(filepath.Base(appPath), ".app")
	candidates := []string{filepath.Join(macOSDir, baseName)}

	entries, err := os.ReadDir(macOSDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			candidate := filepath.Join(macOSDir, entry.Name())
			if candidate != candidates[0] {
				candidates = append(candidates, candidate)
			}
		}
	}

	for _, candidate := range candidates {
		if isExecutableFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

func (s *browserFeatureSuite) Close() {
	if s.browserCancel != nil {
		s.browserCancel()
	}
	if s.allocCancel != nil {
		s.allocCancel()
	}
	if s.harnessServer != nil {
		s.harnessServer.Close()
	}
	if s.tmpDir != "" {
		_ = os.RemoveAll(s.tmpDir)
	}
	if s.workDir != "" {
		_ = os.RemoveAll(s.workDir)
	}
}

type scenarioWorld struct {
	suite *browserFeatureSuite

	stepArg *messages.PickleStepArgument

	backends     map[string]*backendProcess
	signalingURL string
	testproxyURL string
	databaseURL  string

	useTestProxy bool
	players      map[string]*browserPlayer
	lastError    map[string]string
}

func newScenarioWorld(suite *browserFeatureSuite) *scenarioWorld {
	return &scenarioWorld{
		suite: suite,
	}
}

func (w *scenarioWorld) InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		w.reset()
		return ctx, nil
	})
	ctx.BeforeStep(func(st *godog.Step) {
		w.stepArg = st.Argument
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		return ctx, w.cleanup()
	})

	w.registerSteps(ctx)
}

func (w *scenarioWorld) reset() {
	w.stepArg = nil
	w.backends = make(map[string]*backendProcess)
	w.signalingURL = ""
	w.testproxyURL = ""
	w.databaseURL = ""
	w.useTestProxy = false
	w.players = make(map[string]*browserPlayer)
	w.lastError = make(map[string]string)
}

func (w *scenarioWorld) cleanup() error {
	var errs []error
	for _, player := range w.players {
		if err := player.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	w.players = make(map[string]*browserPlayer)

	keys := make([]string, 0, len(w.backends))
	for key := range w.backends {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := w.backends[key].Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
	}
	w.backends = make(map[string]*backendProcess)
	return errors.Join(errs...)
}

type browserPlayer struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *scenarioWorld) newBrowserPlayer(name string) (*browserPlayer, error) {
	tabCtx, cancel := chromedp.NewContext(w.suite.browserCtx)
	player := &browserPlayer{
		name:   name,
		ctx:    tabCtx,
		cancel: cancel,
	}

	chromedp.ListenTarget(tabCtx, func(ev any) {
		switch ev := ev.(type) {
		case *cdpRuntime.EventConsoleAPICalled:
			var parts []string
			for _, arg := range ev.Args {
				if len(arg.Value) > 0 {
					parts = append(parts, string(arg.Value))
				} else {
					parts = append(parts, arg.Description)
				}
			}
			if len(parts) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "chrome[%s] console: %s\n", name, strings.Join(parts, " "))
			}
		case *cdpRuntime.EventExceptionThrown:
			_, _ = fmt.Fprintf(os.Stderr, "chrome[%s] exception: %s\n", name, ev.ExceptionDetails.Text)
		}
	})

	if err := chromedp.Run(tabCtx); err != nil {
		cancel()
		return nil, err
	}

	if err := player.run(defaultActionTimeout,
		chromedp.Navigate(w.suite.harnessURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		cancel()
		return nil, err
	}
	if err := player.waitForHarness(); err != nil {
		cancel()
		return nil, err
	}
	return player, nil
}

func (p *browserPlayer) waitForHarness() error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var ready bool
		err := p.run(2*time.Second, chromedp.Evaluate(`window.netlibTestReady === true`, &ready))
		if err == nil && ready {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	diagnostics, err := p.harnessDiagnostics()
	if err != nil {
		return fmt.Errorf("browser harness did not become ready; diagnostics failed: %w", err)
	}
	return fmt.Errorf("browser harness did not become ready: %s", diagnostics)
}

func (p *browserPlayer) harnessDiagnostics() (string, error) {
	var diagnostics struct {
		Href      string   `json:"href"`
		Ready     any      `json:"ready"`
		TestType  string   `json:"testType"`
		Errors    []string `json:"errors"`
		BodyText  string   `json:"bodyText"`
		ScriptSrc []string `json:"scriptSrc"`
	}
	err := p.run(2*time.Second, chromedp.Evaluate(`(() => ({
  href: window.location.href,
  ready: window.netlibTestReady,
  testType: typeof window.netlibTest,
  errors: window.netlibTestLoadErrors || [],
  bodyText: document.body ? document.body.innerText : "",
  scriptSrc: Array.from(document.scripts).map(script => script.src || "<inline>")
}))()`, &diagnostics))
	if err != nil {
		return "", err
	}
	return compactJSON(diagnostics), nil
}

func (p *browserPlayer) call(name string, result any, args ...any) error {
	if args == nil {
		args = []any{}
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return err
	}
	expr := fmt.Sprintf(`(async () => {
  const args = %s
  try {
    return await window.netlibTest[%q](...args)
  } catch (e) {
    const message = e && (e.stack || e.message) ? (e.stack || e.message) : String(e)
    throw new Error(message)
  }
})()`, argsJSON, name)
	return p.run(defaultActionTimeout, chromedp.Evaluate(expr, result, evalAwaitPromise))
}

func (p *browserPlayer) run(timeout time.Duration, actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

func (p *browserPlayer) Close() error {
	_ = p.call("closeNetwork", nil, "closing test suite")
	p.cancel()
	return nil
}

func evalAwaitPromise(p *cdpRuntime.EvaluateParams) *cdpRuntime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

type backendProcess struct {
	name   string
	cmd    *exec.Cmd
	logMu  sync.Mutex
	logs   bytes.Buffer
	waitCh chan error
}

func (w *scenarioWorld) startBackend(backend string) error {
	if _, ok := w.backends[backend]; ok {
		return nil
	}

	port, err := getFreePort()
	if err != nil {
		return err
	}

	var binary string
	switch backend {
	case "signaling":
		binary = w.suite.signalingBinary
	case "testproxy":
		binary = w.suite.testproxyBinary
	default:
		return fmt.Errorf("unknown backend %q", backend)
	}

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"ADDR=127.0.0.1:"+port,
		"ENV=test",
	)
	if w.databaseURL != "" {
		cmd.Env = append(cmd.Env, "DATABASE_URL="+w.databaseURL)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	proc := &backendProcess{
		name:   backend,
		cmd:    cmd,
		waitCh: make(chan error, 1),
	}

	readyCh := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return err
	}

	go copyBackendOutput(stdout, proc, nil)
	go copyBackendOutput(stderr, proc, func(line string) {
		entry := struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
			URL      string `json:"url"`
		}{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return
		}
		severity := strings.ToLower(entry.Severity)
		if entry.Message == "using database" && entry.URL != "" {
			w.databaseURL = entry.URL
		}
		if entry.Message == "listening" {
			select {
			case readyCh <- nil:
			default:
			}
		}
		if severity == "error" || severity == "emergency" {
			select {
			case readyCh <- fmt.Errorf("error before backend was started: %s", entry.Message):
			default:
			}
		}
	})
	go func() {
		proc.waitCh <- cmd.Wait()
		close(proc.waitCh)
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			_ = proc.Close()
			return err
		}
	case err := <-proc.waitCh:
		return fmt.Errorf("%s exited before it was ready: %w\n%s", backend, err, proc.Logs())
	case <-time.After(60 * time.Second):
		_ = proc.Close()
		return fmt.Errorf("%s did not start within 60s\n%s", backend, proc.Logs())
	}

	w.backends[backend] = proc
	switch backend {
	case "signaling":
		w.signalingURL = "ws://127.0.0.1:" + port + "/v0/signaling"
	case "testproxy":
		w.testproxyURL = "http://127.0.0.1:" + port
	}
	return nil
}

func copyBackendOutput(r io.Reader, proc *backendProcess, lineFunc func(string)) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		proc.appendLog(line + "\n")
		if lineFunc != nil {
			lineFunc(strings.TrimSpace(line))
		}
	}
	if err := scanner.Err(); err != nil {
		proc.appendLog("error reading " + proc.name + " output: " + err.Error() + "\n")
	}
}

func getFreePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	return port, err
}

func (p *backendProcess) Close() error {
	if p.cmd.Process == nil {
		return nil
	}

	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.waitCh:
		return nil
	case <-time.After(10 * time.Second):
		_ = p.cmd.Process.Kill()
		err := <-p.waitCh
		if err != nil {
			return fmt.Errorf("killed after timeout: %w", err)
		}
		return errors.New("killed after timeout")
	}
}

func (p *backendProcess) appendLog(line string) {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	_, _ = p.logs.WriteString(line)
}

func (p *backendProcess) Logs() string {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	return p.logs.String()
}

func withGeo(signalingURL string, country string, region string) (string, error) {
	if country == "" && region == "" {
		return signalingURL, nil
	}
	parsed, err := url.Parse(signalingURL)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	if country != "" {
		q.Set("country", country)
	}
	if region != "" {
		q.Set("region", region)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func parseJSON(raw string) (any, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func compactJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(raw)
}

func tableHashes(table *godog.Table) []map[string]string {
	if table == nil || len(table.Rows) == 0 {
		return nil
	}
	headers := make([]string, 0, len(table.Rows[0].Cells))
	for _, cell := range table.Rows[0].Cells {
		headers = append(headers, cell.Value)
	}

	hashes := make([]map[string]string, 0, len(table.Rows)-1)
	for _, row := range table.Rows[1:] {
		hash := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(row.Cells) {
				hash[header] = row.Cells[i].Value
			} else {
				hash[header] = ""
			}
		}
		hashes = append(hashes, hash)
	}
	return hashes
}

func tableHeaders(table *godog.Table) []string {
	if table == nil || len(table.Rows) == 0 {
		return nil
	}
	headers := make([]string, 0, len(table.Rows[0].Cells))
	for _, cell := range table.Rows[0].Cells {
		headers = append(headers, cell.Value)
	}
	return headers
}

func rowsHash(table *godog.Table) map[string]string {
	out := make(map[string]string)
	if table == nil {
		return out
	}
	for _, row := range table.Rows {
		if len(row.Cells) >= 2 {
			out[row.Cells[0].Value] = row.Cells[1].Value
		}
	}
	return out
}

func httpGetOK(rawURL string) error {
	resp, err := http.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: %s: %s", rawURL, resp.Status, string(body))
	}
	return nil
}

func httpPostOK(rawURL string, body string) error {
	resp, err := http.Post(rawURL, "text/plain", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: %s: %s", rawURL, resp.Status, string(responseBody))
	}
	return nil
}

func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
