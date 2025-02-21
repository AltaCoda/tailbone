package core

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/altacoda/tailbone/proto"
	"github.com/altacoda/tailbone/utils"
)

// AdminServer implements the AdminServiceServer interface
type AdminServer struct {
	proto.UnimplementedAdminServiceServer
	cloudConnector  utils.CloudConnector
	localKeyStorage utils.ILocalKeyStorage
	grpcServer      *grpc.Server
	logger          zerolog.Logger
	done            chan struct{}
}

// NewAdminServer creates a new instance of AdminServer
func NewAdminServer(ctx context.Context) (*AdminServer, error) {
	// Configure logger
	logger := utils.GetLogger("admin-server")
	logger.Info().Msg("initializing admin server")

	// Create S3 connector
	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return nil, err
	}

	localKeyStorage := utils.NewLocalKeyStorage()

	return &AdminServer{
		cloudConnector:  cloudConnector,
		localKeyStorage: localKeyStorage,
		grpcServer:      grpc.NewServer(),
		logger:          logger,
		done:            make(chan struct{}),
	}, nil
}

func (s *AdminServer) Start() error {
	addr := viper.GetString("admin.address")
	s.logger.Info().Str("addr", addr).Msg("creating listener")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	proto.RegisterAdminServiceServer(s.grpcServer, s)

	go func() {
		<-s.done
		s.logger.Info().Msg("received shutdown signal")
		s.grpcServer.GracefulStop()
	}()

	s.logger.Info().Str("addr", addr).Msg("starting admin server")
	return s.grpcServer.Serve(lis)
}

func (s *AdminServer) Stop() {
	s.logger.Info().Msg("stopping admin server")
	close(s.done)
}

// GenerateNewKeys implements the GenerateNewKeys RPC method
func (s *AdminServer) GenerateNewKeys(ctx context.Context, req *proto.GenerateNewKeysRequest) (*proto.GenerateNewKeysResponse, error) {
	s.logger.Info().Msg("generating new key pair")
	tokenGenerator := utils.NewTokenGenerator(s.cloudConnector, s.localKeyStorage)

	// Generate the key pair
	keyPair, err := tokenGenerator.GenerateKeyPair(ctx, viper.GetInt("keys.size"))
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate key pair")
		return nil, err
	}

	// Save the key pair locally
	if err := tokenGenerator.SaveLocally(ctx, keyPair, viper.GetString("keys.dir")); err != nil {
		s.logger.Error().Err(err).Msg("failed to save key pair locally")
		return nil, err
	}

	// Get bucket and key path for upload
	bucket, keyPath, err := s.cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get bucket and key path")
		return nil, fmt.Errorf("failed to get bucket and key path: %w", err)
	}

	// Try to download existing JWKS
	var existingJWKS *utils.JWKS
	data, err := s.cloudConnector.Download(ctx, bucket, keyPath)
	if err != nil {
		// If there's an error (like 404), start with empty JWKS
		existingJWKS = &utils.JWKS{
			Keys: []jwk.Key{},
		}
	} else {
		existingJWKS, err = tokenGenerator.ParseJWKS(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse existing JWKS: %w", err)
		}
	}

	// Check if key with same ID already exists
	keyExists := false
	for i, key := range existingJWKS.Keys {
		kid, _ := key.Get(jwk.KeyIDKey)
		if kid == keyPair.KeyID {
			// Replace existing key
			existingJWKS.Keys[i] = keyPair.PublicKey
			keyExists = true
			break
		}
	}

	// Append new key if not found
	if !keyExists {
		existingJWKS.Keys = append(existingJWKS.Keys, keyPair.PublicKey)
	}

	// Upload combined JWKS to S3
	if err := tokenGenerator.UploadPublicKey(ctx, existingJWKS, bucket, keyPath); err != nil {
		return nil, fmt.Errorf("failed to upload key to S3: %w", err)
	}

	s.logger.Info().Str("key_id", keyPair.KeyID).Msg("successfully generated and stored new key pair")

	return &proto.GenerateNewKeysResponse{
		KeyId: keyPair.KeyID,
	}, nil
}

// ListKeys implements the ListKeys RPC method
func (s *AdminServer) ListKeys(ctx context.Context, req *proto.ListKeysRequest) (*proto.ListKeysResponse, error) {
	s.logger.Info().Msg("listing keys")
	return s.listRemoteKeys(ctx)
}

func (s *AdminServer) listRemoteKeys(ctx context.Context) (*proto.ListKeysResponse, error) {
	tokenGenerator := utils.NewTokenGenerator(s.cloudConnector, s.localKeyStorage)

	// Check if we have S3 bucket configured
	if viper.GetString("keys.bucket") == "" {
		s.logger.Error().Msg("keys.bucket is not configured")
		return nil, fmt.Errorf("keys.bucket is required")
	}

	// Get bucket and key path
	bucket, keyPath, err := s.cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get bucket and key path")
		return nil, err
	}

	// Download JWKS
	jwks, err := tokenGenerator.DownloadJWKS(ctx, bucket, keyPath)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to download JWKS")
		return nil, err
	}

	var keys []*proto.KeyInfo

	for _, key := range jwks.Keys {
		keyInfo := &proto.KeyInfo{}

		// Extract key metadata
		if kid, ok := key.Get(jwk.KeyIDKey); ok {
			keyInfo.KeyId = kid.(string)
			// Extract timestamp from key ID (assuming format "key-{timestamp}")
			if parts := strings.Split(keyInfo.KeyId, "-"); len(parts) > 1 {
				if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					keyInfo.CreatedAt = ts
				}
			}
		}

		keyInfo.Algorithm = key.Algorithm().String()

		keys = append(keys, keyInfo)
	}

	s.logger.Info().Int("key_count", len(jwks.Keys)).Msg("successfully retrieved keys")
	return &proto.ListKeysResponse{
		Keys: keys,
	}, nil
}

// RemoveKey implements the RemoveKey RPC method
func (s *AdminServer) RemoveKey(ctx context.Context, req *proto.RemoveKeyRequest) (*proto.RemoveKeyResponse, error) {
	s.logger.Info().Str("key_id", req.KeyId).Msg("removing key")
	tokenGenerator := utils.NewTokenGenerator(s.cloudConnector, s.localKeyStorage)
	localKeyStorage := utils.NewLocalKeyStorage()

	bucket, keyPath, err := s.cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get bucket and key path")
		return nil, fmt.Errorf("failed to get bucket and key path: %w", err)
	}

	// Download existing JWKS
	jwks, err := tokenGenerator.DownloadJWKS(ctx, bucket, keyPath)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to download JWKS")
		return nil, fmt.Errorf("failed to download JWKS: %w", err)
	}

	// Remove the specified key
	updatedJWKS := tokenGenerator.RemoveKeyFromJWKS(jwks, req.KeyId)

	// Upload the updated JWKS back to S3
	if err := tokenGenerator.UploadPublicKey(ctx, updatedJWKS, bucket, keyPath); err != nil {
		s.logger.Error().Err(err).Msg("failed to upload updated JWKS")
		return nil, fmt.Errorf("failed to upload updated JWKS: %w", err)
	}

	// Remove the key from local storage
	if err := localKeyStorage.DeleteLocalJWK(ctx, req.KeyId); err != nil {
		s.logger.Error().Err(err).Msg("failed to remove key from local storage")
		return nil, fmt.Errorf("failed to remove key from local storage: %w", err)
	}

	s.logger.Info().Str("key_id", req.KeyId).Msg("successfully removed key")
	return &proto.RemoveKeyResponse{}, nil
}

var _ utils.IServer = &AdminServer{}
