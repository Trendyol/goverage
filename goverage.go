//go:build goverage

package goverage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime/coverage"
	"strings"
	"time"
)

type CoverageProfileRequest struct {
	SkipFile []string `json:"skipFile"`
}

type CoverageGenerator struct {
	config *Config
	logger *Logger
}

type CoverageServer struct {
	server    *http.Server
	config    *Config
	generator *CoverageGenerator
	logger    *Logger
}

func NewCoverageGenerator(config *Config, logger *Logger) *CoverageGenerator {
	return &CoverageGenerator{
		config: config,
		logger: logger,
	}
}

func NewCoverageServer() *CoverageServer {
	config := NewConfig()
	logger := NewLogger()
	generator := NewCoverageGenerator(config, logger)

	cs := &CoverageServer{
		config:    config,
		generator: generator,
		logger:    logger,
	}
	cs.setupServer()

	return cs
}

func (cs *CoverageServer) Start() {
	go func() {
		cs.logger.Info("coverage server listening on :%s", cs.config.Port)
		if err := cs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cs.logger.Error("coverage server error: %v", err)
		}
	}()
}

func (cs *CoverageServer) setupServer() {
	mux := http.NewServeMux()
	mux.HandleFunc(coverageEndpoint, cs.handleCoverageProfile)

	cs.server = &http.Server{
		Addr:              ":" + cs.config.Port,
		Handler:           mux,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func (cs *CoverageServer) httpError(w http.ResponseWriter, message string, code int) {
	cs.logger.Error(message)
	http.Error(w, message, code)
}

func (cs *CoverageServer) logRequest(r *http.Request) {
	cs.logger.Info("incoming request method=%s path=%s remote=%s ua=%s",
		r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
}

func (cs *CoverageServer) handleCoverageProfile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	cs.logRequest(r)

	req, err := cs.parseRequestBody(r)
	if err != nil {
		cs.httpError(w, fmt.Sprintf("failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := cs.generateAndServeCoverageProfile(w, r, req, start); err != nil {
		cs.logger.Error("failed to generate coverage profile: %v", err)
	}
}

func (cs *CoverageServer) parseRequestBody(r *http.Request) (*CoverageProfileRequest, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	cs.logger.Info("request body: %s", string(bodyBytes))

	var req CoverageProfileRequest
	if len(bodyBytes) > 0 {
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&req); err != nil {
			return nil, fmt.Errorf("decoding JSON: %w", err)
		}
	}

	cs.logger.Info("skipFile count=%d list=%v", len(req.SkipFile), req.SkipFile)
	return &req, nil
}

func (cs *CoverageServer) generateAndServeCoverageProfile(w http.ResponseWriter, r *http.Request, req *CoverageProfileRequest, start time.Time) error {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	coverageData, err := cs.generator.GenerateCoverageProfile(ctx, req.SkipFile)
	if err != nil {
		cs.httpError(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	cs.writeResponse(w, coverageData, len(coverageData), start)
	return nil
}

func (cg *CoverageGenerator) GenerateCoverageProfile(ctx context.Context, skipFiles []string) (string, error) {
	if err := cg.validateCoverDir(); err != nil {
		return "", err
	}

	if err := cg.writeCoverageData(); err != nil {
		return "", err
	}

	coverageData, err := cg.generateCoverageReport(ctx)
	if err != nil {
		return "", err
	}

	return cg.filterCoverageText(string(coverageData), skipFiles), nil
}

func (cg *CoverageGenerator) validateCoverDir() error {
	if cg.config.CoverDir == "" {
		return errors.New("GOCOVERDIR is not set")
	}

	cg.logger.Info("using GOCOVERDIR=%s", cg.config.CoverDir)

	if err := os.MkdirAll(cg.config.CoverDir, coverageFileMode); err != nil {
		return fmt.Errorf("creating coverage directory: %w", err)
	}

	cg.logger.Info("ensured coverage dir exists: %s", cg.config.CoverDir)
	return nil
}

func (cg *CoverageGenerator) writeCoverageData() error {
	if err := coverage.WriteMetaDir(cg.config.CoverDir); err != nil {
		return fmt.Errorf("writing coverage meta: %w", err)
	}

	if err := coverage.WriteCountersDir(cg.config.CoverDir); err != nil {
		return fmt.Errorf("writing coverage counters: %w", err)
	}

	cg.logger.Info("wrote coverage meta and counters to %s", cg.config.CoverDir)
	return nil
}

func (cg *CoverageGenerator) generateCoverageReport(ctx context.Context) ([]byte, error) {
	binary, args := cg.prepareCoverageCommand()

	cmd := exec.CommandContext(ctx, binary, args...)
	cg.logger.Info("executing: %s %v", binary, args)

	output, err := cmd.CombinedOutput()
	if err != nil {
		cg.logger.Error("textfmt failed: err=%v, out=%s", err, string(output))
		return nil, fmt.Errorf("coverage command failed: %w", err)
	}

	cg.logger.Info("textfmt wrote to: %s", temporaryCoverageFile)

	data, err := os.ReadFile(temporaryCoverageFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("no coverage output produced")
		}
		return nil, fmt.Errorf("reading coverage file: %w", err)
	}

	return data, nil
}

func (cg *CoverageGenerator) prepareCoverageCommand() (string, []string) {
	baseArgs := []string{"textfmt", "-i=" + cg.config.CoverDir, "-o=" + temporaryCoverageFile}

	if covdataPath, err := exec.LookPath("covdata"); err == nil {
		cg.logger.Info("selected tool bin=%s args=%v", covdataPath, baseArgs)
		return covdataPath, baseArgs
	}

	goArgs := append([]string{"tool", "covdata"}, baseArgs...)
	cg.logger.Info("selected tool bin=%s args=%v", "go", goArgs)
	return "go", goArgs
}

func (cg *CoverageGenerator) filterCoverageText(text string, skipPatterns []string) string {
	lines := strings.Split(text, "\n")
	compiledPatterns := cg.compileSkipPatterns(skipPatterns)

	filtered := make([]string, 0, len(lines))
	for i, line := range lines {
		if cg.shouldKeepLine(line, i, compiledPatterns) {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

func (cg *CoverageGenerator) compileSkipPatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, re)
		} else {
			cg.logger.Error("invalid regex pattern: %s, error: %v", pattern, err)
		}
	}
	return compiled
}

func (cg *CoverageGenerator) shouldKeepLine(line string, lineIndex int, patterns []*regexp.Regexp) bool {
	if lineIndex == 0 && strings.HasPrefix(line, modePrefix) {
		return true
	}

	if strings.TrimSpace(line) == "" || strings.Contains(line, "goverage.go") {
		return false
	}

	pathEndIndex := strings.IndexByte(line, ':')
	if pathEndIndex < 0 {
		return true
	}

	path := line[:pathEndIndex]
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return false
		}
	}

	return true
}

func (cs *CoverageServer) writeResponse(w http.ResponseWriter, data string, originalSize int, start time.Time) {
	w.Header().Set("Content-Type", contentTypeTextPlain)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(data)); err != nil {
		cs.logger.Error("failed to write response: %v", err)
	}

	cs.logger.Info("served coverage profile bytes=%d duration=%s", originalSize, time.Since(start))
}

func init() {
	server := NewCoverageServer()
	server.Start()
}

func main() {
	select {}
}
