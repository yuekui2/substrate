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

// Package mock provides an in-memory mock implementation of imagepluginclient.Client
// for unit testing atelet without a live ImagePlugin daemon.
package mock

import (
	"context"
	"sync"

	"go.ate.dev/substrate/api/imageplugin/v1"
	"go.ate.dev/substrate/internal/imagepluginclient"
)

// Client is a configurable mock implementation of imagepluginclient.Client.
type Client struct {
	mu sync.Mutex

	ProviderName string
	// PluginState is returned by State. The zero value means SERVING.
	PluginState      imageplugin.PluginState
	CanProvideFunc   func(ctx context.Context, req *imagepluginclient.PrepareRequest) (bool, error)
	StateFunc        func(ctx context.Context) (imageplugin.PluginState, error)
	PrepareImageFunc func(ctx context.Context, req *imagepluginclient.PrepareRequest) (*imagepluginclient.PrepareResult, error)
	ReleaseImageFunc func(ctx context.Context, req *imagepluginclient.ReleaseRequest) error
	ReconcileFunc    func(ctx context.Context, active []*imagepluginclient.ActiveImageLayers) error

	PrepareCalls   []*imagepluginclient.PrepareRequest
	ReleaseCalls   []*imagepluginclient.ReleaseRequest
	ReconcileCalls [][]*imagepluginclient.ActiveImageLayers
}

var _ imagepluginclient.Client = (*Client)(nil)

// New returns a new mock Client with default no-op handlers.
func New(providerName string) *Client {
	if providerName == "" {
		providerName = "mock"
	}
	return &Client{ProviderName: providerName}
}

// Name returns the mock provider name.
func (m *Client) Name() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ProviderName != "" {
		return m.ProviderName
	}
	return "mock"
}

// CanProvide delegates to CanProvideFunc if set, or returns true by default.
func (m *Client) CanProvide(ctx context.Context, req *imagepluginclient.PrepareRequest) (bool, error) {
	if m.CanProvideFunc != nil {
		return m.CanProvideFunc(ctx, req)
	}
	return true, nil
}

// State delegates to StateFunc if set, or returns m.PluginState, with
// PLUGIN_STATE_UNSPECIFIED reported as PLUGIN_STATE_SERVING.
func (m *Client) State(ctx context.Context) (imageplugin.PluginState, error) {
	m.mu.Lock()
	fn := m.StateFunc
	state := m.PluginState
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx)
	}
	if state == imageplugin.PluginState_PLUGIN_STATE_UNSPECIFIED {
		return imageplugin.PluginState_PLUGIN_STATE_SERVING, nil
	}
	return state, nil
}

// PrepareImage records the call and delegates to PrepareImageFunc.
func (m *Client) PrepareImage(ctx context.Context, req *imagepluginclient.PrepareRequest) (*imagepluginclient.PrepareResult, error) {
	m.mu.Lock()
	m.PrepareCalls = append(m.PrepareCalls, req)
	fn := m.PrepareImageFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return nil, imagepluginclient.ErrNotSupported
}

// ReleaseImage records the call and delegates to ReleaseImageFunc.
func (m *Client) ReleaseImage(ctx context.Context, req *imagepluginclient.ReleaseRequest) error {
	m.mu.Lock()
	m.ReleaseCalls = append(m.ReleaseCalls, req)
	fn := m.ReleaseImageFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return nil
}

// ReconcileImages records the call and delegates to ReconcileFunc.
func (m *Client) ReconcileImages(ctx context.Context, active []*imagepluginclient.ActiveImageLayers) error {
	m.mu.Lock()
	m.ReconcileCalls = append(m.ReconcileCalls, active)
	fn := m.ReconcileFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, active)
	}
	return nil
}

// Close is a no-op for mock.
func (m *Client) Close() error {
	return nil
}
