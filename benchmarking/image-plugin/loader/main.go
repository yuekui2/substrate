// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command loader drives the image plugin benchmark from inside the cluster.
//
//	loader bake -arm riptide ...   create one template per image, wait for goldens
//	loader run  -arm riptide ...   N actors x T turns of resume, probe, work,
//	                               suspend and think
//
// run writes one JSON line per actor turn to -out. With more actors than
// images, actor i uses image i mod len(images).
package main

import (
	"bufio"
	"compress/zlib"
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"go.ate.dev/substrate/api/ateapi/v1"
	"go.ate.dev/substrate/internal/ateclient"
	"go.ate.dev/substrate/internal/atenet"
	"go.ate.dev/substrate/internal/clustertrustbundle"
	"go.ate.dev/substrate/internal/podidentity"
	"go.ate.dev/substrate/internal/resources"
)

//go:embed guest.py
var guestPy []byte

const (
	systemNamespace = "ate-system"
	apiServerName   = "api.ate-system.svc"
	routerName      = "atenet-router.ate-system.svc"
	routerAddr      = "atenet-router.ate-system.svc:8444"
	clientSA        = "ate-client"
	// routerSA is the ServiceAccount of the router client certificate. This
	// install's router allows only ate-client (--ingress-allowed-spiffe-ids).
	routerSA      = "ate-client"
	certRequester = "image-plugin-bench"
)

type image struct {
	InstanceID string `json:"instance_id"`
	OCI        string `json:"oci"`
	Nydus      string `json:"nydus"`
	Python     string `json:"python"`
}

type opts struct {
	arm, atespace, pool, imagesPath, out, snapshotLoc string
	actors, turns, rep, reads, writes, concurrency    int
	cpuMilli, memBytes                                int64
	probeTimeout                                      int
	wakeBudget, workTimeout, thinkMin, thinkMax       time.Duration
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: loader bake|run [flags]")
		os.Exit(2)
	}
	mode := os.Args[1]
	var o opts
	fs := flag.NewFlagSet(mode, flag.ExitOnError)
	fs.StringVar(&o.arm, "arm", "", "arm name: pull, riptide, nydus or spegel")
	fs.StringVar(&o.atespace, "atespace", "", "atespace of the arm's templates and actors")
	fs.StringVar(&o.pool, "workload", "", "workerSelector workload label of the arm's WorkerPool")
	fs.StringVar(&o.imagesPath, "images", "/config/images-320.jsonl", "image list")
	fs.StringVar(&o.out, "out", "/results/turns.jsonl", "output JSONL")
	fs.StringVar(&o.snapshotLoc, "snapshot-location", "", "template snapshot storage location")
	fs.IntVar(&o.actors, "actors", 320, "actors; actor i uses image i mod len(images)")
	fs.IntVar(&o.turns, "turns", 10, "turns per actor (first restore included)")
	fs.IntVar(&o.rep, "rep", 1, "rep number, used in actor names")
	fs.IntVar(&o.reads, "reads", 200, ".py files read per turn")
	fs.IntVar(&o.writes, "writes", 2, ".py files written per turn")
	fs.IntVar(&o.concurrency, "concurrency", 32, "bake concurrency")
	fs.Int64Var(&o.cpuMilli, "cpu-millicores", 100, "actor CPU limit")
	fs.Int64Var(&o.memBytes, "memory-bytes", 204*1024*1024, "actor memory limit")
	fs.IntVar(&o.probeTimeout, "wakeup-timeout-seconds", 180, "template wakeup probe timeout")
	fs.DurationVar(&o.wakeBudget, "wake-budget", 15*time.Minute, "time a turn may wait for a worker and the first probe")
	fs.DurationVar(&o.workTimeout, "work-timeout", 2*time.Minute, "timeout of the work call")
	fs.DurationVar(&o.thinkMin, "think-min", 12*time.Second, "minimum think time between a suspend and the next resume")
	fs.DurationVar(&o.thinkMax, "think-max", 18*time.Second, "maximum think time between a suspend and the next resume")
	linger := fs.Duration("linger", 0, "after finishing, write <out dir>/done and wait this long so the results can be copied out")
	serve := fs.String("serve", ":8080", "address to serve the results directory on, empty to disable")
	_ = fs.Parse(os.Args[2:])
	lingerFor, outDir = *linger, dirOf(o.out)
	defer finish(0)
	if *serve != "" {
		_ = os.MkdirAll(outDir, 0o755)
		go func() {
			slog.Info("serving results", "addr", *serve, "dir", outDir)
			_ = http.ListenAndServe(*serve, http.FileServer(http.Dir(outDir)))
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	imgs, err := readImages(o.imagesPath, o.actors)
	if err != nil {
		fatal("reading images", err)
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		fatal("in-cluster config", err)
	}
	cfg.WarningHandlerWithContext = &rest.NoWarnings{}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fatal("clientset", err)
	}
	api, err := newAPIClient(ctx, cs)
	if err != nil {
		fatal("ate-api client", err)
	}
	defer api.Close()

	switch mode {
	case "bake":
		err = bake(ctx, api, o, imgs)
	case "run":
		var d *dialer
		d, err = newDialer(ctx, cs)
		if err == nil {
			err = run(ctx, api, d, o, imgs)
		}
	default:
		err = fmt.Errorf("unknown mode %q", mode)
	}
	if err != nil {
		fatal(mode, err)
	}
}

