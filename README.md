Sponsored by [Fortworx](https://fortworx.com)
<div>
  <img alt="Fortworx Logo" src="https://cdn.fortworx.com/images/logo_full_dark.png" width="300"/>
</div>

# Tailbone

Tailbone is a JWT issuer that uses Tailscale as identity provider.

## What is it for?
If you need to identify callers to your services you can use Tailbone to do so. JWTs issued by Tailbone are signed with RSA keys and can be verified by any service that has access to the JWKS endpoint. This means the service does not require access to any shared secret, database or to be part of a VPN. 

## How does it work?

1. The client (caller) makes a call to Tailbone in a Tailscale network.
2. Tailbone verifies the client's identity using Tailscale and issues a JWT token that contains the client's identity.
3. The client uses the JWT token to make calls to your services.
4. Your services verify the JWT token using the JWKS endpoint which usually is publicly accessible.
5. If the token is valid, the service can trust the caller's identity.

## Why not use Tailscale directly?
You can, if both your client and service are part of your VPN then you can use Tailscale, although you will still need to obtain the identity of the caller via a Tailscale client.

## Features

- JWT-based authentication using RSA key pairs
- Embedded Tailscale integration for user verification (no need for Tailscale client running on the server).
- Management of JWKS keys in S3 so the services can verify the JWT tokens.

## Installation

You can install Tailbone using `go install github.com/altacoda/tailbone@latest` or download a release from the [releases page](https://github.com/altacoda/tailbone/releases).
You can also use the Dockerfile to build and run Tailbone in a container environment. Since Tailbone has a Tailscale service embedded in it, you don't need to install Tailscale on the host machine or run Tailbone with elevated privileges.

## Setup
You need two things to get started:

- A Tailscale auth key
- A S3 bucket to store the JWKS.

You can get a Tailscale auth key from the [Tailscale dashboard](https://login.tailscale.com/admin/settings/authkey).

Tailbone uses an S3 compatible API to store the JWKS. The JWKS file is stored at a location specified by the `key-path` configuration option (default: ".well-known/jwks.json"). Your services will need to be able to access this endpoint to verify the tokens.

### What is S3 for?
Your services will need to verify the JWT tokens issued by Tailbone. These are verifiable using a JWKS (JSON Web Key Set) that contains the public keys used to sign the tokens. Tailbone manages a JWKs file that contains the public keys used to sign the JWT tokens. This file is stored in an S3 bucket. You can then make this bucket publicly accessible so your services can verify the JWT tokens, usually with a URL like `https://<bucket>.s3.amazonaws.com/.well-known/jwks.json`.

## Usage

> A Note on environment variables 
> All parameters in Tailbone are configurable via command line parameters or environment variables (see below for details).


### Server Mode
Start the server with your Tailscale auth key.
```bash
tailbone server start --ts-authkey <tailscale-auth-key>
```

### Housekeeping
You can schedule running Tailbone housekeeping to ensure private keys stored locally are in sync with the public ones on S3.

```bash
tailbone server housekeeping
```

### Client Mode
Tailbone CLI can be used as a management client for Tailbone.


> A Note on Tailbone Admin API
> Tailbone server runs a gRPC API admin on port 50051 that is used by the Tailbone CLI for management. This API is open to the Tailscale network and access to it should be managed using Tailscale ACLs.

Generate a new key pair.
```bash
tailbone keys generate
```

Remove a key

> IMPORTANT: Removing a key will invalidate all tokens signed with that key. This is a destructive operation and should be used with caution.

```bash
tailbone keys remove <keyID>
```

List the keys in S3.
```bash 
tailbone keys list
```

## API Endpoints

> Tailbone is built to run on Tailscale network and doesn't use HTTPs. Do not expose it on a public network!

Tailbone has two endpoints:

- `/_healthz`: Health check endpoint (GET)
- `/issue`: Token issue endpoint (POST)

### Health Check Endpoint

The health check endpoint is used to check if the server is running.

```bash
curl http://<IP>/_healthz
```

### Token Endpoint

The token endpoint is used to issue a JWT token for a given Tailscale user.

```bash
curl -X POST http://<IP>/issue
```

This will a JSON response with the following fields:

- `token`: The issued JWT token

This token is signed with the most recent key found in the `dir` directory.

## Configuration

Tailbone can be configured using:
- Configuration file (`$HOME/.tailbone.toml`)
- Command line flags
- Environment variables (prefixed with `TB_`)

### Configuration Reference

#### Global Server Configuration
| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--log-level` | `TB_LOG_LEVEL` | "info" | Log level (trace, debug, info, warn, error) |
| `--log-format` | `TB_LOG_FORMAT` | "console" | Log format (console, json) |
| `--dir` | `TB_KEYS_DIR` | "keys" | Directory containing the JWK files |
| `--bucket` | `TB_KEYS_BUCKET` | | S3 bucket for JWKS storage |
| `--key-path` | `TB_KEYS_KEYPATH` | ".well-known/jwks.json" | Path/key for the JWKS file in S3 |

#### Server Start Configuration
| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `TB_SERVER_PORT` | 80 | Port to run the issuer server on |
| `--binding` | `TB_SERVER_BINDING` | "auto" | Binding address for the issuer server |
| `--ts-authkey` | `TB_SERVER_TAILSCALE_AUTHKEY` | | Tailscale auth key |
| `--ts-join-timeout` | `TB_SERVER_TAILSCALE_JOINTIMEOUT` | 60s | Time to wait for Tailscale to join the network |
| `--ts-join-retry` | `TB_SERVER_TAILSCALE_JOINRETRY` | 1s | Interval between join attempts |
| `--ts-dir` | `TB_SERVER_TAILSCALE_DIR` | ".tsnet" | Tailscale state directory |
| `--ts-hostname` | `TB_SERVER_TAILSCALE_HOSTNAME` | "tailbone" | Tailscale hostname |
| `--issuer` | `TB_KEYS_ISSUER` | "tailbone" | Issuer name for JWT tokens |
| `--expiry` | `TB_KEYS_EXPIRY` | 20m | Token expiry duration |
| `--admin-binding` | `TB_ADMIN_BINDING` | "auto" | Admin server binding address |
| `--admin-port` | `TB_ADMIN_PORT` | 50051 | Admin server port |
| `--components` | `TB_COMPONENTS` | ["issuer", "admin"] | Components to start |

#### Client Configuration
| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--host` | `TB_ADMIN_CLIENT_HOST` | | Tailbone server host |
| `--port` | `TB_ADMIN_CLIENT_PORT` | | Tailbone server port |

#### Key Generation Configuration
| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--size` | `TB_KEYS_SIZE` | 2048 | RSA key size in bits |

#### Additional Configuration Options
| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `TB_KEY_PREFIX` | "tb" | Prefix used for key IDs when generating new keys |
| `TB_LOG_NO_COLOR` | false | Disable colored output in console log format |

> Do not change `TB_KEY_PREFIX` on a system with existing keys. This will break compatibility with existing tokens. A valid prefix should not include any special characters or spaces.

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

#### `server start`
Starts the Tailbone identity server with the following options:

- `-p, --port`: Port to run the issuer server on (default: 80)
- `-b, --binding`: Binding address for the issuer server (default: "auto")
- `--ts-authkey`: Tailscale auth key
- `--ts-join-timeout`: Time to wait for Tailscale to join the network (default: 60s)
- `--ts-join-retry`: Interval between join attempts (default: 1s)
- `--ts-dir`: Tailscale state directory (default: ".tsnet")
- `--ts-hostname`: Tailscale hostname (default: "tailbone")
- `--issuer`: Issuer name for JWT tokens (default: "tailbone")
- `--expiry`: Token expiry duration (default: 20m)
- `--admin-binding`: Admin server binding address (default: "auto")
- `--admin-port`: Admin server port (default: 50051)
- `--components`: Components to start (default: ["issuer", "admin"])

> The `auto` binging address means that the server will bind only to the Tailscale network interface. This is the default behavior.

#### `server housekeeping`
Runs maintenance tasks to ensure private keys stored locally are in sync with public keys on S3. No additional parameters beyond global flags.

### Global Server Flags
These flags apply to all server commands:

- `--log-level`: Log level (trace, debug, info, warn, error) (default: "info")
- `--log-format`: Log format (console, json) (default: "console")
- `--dir`: Directory containing the JWK files (default: "keys")
- `--bucket`: S3 bucket for JWKS storage
- `--key-path`: Path/key for the JWKS file in S3 (default: ".well-known/jwks.json")

### Global Flags (client mode)
- `--host`: Tailbone server host
- `--port`: Tailbone server port

### Key Management Commands

#### `keys generate`
Generate a new RSA key pair for signing JWTs. The keys will be saved in JWK format.

Flags:
- `-s, --size`: RSA key size in bits (default: 2048)

Example:
```bash
tailbone keys generate --size 4096
```

#### `keys list`
List all available signing keys from the JWKS endpoint.

Flags:
- `--local`: List keys from local filesystem instead of S3 (default: false)

Example:
```bash
tailbone keys list
```

#### `keys remove [keyID]`
Remove a key from the JSON Web Key Set (JWKS) stored in S3.

Example:
```bash
tailbone keys remove key_12345
```

Use `--yes` to skip the confirmation prompt.

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
