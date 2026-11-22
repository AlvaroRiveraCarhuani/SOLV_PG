package seaweedfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"solv-backend/internal/core/domain"
)

// WorkspaceProjectArtifactStore guarda los tar.gz de proyectos grandes en el
// mismo depósito que los avances, bajo un prefijo propio. Sin configuración el
// endpoint de carga responde 503 y el camino liviano no se entera.
type WorkspaceProjectArtifactStore struct {
	client   *s3.Client
	bucket   string
	maxBytes int64
}

func NewWorkspaceProjectArtifactStore(ctx context.Context, cfg Config) (*WorkspaceProjectArtifactStore, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("seaweedfs: faltan endpoint, bucket o credenciales")
	}
	if cfg.MaxBytes <= 0 {
		return nil, fmt.Errorf("seaweedfs: el tope del artefacto debe ser positivo")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("seaweedfs: cargar configuración AWS: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})
	return &WorkspaceProjectArtifactStore{client: client, bucket: cfg.Bucket, maxBytes: cfg.MaxBytes}, nil
}

// Put lee el cuerpo hasta el tope y devuelve el SHA-256 y el tamaño de lo que
// subió. El hash no se recibe del cliente: se calcula sobre los bytes reales.
func (s *WorkspaceProjectArtifactStore) Put(ctx context.Context, key string, body io.Reader) (string, int64, error) {
	if key == "" {
		return "", 0, fmt.Errorf("seaweedfs: clave de artefacto vacía")
	}
	data, err := io.ReadAll(io.LimitReader(body, s.maxBytes+1))
	if err != nil {
		return "", 0, fmt.Errorf("seaweedfs: leer artefacto: %w", err)
	}
	if int64(len(data)) > s.maxBytes {
		return "", 0, domain.ErrWorkspaceProjectArchiveTooLarge
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String("application/gzip"),
	}); err != nil {
		return "", 0, fmt.Errorf("seaweedfs: subir artefacto: %w", err)
	}
	return hash, int64(len(data)), nil
}

// Get entrega un lector del objeto. La verificación del hash la hace quien
// siembra, porque es quien conoce la huella emitida.
func (s *WorkspaceProjectArtifactStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("seaweedfs: leer artefacto: %w", err)
	}
	return out.Body, nil
}
