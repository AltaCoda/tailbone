# Tailbone

Tailbone is an identity provider based on JWT that uses Tailscale for authentication. 

## What is it for?
Tailscale offer an easy way to setup a secure VPN. It also has built-in authentication mechanism so any client can be safely authenticated with Tailscale.

Tailbone uses this feature to authenticate any inbound calls it receives over a Tailscale network and issue a JWT token for the caller. This JWT token includes the Tailscale user identity (or server's tag). The caller than can use this token to authenticate with other services that are not behind Tailscale.

The tokens are issued with RSA keys that are verifiable via a JWKS endpoint. This means no secrets are shared between the caller and the server.

## Features

- JWT-based authentication using RSA key pairs
- Tailscale integration for user verification
- Key management system with local and S3 storage

## Installation

You can install Tailbone using `go install github.com/tailbone-io/tailbone@latest` or download a release from the [releases page](https://github.com/tailbone-io/tailbone/releases).

## Setup
You need two things to get started:

- A Tailscale auth key
- A S3 bucket to store the JWKS.

You can get a Tailscale auth key from the [Tailscale dashboard](https://login.tailscale.com/admin/settings/authkey).

Tailbone uses an S3 compatible API to store the JWKS. The JWKS file is stored at a location specified by the `key-path` configuration option (default: ".well-known/jwks.json"). Your services will need to be able to access this endpoint to verify the tokens.

## Usage

Start the server with your Tailscale auth key.
```bash
tailbone server start --ts-authkey <tailscale-auth-key>
```

Generate a new key pair. The key pair will be stored in the `dir` directory.
```bash
tailbone keys generate
```

Upload the public key to S3. If there are other keys at the endpoint, they will be merged with the new key.
```bash
tailbone keys upload <keyID>
```

Remove a key from the JWKS in S3.
```bash
tailbone keys remove <keyID>
```

List the keys in S3.
```bash 
tailbone keys list
```

List the keys in the `dir` directory.
```bash
tailbone keys list --local
```

## API Endpoints

Tailbone has two endpoints:

- `/_healthz`: Health check endpoint (GET)
- `/issue`: Token issue endpoint (POST)

### Health Check Endpoint

The health check endpoint is used to check if the server is running.

```bash
curl http://localhost:80/_healthz
```

### Token Endpoint

The token endpoint is used to issue a JWT token for a given Tailscale user.

```bash
curl -X POST http://localhost:80/issue
```

This will a JSON response with the following fields:

- `token`: The issued JWT token

This token is signed with the most recent key found in the `dir` directory.

## Configuration

Tailbone can be configured using:
- Configuration file (`$HOME/.tailbone.toml`)
- Command line flags
- Environment variables (prefixed with `TB_`)

## AWS Configuration for S3

For S3 key storage functionality, Tailbone uses the AWS SDK default configuration chain. You'll need to configure AWS credentials using one of these methods:

- Environment variables:
  - `AWS_ACCESS_KEY_ID`: Your AWS access key
  - `AWS_SECRET_ACCESS_KEY`: Your AWS secret key
  - `AWS_REGION`: AWS region for S3 operations
  - `AWS_SESSION_TOKEN`: (Optional) AWS session token if using temporary credentials

- Or use other standard AWS configuration methods:
  - AWS credentials file (`~/.aws/credentials`)
  - IAM roles when running on AWS services

The following configuration values are required for S3 operations:
- `keys.bucket`: S3 bucket name for JWKS storage
- `keys.keyPath`: Path/key for the JWKS file in S3 (default: ".well-known/jwks.json")

## Commands

### Server Commands

- `server start`: Start the Tailbone identity server
  - `-p, --port`: Port to run the server on (default: 80)
  - `-H, --host`: Host address to bind to (default: "0.0.0.0")
  - `--ts-authkey`: Tailscale auth key
  - `--log-level`: Log level (trace, debug, info, warn, error)
  - `--log-format`: Log format (console, json)
  - `--dir`: Directory containing the JWK files (default: "keys")
  - `--issuer`: Issuer name for JWT tokens (default: "tailbone")
  - `--expiry`: Token expiry duration (default: 20m)
  - `--ts-dir`: Tailscale state directory (default: ".tsnet")
  - `--ts-hostname`: Tailscale hostname (default: "tailbone")
  - `--log.no_color`: Disable color in the logs

### Key Management Commands

- `keys generate`: Generate a new signing key pair
  - `-s, --size`: RSA key size in bits (default: 2048)

- `keys list`: List available signing keys
  - `-l, --local`: List keys from local filesystem instead of remote JWKS

- `keys upload [keyID]`: Upload a public key to S3 as JWKS
  - Requires the key to exist in the keys directory
  - Will merge with existing JWKS if present

- `keys remove [keyID]`: Remove a key from the JWKS in S3
  - Requires `--bucket` flag to be set

### Global Flags

- `--config`: Config file (default: $HOME/.tailbone.*)
- `--version`: Print version information

### Key Management Global Flags

- `--dir`: Directory containing the keys (default: "keys")
- `--bucket`: S3 bucket for JWKS storage
- `--key-path`: Path/key for the JWKS file in S3 (default: ".well-known/jwks.json")

## Contributing

We welcome contributions to Tailbone! Here's how you can help:

### Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/altacoda/tailbone`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`

### Submitting Changes

1. Push to your fork: `git push origin feature/your-feature-name`
2. Open a Pull Request from your fork to our main repository
3. Ensure your PR description clearly describes the problem and solution
4. Include any relevant issue numbers

### Code Guidelines

- Follow Go best practices and conventions
- Add tests for new features
- Update documentation as needed
- Keep commits atomic and well-described

### Need Help?

- Open an issue for bugs or feature requests
- Check existing issues and PRs before creating new ones
