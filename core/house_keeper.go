package core

import (
	"context"

	"github.com/altacoda/tailbone/utils"
	"github.com/rs/zerolog"
)

type HouseKeeper struct {
	logger          zerolog.Logger
	cloudConnector  utils.CloudConnector
	tokenGenerator  utils.IKeyManager
	localKeyStorage utils.ILocalKeyStorage
}

func NewHouseKeeper(ctx context.Context) (*HouseKeeper, error) {
	utils.InitLogger()
	logger := utils.GetLogger("housekeeper")
	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return nil, err
	}

	localKeyStorage := utils.NewLocalKeyStorage()
	tokenGenerator := utils.NewKeyManager(cloudConnector, localKeyStorage)

	return &HouseKeeper{
		logger:          logger,
		cloudConnector:  cloudConnector,
		tokenGenerator:  tokenGenerator,
		localKeyStorage: localKeyStorage,
	}, nil
}

func (h *HouseKeeper) Run(ctx context.Context) error {
	h.logger.Info().Msg("starting housekeeping")

	// download JWKs from the URL and compare with the local JWKs
	// if they are different, upload the new JWKs to the S3 bucket
	// if they are the same, do nothing

	// delete the local JWKs that are not present in the S3 bucket
	localJWKs, err := h.localKeyStorage.GetLocalJWKs(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get local JWKs")
		return err
	}

	h.logger.Info().Int("local_jwks", len(localJWKs.Keys)).Msg("got local JWKs")

	bucket, keyPath, err := h.cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get bucket and key path")
		return err
	}

	remoteJWKs, err := h.tokenGenerator.DownloadJWKS(ctx, bucket, keyPath)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to download JWKs from S3")
		return err
	}

	h.logger.Info().Int("remote_jwks", len(remoteJWKs.Keys)).Msg("got remote JWKs")

	// comprare by key
	for _, localKey := range localJWKs.Keys {
		found := false
		for _, remoteKey := range remoteJWKs.Keys {
			if localKey.KeyID() == remoteKey.KeyID() {
				found = true
				break
			}
		}

		if !found {
			h.logger.Info().Str("keyID", localKey.KeyID()).Msg("deleting local key")
			err := h.localKeyStorage.DeleteLocalJWK(ctx, localKey.KeyID())
			if err != nil {
				h.logger.Error().Err(err).Msg("failed to delete local key")
				return err
			}
		}
	}

	h.logger.Info().Msg("housekeeping completed")

	return nil
}
