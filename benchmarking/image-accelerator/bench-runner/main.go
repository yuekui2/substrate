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

// Command bench-runner drives e2e functionality, upgrade, and latency
// benchmarks for ImageAccelerator adapters on a live Agent Substrate cluster.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	imageaccel "go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	"go.ate.dev/substrate/pkg/proto/ateapipb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const targetActorHeader = "ate-target-actor"

type ProcessRequest struct {
	Command []string          `json:"command"`
	EnvVars map[string]string `json:"envvars,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
}

type ProcessResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

type BenchResult struct {
	Adapter          string  `json:"adapter"`
	Scenario         string  `json:"scenario"`
	Workload         string  `json:"workload"`
	ActorRef         string  `json:"actor_ref"`
	WorkerPod        string  `json:"worker_pod"`
	SnapshotURI      string  `json:"snapshot_uri,omitempty"`
	ResumeRPCStartNs int64   `json:"resume_rpc_start_ns"`
	ResumeRPCDoneNs  int64   `json:"resume_rpc_done_ns"`
	ResumeRPCDurMs   float64 `json:"resume_rpc_dur_ms"`
	CmdT0Ns          int64   `json:"cmd_t0_ns"`
	CmdT1Ns          int64   `json:"cmd_t1_ns"`
	HTTPDoneNs       int64   `json:"http_done_ns"`
	ClientTTCEMs     float64 `json:"client_ttce_ms"`
	WorkloadRunMs    float64 `json:"workload_run_ms"`
	ClientE2EMs      float64 `json:"client_e2e_ms"`
	ExitCode         int     `json:"exit_code"`
	StdoutPreview    string  `json:"stdout_preview"`
}

type ProbeReport struct {
	Provider       string   `json:"provider"`
	ImageRef       string   `json:"imageRef"`
	ImageDigest    string   `json:"imageDigest,omitempty"`
	CanProvide     bool     `json:"canProvide"`
	LayerDirs      []string `json:"layerDirs,omitempty"`
	CanProvideMs   float64  `json:"canProvideMs"`
	PrepareMs      float64  `json:"prepareMs"`
	ReleaseMs      float64  `json:"releaseMs"`
	VerifiedLayers int      `json:"verifiedLayers"`
}

var timingRegex = regexp.MustCompile(`___BENCH_TIMING___ t0=(\d+) t1=(\d+) rc=(\d+)`)

type bearerTokenCreds string

func (c bearerTokenCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	if c == "" {
		return nil, nil
	}
	return map[string]string{"authorization": "Bearer " + string(c)}, nil
}

func (c bearerTokenCreds) RequireTransportSecurity() bool { return true }

func dialAteAPI(endpoint, token string) (ateapipb.ControlClient, *grpc.ClientConn, error) {
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	if token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(bearerTokenCreds(token)))
	}
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, nil, err
	}
	return ateapipb.NewControlClient(conn), conn, nil
}

func execInActor(ctx context.Context, atenetAddr, actorRefStr, cmdStr, timeoutStr string) (*ProcessResponse, error) {
	url := fmt.Sprintf("https://%s/process", atenetAddr)
	reqBody := ProcessRequest{
		Command: []string{"sh", "-c", cmdStr},
		Timeout: timeoutStr,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(targetActorHeader, actorRefStr)

	tlsCfg := &tls.Config{InsecureSkipVerify: true}
	if cert, err := tls.LoadX509KeyPair("/run/podidentity.podcert.ate.dev/credential-bundle.pem", "/run/podidentity.podcert.ate.dev/credential-bundle.pem"); err == nil {
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	httpCli := &http.Client{
		Timeout:   15 * time.Minute,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}
	resp, err := httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out ProcessResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (raw: %s)", err, string(body))
	}
	return &out, nil
}

func main() {
	mode := flag.String("mode", "resume-and-run", "Operation mode: probe | release-image | ensure-template | delete-template | create-boot-run | resume-and-run | exec | suspend | delete")
	adapterName := flag.String("adapter", "", "Adapter label for output")
	scenarioName := flag.String("scenario", "", "Scenario label for output")
	workloadName := flag.String("workload", "", "Workload label (small|medium|large)")
	cmdStr := flag.String("cmd", "", "Workload shell command to execute")
	atespace := flag.String("atespace", "ate-bench-stream", "Atespace name")
	templateName := flag.String("template", "", "ActorTemplate name")
	actorName := flag.String("actor", "", "Actor name")
	imageRef := flag.String("image", "", "Container image reference with digest")
	poolLabel := flag.String("pool-label", "", "WorkerPool workload label value (for ensure-template)")
	ateapiAddr := flag.String("ateapi", "api.ate-system.svc.cluster.local:443", "ateapi gRPC address")
	atenetAddr := flag.String("atenet", "atenet-router.ate-system.svc.cluster.local:443", "atenet-router HTTPS address")
	cleanPycache := flag.Bool("clean-pycache", true, "Clean /testbed __pycache__ before suspending")
	waitGolden := flag.Bool("wait-golden", false, "Wait for golden snapshot to finish in ensure-template")
	accelSocket := flag.String("socket", "/run/ate/image-accelerator.sock", "ImageAccelerator Unix socket for probe / release-image")
	bypassTTL := flag.Bool("bypass-ttl", true, "Pass bypass_ttl=true in probe / release-image")
	ateToken := flag.String("token", "", "ateapi bearer token")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	if *mode == "probe" {
		if *imageRef == "" {
			log.Fatal("--image is required for probe")
		}
		client, err := imageaccel.NewGRPCClient(*accelSocket)
		if err != nil {
			log.Fatalf("NewGRPCClient: %v", err)
		}
		defer client.Close()

		req := &imageaccel.PrepareRequest{ImageRef: *imageRef}
		canStart := time.Now()
		ok, err := client.CanProvide(ctx, req)
		canDur := time.Since(canStart)
		if err != nil {
			log.Fatalf("CanProvide: %v", err)
		}
		report := ProbeReport{
			Provider:     client.Name(),
			ImageRef:     *imageRef,
			CanProvide:   ok,
			CanProvideMs: float64(canDur.Microseconds()) / 1000.0,
		}
		if ok {
			prepStart := time.Now()
			res, err := client.PrepareImage(ctx, req)
			prepDur := time.Since(prepStart)
			if err != nil {
				log.Fatalf("PrepareImage: %v", err)
			}
			report.Provider = client.Name()
			report.ImageDigest = res.ImageDigest
			report.LayerDirs = res.LayerDirs
			report.PrepareMs = float64(prepDur.Microseconds()) / 1000.0
			for i, dir := range res.LayerDirs {
				if _, err := os.ReadDir(filepath.Join(dir, "fs")); err != nil {
					log.Fatalf("reading layer %d (%s/fs): %v", i, dir, err)
				}
				report.VerifiedLayers++
			}
			relStart := time.Now()
			if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{
				ImageRef:  *imageRef,
				BypassTTL: *bypassTTL,
			}); err != nil {
				log.Fatalf("ReleaseImage: %v", err)
			}
			report.ReleaseMs = float64(time.Since(relStart).Microseconds()) / 1000.0
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}

	if *mode == "release-image" {
		if *imageRef == "" {
			log.Fatal("--image is required for release-image")
		}
		client, err := imageaccel.NewGRPCClient(*accelSocket)
		if err != nil {
			log.Fatalf("NewGRPCClient: %v", err)
		}
		defer client.Close()
		if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{
			ImageRef:  *imageRef,
			BypassTTL: *bypassTTL,
		}); err != nil {
			log.Fatalf("ReleaseImage: %v", err)
		}
		return
	}

	cli, conn, err := dialAteAPI(*ateapiAddr, *ateToken)
	if err != nil {
		log.Fatalf("dialAteAPI: %v", err)
	}
	defer conn.Close()

	switch *mode {
	case "ensure-template":
		if *templateName == "" || *imageRef == "" || *poolLabel == "" {
			log.Fatal("--template, --image, and --pool-label are required for ensure-template")
		}
		_, err := cli.CreateAtespace(ctx, &ateapipb.CreateAtespaceRequest{
			Atespace: &ateapipb.Atespace{
				Metadata: &ateapipb.ResourceMetadata{Name: *atespace},
			},
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			log.Fatalf("CreateAtespace(%s) failed: %v", *atespace, err)
		}
		tmpl, err := cli.CreateActorTemplate(ctx, &ateapipb.CreateActorTemplateRequest{
			ActorTemplate: &ateapipb.ActorTemplate{
				Metadata: &ateapipb.ResourceMetadata{
					Atespace: *atespace,
					Name:     *templateName,
				},
				WorkerSelector: &ateapipb.Selector{
					MatchLabels: map[string]string{
						"workload": *poolLabel,
					},
				},
				Containers: []*ateapipb.Container{
					{
						Name:    "sandbox",
						Image:   *imageRef,
						Command: []string{"/ko-app/sandbox"},
						Env: []*ateapipb.EnvVar{
							{Name: "PORT", Value: "80"},
						},
					},
				},
				Resources: &ateapipb.ResourceRequirements{
					Limits: &ateapipb.Resources{
						CpuMillicores: 1000,
						MemoryBytes:   2 * 1024 * 1024 * 1024,
					},
				},
				SnapshotConfig: &ateapipb.SnapshotConfig{
					PreferredFidelity: ateapipb.SnapshotFidelity_SNAPSHOT_FIDELITY_MEMORY,
				},
			},
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			log.Fatalf("CreateActorTemplate(%s) failed: %v", *templateName, err)
		}
		if err == nil {
			fmt.Printf("Created ActorTemplate %s/%s (uid=%s)\n", *atespace, *templateName, tmpl.GetMetadata().GetUid())
		}
		if *waitGolden {
			for {
				t, err := cli.GetActorTemplate(ctx, &ateapipb.GetActorTemplateRequest{
					ActorTemplate: &ateapipb.ObjectRef{Atespace: *atespace, Name: *templateName},
				})
				if err != nil {
					log.Fatalf("GetActorTemplate: %v", err)
				}
				gs := t.GetStatus().GetGoldenSnapshotStatus()
				if gs.GetErrorMessage() != "" {
					log.Fatalf("Golden snapshot failed: %s", gs.GetErrorMessage())
				}
				if gs.GetGoldenTag().GetName() != "" {
					fmt.Printf("Golden snapshot ready: %s\n", gs.GetGoldenTag().GetName())
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
		return

	case "delete-template":
		if *templateName == "" {
			log.Fatal("--template is required for delete-template")
		}
		_, err := cli.DeleteActorTemplate(ctx, &ateapipb.DeleteActorTemplateRequest{
			ActorTemplate: &ateapipb.ObjectRef{Atespace: *atespace, Name: *templateName},
		})
		if err != nil && status.Code(err) != codes.NotFound {
			log.Fatalf("DeleteActorTemplate(%s) failed: %v", *templateName, err)
		}
		fmt.Printf("Deleted ActorTemplate %s/%s\n", *atespace, *templateName)
		return
	}

	if *actorName == "" {
		log.Fatal("--actor is required")
	}
	actorRefStr := fmt.Sprintf("%s/%s", *atespace, *actorName)
	actorObjRef := &ateapipb.ObjectRef{Atespace: *atespace, Name: *actorName}

	switch *mode {
	case "create-boot-run", "resume-and-run":
		if *cmdStr == "" {
			log.Fatal("--cmd is required for run modes")
		}
		if *mode == "create-boot-run" {
			if *templateName == "" {
				log.Fatal("--template is required for create-boot-run")
			}
			_, err := cli.CreateActor(ctx, &ateapipb.CreateActorRequest{
				Actor: &ateapipb.Actor{
					Metadata: &ateapipb.ResourceMetadata{
						Atespace: *atespace,
						Name:     *actorName,
					},
					ActorTemplate: &ateapipb.ObjectRef{
						Atespace: *atespace,
						Name:     *templateName,
					},
				},
			})
			if err != nil {
				log.Fatalf("CreateActor(%s) failed: %v", *actorName, err)
			}
		}

		wrappedCmd := fmt.Sprintf(`t0=$(date +%%s%%N); %s; rc=$?; t1=$(date +%%s%%N); echo ""; echo "___BENCH_TIMING___ t0=$t0 t1=$t1 rc=$rc"`, *cmdStr)

		tResumeStart := time.Now()
		resumeResp, err := cli.ResumeActor(ctx, &ateapipb.ResumeActorRequest{
			Actor: actorObjRef,
		})
		if err != nil {
			log.Fatalf("ResumeActor(%s) failed: %v", *actorName, err)
		}
		tResumeDone := time.Now()

		var procResp *ProcessResponse
		var execErr error
		for attempt := 0; attempt < 5; attempt++ {
			procResp, execErr = execInActor(ctx, *atenetAddr, actorRefStr, wrappedCmd, "600s")
			if execErr == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		tHTTPDone := time.Now()
		if execErr != nil {
			log.Fatalf("execInActor(%s) failed: %v", *actorName, execErr)
		}

		var t0, t1 int64
		var rc int
		matches := timingRegex.FindStringSubmatch(procResp.Stdout)
		if len(matches) == 4 {
			t0, _ = strconv.ParseInt(matches[1], 10, 64)
			t1, _ = strconv.ParseInt(matches[2], 10, 64)
			rc, _ = strconv.Atoi(matches[3])
		} else {
			log.Fatalf("missing ___BENCH_TIMING___ in stdout: %q (stderr: %q)", procResp.Stdout, procResp.Stderr)
		}

		cleanOut := strings.TrimSpace(timingRegex.ReplaceAllString(procResp.Stdout, ""))
		if len(cleanOut) > 240 {
			cleanOut = fmt.Sprintf("%s ... (total %d bytes)", cleanOut[:120], len(cleanOut))
		}

		workerPod := ""
		snapshotURI := ""
		if a := resumeResp.GetActor(); a != nil {
			workerPod = a.GetStatus().GetWorkerAssignment().GetWorkerPod()
			for _, s := range a.GetStatus().GetSnapshots() {
				if uri := s.GetDurableSnapshot().GetObject().GetSnapshotUri(); uri != "" {
					snapshotURI = uri
				}
			}
		}

		res := BenchResult{
			Adapter:          *adapterName,
			Scenario:         *scenarioName,
			Workload:         *workloadName,
			ActorRef:         actorRefStr,
			WorkerPod:        workerPod,
			SnapshotURI:      snapshotURI,
			ResumeRPCStartNs: tResumeStart.UnixNano(),
			ResumeRPCDoneNs:  tResumeDone.UnixNano(),
			ResumeRPCDurMs:   float64(tResumeDone.Sub(tResumeStart).Nanoseconds()) / 1e6,
			CmdT0Ns:          t0,
			CmdT1Ns:          t1,
			HTTPDoneNs:       tHTTPDone.UnixNano(),
			ClientTTCEMs:     float64(t0-tResumeStart.UnixNano()) / 1e6,
			WorkloadRunMs:    float64(t1-t0) / 1e6,
			ClientE2EMs:      float64(t1-tResumeStart.UnixNano()) / 1e6,
			ExitCode:         rc,
			StdoutPreview:    cleanOut,
		}
		enc, _ := json.Marshal(res)
		fmt.Println(string(enc))

	case "exec":
		if *cmdStr == "" {
			log.Fatal("--cmd is required for exec")
		}
		procResp, err := execInActor(ctx, *atenetAddr, actorRefStr, *cmdStr, "600s")
		if err != nil {
			log.Fatalf("execInActor(%s) failed: %v", *actorName, err)
		}
		enc, _ := json.Marshal(procResp)
		fmt.Println(string(enc))

	case "suspend":
		if *cleanPycache {
			_, _ = execInActor(ctx, *atenetAddr, actorRefStr, `find /testbed -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true`, "30s")
		}
		t0 := time.Now()
		resp, err := cli.SuspendActor(ctx, &ateapipb.SuspendActorRequest{
			Actor: actorObjRef,
			Options: &ateapipb.SuspendOptions{
				Mode: ateapipb.DurabilityMode_DURABILITY_MODE_SYNCHRONOUS,
			},
		})
		if err != nil {
			log.Fatalf("SuspendActor(%s) failed: %v", *actorName, err)
		}
		durMs := float64(time.Since(t0).Nanoseconds()) / 1e6
		snapURI := ""
		for _, s := range resp.GetActor().GetStatus().GetSnapshots() {
			if uri := s.GetDurableSnapshot().GetObject().GetSnapshotUri(); uri != "" {
				snapURI = uri
			}
		}
		fmt.Printf(`{"action":"suspend","actor_ref":%q,"suspend_dur_ms":%.2f,"snapshot_uri":%q}`+"\n", actorRefStr, durMs, snapURI)

	case "revert":
		t0 := time.Now()
		resp, err := cli.RevertActor(ctx, &ateapipb.RevertActorRequest{
			Actor: actorObjRef,
		})
		if err != nil {
			log.Fatalf("RevertActor(%s) failed: %v", *actorName, err)
		}
		durMs := float64(time.Since(t0).Nanoseconds()) / 1e6
		snapURI := ""
		for _, s := range resp.GetActor().GetStatus().GetSnapshots() {
			if uri := s.GetDurableSnapshot().GetObject().GetSnapshotUri(); uri != "" {
				snapURI = uri
			}
		}
		fmt.Printf(`{"action":"revert","actor_ref":%q,"revert_dur_ms":%.2f,"snapshot_uri":%q}`+"\n", actorRefStr, durMs, snapURI)

	case "delete":
		_, _ = cli.SuspendActor(ctx, &ateapipb.SuspendActorRequest{Actor: actorObjRef})
		_, err := cli.DeleteActor(ctx, &ateapipb.DeleteActorRequest{
			Actor: actorObjRef,
		})
		if err != nil && status.Code(err) != codes.NotFound {
			log.Fatalf("DeleteActor(%s) failed: %v", *actorName, err)
		}
		fmt.Printf(`{"action":"delete","actor_ref":%q}`+"\n", actorRefStr)

	default:
		log.Fatalf("unknown --mode=%s", *mode)
	}
}
