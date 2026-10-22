// Package dockerx is the only package that talks to the Docker Engine API.
// It wraps the official SDK behind small types owned by Dockerity.
package dockerx

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// ServerInfo describes the connected Docker engine.
type ServerInfo struct {
	Version    string
	APIVersion string
	OS         string
	Arch       string
}

// Client is a Docker Engine connection.
type Client struct {
	api *client.Client
}

// New connects using the standard environment (DOCKER_HOST etc.), defaulting
// to the local socket, with API version negotiation.
func New() (*Client, error) {
	api, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Client{api: api}, nil
}

// Close releases the underlying connection.
func (c *Client) Close() error {
	return c.api.Close()
}

// Ping checks the engine is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.api.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true})
	return err
}

// ServerInfo returns the engine version details.
func (c *Client) ServerInfo(ctx context.Context) (ServerInfo, error) {
	v, err := c.api.ServerVersion(ctx, client.ServerVersionOptions{})
	if err != nil {
		return ServerInfo{}, err
	}
	return ServerInfo{Version: v.Version, APIVersion: v.APIVersion, OS: v.Os, Arch: v.Arch}, nil
}