var (
	lingerFor time.Duration
	outDir    string
)

// finish writes <out dir>/done, waits -linger so the results can be copied
// out, and exits with code.
func finish(code int) {
	if lingerFor > 0 {
		_ = os.MkdirAll(outDir, 0o755)
		_ = os.WriteFile(outDir+"/done", []byte(fmt.Sprintf("%d\n", code)), 0o644)
		time.Sleep(lingerFor)
	}
	os.Exit(code)
}

func fatal(what string, err error) {
	slog.Error("loader failed", "step", what, "err", err)
	finish(1)
}

// readImages reads up to n images from path. It returns fewer when the file
// has fewer: bake makes one template each, and run reuses them in order.
func readImages(path string, n int) ([]image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []image
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() && len(out) < n {
		var img image
		if err := json.Unmarshal(sc.Bytes(), &img); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no images in %s", path)
	}
	return out, sc.Err()
}

// newAPIClient dials ate-api-server as the ate-client ServiceAccount. The
// token is refreshed into a file, which the client reads on every call.
func newAPIClient(ctx context.Context, cs *kubernetes.Clientset) (*ateclient.Client, error) {
	tokenFile := "/tmp/ateapi-token"
	refresh := func() error {
		exp := int64(3600)
		tok, err := cs.CoreV1().ServiceAccounts(systemNamespace).CreateToken(ctx, clientSA, &authv1.TokenRequest{
			Spec: authv1.TokenRequestSpec{Audiences: []string{apiServerName}, ExpirationSeconds: &exp},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return os.WriteFile(tokenFile, []byte(tok.Status.Token), 0o600)
	}
	if err := refresh(); err != nil {
		return nil, fmt.Errorf("minting ate-client token: %w", err)
	}
	go func() {
		t := time.NewTicker(20 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := refresh(); err != nil {
					slog.Warn("token refresh failed", "err", err)
				}
			}
		}
	}()
	return ateclient.NewClient(ctx, "", "", "dns:///"+apiServerName+":443", tokenFile, false)
}

func templateName(i int) string { return fmt.Sprintf("img-%03d", i) }

func templateJSON(o opts, i int, img image) (string, error) {
	ref := img.OCI
	if o.arm == "nydus" {
		ref = img.Nydus
	}
	python := img.Python
	if python == "" {
		python = "/opt/miniconda3/envs/testbed/bin/python"
	}
	// A command argument may hold at most 4096 characters, so the guest is
	// compressed.
	var z strings.Builder
	enc := base64.NewEncoder(base64.StdEncoding, &z)
	zw := zlib.NewWriter(enc)
	_, _ = zw.Write(guestPy)
	_ = zw.Close()
	_ = enc.Close()
	boot := fmt.Sprintf("import base64,zlib;exec(zlib.decompress(base64.b64decode('%s')))", z.String())
	t := map[string]any{
		"metadata":       map[string]any{"atespace": o.atespace, "name": templateName(i)},
		"workerSelector": map[string]any{"matchLabels": map[string]string{"workload": o.pool}},
		"containers": []any{map[string]any{
			"name":        "guest",
			"image":       ref,
			"command":     []string{python, "-c", boot},
			"wakeupProbe": map[string]any{"httpGet": map[string]any{"path": "/readyz", "port": 80}, "timeoutSeconds": o.probeTimeout},
		}},
		"resources":      map[string]any{"limits": map[string]any{"cpuMillicores": o.cpuMilli, "memoryBytes": o.memBytes}},
		"snapshotConfig": map[string]any{"preferredFidelity": "SNAPSHOT_FIDELITY_MEMORY", "storageLocation": o.snapshotLoc},
	}
	b, err := json.Marshal(t)
	return string(b), err
}

// bake creates one template per image and waits until every golden snapshot
// exists. Bakes are not timed.
func bake(ctx context.Context, api *ateclient.Client, o opts, imgs []image) error {
	sem := make(chan struct{}, o.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string
	for i, img := range imgs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := bakeOne(ctx, api, o, i, img); err != nil {
				slog.Error("bake failed", "template", templateName(i), "image", img.InstanceID, "err", err)
				mu.Lock()
				failed = append(failed, templateName(i))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	slog.Info("bake done", "arm", o.arm, "templates", len(imgs), "failed", len(failed))
	if len(failed) > 0 {
		return fmt.Errorf("%d goldens failed: %v", len(failed), failed)
	}
	return nil
}

func bakeOne(ctx context.Context, api *ateclient.Client, o opts, i int, img image) error {
	ref := &ateapi.ObjectRef{Atespace: o.atespace, Name: templateName(i)}
	if _, err := api.GetActorTemplate(ctx, &ateapi.GetActorTemplateRequest{ActorTemplate: ref}); status.Code(err) == codes.NotFound {
		js, err := templateJSON(o, i, img)
		if err != nil {
			return err
		}
		var tpl ateapi.ActorTemplate
		if err := protojson.Unmarshal([]byte(js), &tpl); err != nil {
			return fmt.Errorf("template JSON: %w", err)
		}
		if _, err := api.CreateActorTemplate(ctx, &ateapi.CreateActorTemplateRequest{ActorTemplate: &tpl}); err != nil {
			return fmt.Errorf("create: %w", err)
		}
	} else if err != nil {
		return err
	}
	start := time.Now()
	for {
		tpl, err := api.GetActorTemplate(ctx, &ateapi.GetActorTemplateRequest{ActorTemplate: ref})
		if err == nil {
			g := tpl.GetStatus().GetGoldenSnapshotStatus()
			if g.GetGoldenTag().GetName() != "" {
				slog.Info("golden ready", "template", ref.GetName(), "elapsed", time.Since(start).Round(time.Second))
				return nil
			}
			if g.GetErrorMessage() != "" {
				return errors.New(g.GetErrorMessage())
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// dialer opens CONNECT tunnels to actors through atenet-router.
type dialer struct {
	cs     kubernetes.Interface
	tlsCfg *tls.Config

	mu        sync.Mutex
	cert      *tls.Certificate
	refreshAt time.Time
}

func newDialer(ctx context.Context, cs kubernetes.Interface) (*dialer, error) {
	bundles, err := clustertrustbundle.NewClient(cs, nil)
	if err != nil {
		return nil, err
	}
	ctbs, err := bundles.List(ctx, metav1.ListOptions{LabelSelector: "podcert.ate.dev/canarying=live"})
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	for _, ctb := range ctbs.Items {
		if ctb.Spec.SignerName == "servicedns.podcert.ate.dev/identity" {
			pool.AppendCertsFromPEM([]byte(ctb.Spec.TrustBundle))
		}
	}
	d := &dialer{cs: cs}
	d.tlsCfg = &tls.Config{
		MinVersion: tls.VersionTLS13,
		RootCAs:    pool,
		ServerName: routerName,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return d.clientCert(info.Context())
		},
	}
	if _, err := d.clientCert(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dialer) clientCert(ctx context.Context) (*tls.Certificate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cert != nil && time.Now().Before(d.refreshAt) {
		return d.cert, nil
	}
	cert, err := podidentity.RequestCertificate(ctx, d.cs, certRequester, systemNamespace, routerSA, 2*time.Minute)
	if err != nil {
		if d.cert != nil && time.Now().Before(d.cert.Leaf.NotAfter) {
			d.refreshAt = time.Now().Add(time.Minute)
			return d.cert, nil
		}
		return nil, err
	}
	d.cert = &cert
	d.refreshAt = cert.Leaf.NotBefore.Add(cert.Leaf.NotAfter.Sub(cert.Leaf.NotBefore) / 2)
	return d.cert, nil
}

// get sends one HTTP GET to the actor's port 80 over a fresh tunnel.
func (d *dialer) get(ctx context.Context, ref resources.ActorRef, path string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return ateclient.ConnectActorWithHeaders(ctx, routerAddr, d.tlsCfg, ref, 80,
				http.Header{atenet.TargetActorHeader: []string{ref.String()}}, nil)
		},
	}
	defer tr.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://actor"+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// turnResult is one JSON line of run's output.
type turnResult struct {
	Arm        string  `json:"arm"`
	Rep        int     `json:"rep"`
	Actor      string  `json:"actor"`
	Image      string  `json:"image"`
	Turn       int     `json:"turn"`
	Start      string  `json:"start"`
	CreateS    float64 `json:"create_s,omitempty"`
	QueueS     float64 `json:"queue_s"`
	ResumeS    float64 `json:"resume_s"`
	NoRoom     int     `json:"no_room_retries"`
	ProbeTries int     `json:"probe_tries"`
	TTFIS      float64 `json:"ttfi_s"`
	ReadS      float64 `json:"read_s"`
	WriteS     float64 `json:"write_s"`
	WorkS      float64 `json:"work_s"`
	WorkCallS  float64 `json:"work_call_s"`
	TaskS      float64 `json:"task_s"`
	SuspendS   float64 `json:"suspend_s"`
	TurnS      float64 `json:"turn_s"`
	ThinkS     float64 `json:"think_s"`
	FilesRead  int     `json:"files_read"`
	Node       string  `json:"node,omitempty"`
	PrevNode   string  `json:"prev_node,omitempty"`
	Moved      bool    `json:"moved"`
	Error      string  `json:"error,omitempty"`
	ErrorStep  string  `json:"error_step,omitempty"`
}

func run(ctx context.Context, api *ateclient.Client, d *dialer, o opts, imgs []image) error {
	if err := os.MkdirAll(dirOf(o.out), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(o.out, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	var outMu sync.Mutex
	emit := func(r turnResult) {
		b, _ := json.Marshal(r)
		outMu.Lock()
		defer outMu.Unlock()
		_, _ = f.Write(append(b, '\n'))
	}

	// Use only templates whose golden exists. An image whose guest never
	// starts fails its bake on every arm.
	var ready []int
	for i := range imgs {
		tpl, err := api.GetActorTemplate(ctx, &ateapi.GetActorTemplateRequest{
			ActorTemplate: &ateapi.ObjectRef{Atespace: o.atespace, Name: templateName(i)}})
		if err != nil {
			return fmt.Errorf("getting template %s: %w", templateName(i), err)
		}
		if tpl.GetStatus().GetGoldenSnapshotStatus().GetGoldenTag().GetName() == "" {
			slog.Warn("skipping template without a golden", "template", templateName(i), "image", imgs[i].InstanceID)
			continue
		}
		ready = append(ready, i)
	}
	if len(ready) == 0 {
		return errors.New("no template has a golden")
	}
	slog.Info("templates ready", "ready", len(ready), "images", len(imgs), "actors", o.actors)

	runStart := time.Now()
	var wg sync.WaitGroup
	for i := range o.actors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tpl := ready[i%len(ready)]
			actorLoop(ctx, api, d, o, i, tpl, imgs[tpl], emit)
		}()
	}
	wg.Wait()
	slog.Info("run done", "arm", o.arm, "rep", o.rep, "elapsed", time.Since(runStart).Round(time.Second))
	return nil
}

func dirOf(p string) string {
	if i := strings.LastIndex(p, "/"); i > 0 {
		return p[:i]
	}
	return "."
}

// actorLoop creates actor i from template tpl and runs its turns, with a
// think time between a suspend and the next resume.
func actorLoop(ctx context.Context, api *ateclient.Client, d *dialer, o opts, i, tpl int, img image, emit func(turnResult)) {
	name := fmt.Sprintf("r%d-%03d", o.rep, i)
	objRef := &ateapi.ObjectRef{Atespace: o.atespace, Name: name}
	ref := resources.ActorRef{Atespace: o.atespace, Name: name}
	createStart := time.Now()
	_, err := api.CreateActor(ctx, &ateapi.CreateActorRequest{Actor: &ateapi.Actor{
		Metadata:      &ateapi.ResourceMetadata{Atespace: o.atespace, Name: name},
		ActorTemplate: &ateapi.ObjectRef{Atespace: o.atespace, Name: templateName(tpl)},
	}})
	createS := time.Since(createStart).Seconds()
	if err != nil && status.Code(err) != codes.AlreadyExists {
		emit(turnResult{Arm: o.arm, Rep: o.rep, Actor: name, Image: img.InstanceID, Error: err.Error(), ErrorStep: "create"})
		return
	}
	var prevNode string
	var think time.Duration
	for turn := 0; turn < o.turns && ctx.Err() == nil; turn++ {
		// The first turn's TTFI starts at the create call.
		t0 := time.Now()
		if turn == 0 {
			t0 = createStart
		}
		r := oneTurn(ctx, api, d, o, objRef, ref, img, turn, t0)
		if turn == 0 {
			r.CreateS = createS
		}
		r.ThinkS = think.Seconds()
		r.PrevNode = prevNode
		r.Moved = prevNode != "" && r.Node != "" && r.Node != prevNode
		if r.Node != "" {
			prevNode = r.Node
		}
		emit(r)
		if turn+1 < o.turns {
			think = o.thinkMin
			if o.thinkMax > o.thinkMin {
				think += time.Duration(rand.Int64N(int64(o.thinkMax - o.thinkMin)))
			}
			sleep(ctx, think)
		}
	}
	_, _ = api.DeleteActor(context.Background(), &ateapi.DeleteActorRequest{Actor: objRef})
}

// oneTurn resumes the actor, probes it, runs the work and suspends it. TTFI
// is measured from t0.
func oneTurn(ctx context.Context, api *ateclient.Client, d *dialer, o opts, objRef *ateapi.ObjectRef, ref resources.ActorRef, img image, turn int, t0 time.Time) turnResult {
	r := turnResult{Arm: o.arm, Rep: o.rep, Actor: objRef.GetName(), Image: img.InstanceID, Turn: turn}
	r.Start = t0.UTC().Format(time.RFC3339Nano)
	deadline := t0.Add(o.wakeBudget)
	fail := func(step string, err error) turnResult {
		r.Error, r.ErrorStep = err.Error(), step
		r.TurnS = time.Since(t0).Seconds()
		// Leave the actor suspended for the next turn.
		_, _ = api.SuspendActor(context.Background(), &ateapi.SuspendActorRequest{Actor: objRef})
		return r
	}

	// Substrate refuses a resume with ResourceExhausted when no worker has
	// room, so the loader holds the actor in its own queue and retries. The
	// short cap keeps a freed slot from sitting idle; the wait counts in TTFI.
	backoff := 50 * time.Millisecond
	for {
		callStart := time.Now()
		resp, err := api.ResumeActor(ctx, &ateapi.ResumeActorRequest{Actor: objRef})
		if err == nil {
			r.ResumeS = time.Since(callStart).Seconds()
			r.QueueS = callStart.Sub(t0).Seconds()
			r.Node = resp.GetActor().GetStatus().GetWorkerAssignment().GetNodeName()
			break
		}
		code := status.Code(err)
		if (code != codes.ResourceExhausted && code != codes.Aborted && code != codes.Unavailable) || time.Now().After(deadline) {
			return fail("resume", err)
		}
		if code == codes.ResourceExhausted {
			r.NoRoom++
		}
		sleep(ctx, backoff+time.Duration(rand.Int64N(int64(backoff))))
		backoff = min(backoff*2, 250*time.Millisecond)
	}

	probeStart := time.Now()
	for {
		r.ProbeTries++
		_, err := d.get(ctx, ref, "/probe", 30*time.Second)
		if err == nil {
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return fail("probe", err)
		}
		if r.ProbeTries == 1 || r.ProbeTries%40 == 0 {
			slog.Warn("probe failed; retrying", "actor", objRef.GetName(), "turn", turn, "tries", r.ProbeTries, "err", err)
		}
		sleep(ctx, 250*time.Millisecond)
	}
	r.TTFIS = time.Since(t0).Seconds()
	probeS := time.Since(probeStart).Seconds()

	workStart := time.Now()
	body, err := d.get(ctx, ref, fmt.Sprintf("/work?reads=%d&writes=%d&turn=%d", o.reads, o.writes, turn), o.workTimeout)
	r.WorkCallS = time.Since(workStart).Seconds()
	if err != nil {
		return fail("work", err)
	}
	var w struct {
		FilesRead int     `json:"files_read"`
		ReadS     float64 `json:"read_s"`
		WriteS    float64 `json:"write_s"`
		WorkS     float64 `json:"work_s"`
	}
	if err := json.Unmarshal(body, &w); err != nil {
		return fail("work", fmt.Errorf("decoding %q: %w", body, err))
	}
	r.FilesRead, r.ReadS, r.WriteS, r.WorkS = w.FilesRead, w.ReadS, w.WriteS, w.WorkS
	r.TaskS = probeS + r.WorkCallS

	suspendStart := time.Now()
	if _, err := api.SuspendActor(ctx, &ateapi.SuspendActorRequest{Actor: objRef}); err != nil {
		r.Error, r.ErrorStep = err.Error(), "suspend"
	}
	r.SuspendS = time.Since(suspendStart).Seconds()
	r.TurnS = time.Since(t0).Seconds()
	return r
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
