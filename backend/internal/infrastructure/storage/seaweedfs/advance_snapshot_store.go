// Package seaweedfs implementa domain.AdvanceSnapshotStore sobre la API S3
// compatible de SeaweedFS (ADR-058 §6).
package seaweedfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"solv-backend/internal/core/domain"
)

// Config agrupa los parámetros de conexión al gateway S3 de SeaweedFS.
// Todos los valores llegan desde variables de entorno; nada va fijo en código.
type Config struct {
	Endpoint  string // p. ej. http://seaweedfs:8333
	Region    string // SeaweedFS no valida la región; se usa una cualquiera
	Bucket    string
	AccessKey string
	SecretKey string
	// MaxBytes es el tope de tamaño del snapshot (tar.gz) en bytes.
	MaxBytes int64
	// Archive fija las exclusiones y el tope de archivos al empaquetar el código
	// del docente (ADR-058 §5). Sale de configuración, no del código.
	Archive ArchiveOptions
}

// AdvanceSnapshotStore guarda los tar.gz de los avances en un bucket S3.
type AdvanceSnapshotStore struct {
	client   *s3.Client
	bucket   string
	maxBytes int64
	archive  ArchiveOptions
}

var _ domain.AdvanceSnapshotStore = (*AdvanceSnapshotStore)(nil)
var _ domain.AdvanceSnapshotLister = (*AdvanceSnapshotStore)(nil)

// ErrChecksumMismatch indica que el objeto leído no coincide con el hash guardado.
var ErrChecksumMismatch = errors.New("el snapshot no coincide con su hash SHA-256")

// NewAdvanceSnapshotStore construye el cliente S3 apuntando al gateway de SeaweedFS.
// Usa path-style, que es el modo que SeaweedFS espera.
func NewAdvanceSnapshotStore(ctx context.Context, cfg Config) (*AdvanceSnapshotStore, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("seaweedfs: faltan endpoint, bucket o credenciales")
	}
	if cfg.MaxBytes <= 0 {
		return nil, errors.New("seaweedfs: el tope de tamaño debe ser mayor que cero")
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
	store := &AdvanceSnapshotStore{client: client, bucket: cfg.Bucket, maxBytes: cfg.MaxBytes, archive: cfg.Archive}
	if err := store.ensureBucket(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

// ensureBucket crea el bucket si todavía no existe. Un On-Premise recién
// instalado no lo tiene, y sin bucket la primera emisión fallaría.
func (s *AdvanceSnapshotStore) ensureBucket(ctx context.Context) error {
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err == nil {
		return nil
	}
	if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		return fmt.Errorf("seaweedfs: crear bucket %q: %w", s.bucket, err)
	}
	return nil
}

// PutFiles empaqueta los archivos del entorno del docente y los guarda como
// snapshot. El empaquetado aplica las exclusiones configuradas antes de subir
// nada, así que los archivos del editor y los internos de la plataforma no
// llegan al alumno.
func (s *AdvanceSnapshotStore) PutFiles(ctx context.Context, key string, files map[string]string) (string, int64, int, error) {
	var buf bytes.Buffer
	count, err := PackageFiles(files, s.archive, &buf)
	if err != nil {
		return "", 0, 0, err
	}
	if count == 0 {
		return "", 0, 0, domain.ErrAdvanceSourceMissing
	}
	if int64(buf.Len()) > s.maxBytes {
		return "", 0, 0, domain.ErrAdvanceSnapshotTooLarge
	}
	hash, size, err := s.Put(ctx, key, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", 0, 0, err
	}
	return hash, size, count, nil
}

// Put lee el cuerpo hasta el tope, calcula SHA-256 y lo sube. Si supera el tope
// devuelve domain.ErrAdvanceSnapshotTooLarge sin escribir nada.
func (s *AdvanceSnapshotStore) Put(ctx context.Context, key string, body io.Reader) (string, int64, error) {
	data, err := io.ReadAll(io.LimitReader(body, s.maxBytes+1))
	if err != nil {
		return "", 0, fmt.Errorf("seaweedfs: leer snapshot: %w", err)
	}
	if int64(len(data)) > s.maxBytes {
		return "", 0, domain.ErrAdvanceSnapshotTooLarge
	}
	sum := sha256.Sum256(data)
	hashHex := hex.EncodeToString(sum[:])
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String("application/gzip"),
	})
	if err != nil {
		return "", 0, fmt.Errorf("seaweedfs: subir snapshot: %w", err)
	}
	return hashHex, int64(len(data)), nil
}

// Get devuelve un lector del objeto. El llamador debe cerrarlo. Para verificar
// el hash, usar GetVerified.
func (s *AdvanceSnapshotStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("seaweedfs: leer snapshot: %w", err)
	}
	return out.Body, nil
}

// GetVerified lee el objeto completo, verifica que su SHA-256 coincida con
// expectedSHA256 y devuelve los bytes. Así nunca se copia al alumno un archivo
// alterado o truncado.
func (s *AdvanceSnapshotStore) GetVerified(ctx context.Context, key, expectedSHA256 string) ([]byte, error) {
	rc, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, s.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("seaweedfs: leer cuerpo del snapshot: %w", err)
	}
	if int64(len(data)) > s.maxBytes {
		return nil, domain.ErrAdvanceSnapshotTooLarge
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != expectedSHA256 {
		return nil, ErrChecksumMismatch
	}
	return data, nil
}

// Files devuelve los archivos del snapshot emitido, después de verificar su
// hash. Si el objeto no coincide con lo emitido no se entrega nada: es preferible
// un error a copiar al alumno un archivo alterado o truncado.
func (s *AdvanceSnapshotStore) Files(ctx context.Context, key, expectedSHA256 string, maxFiles int) ([]domain.WorkspaceVolumeFile, error) {
	data, err := s.GetVerified(ctx, key, expectedSHA256)
	if err != nil {
		return nil, err
	}
	return UnpackSnapshot(data, maxFiles)
}

// List recorre el bucket con el prefijo pedido. La retención usa la fecha de
// modificación para no borrar un objeto recién subido que todavía no tiene fila.
func (s *AdvanceSnapshotStore) List(ctx context.Context, prefix string) ([]domain.AdvanceSnapshotObject, error) {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	objects := make([]domain.AdvanceSnapshotObject, 0, 64)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("seaweedfs: listar snapshots: %w", err)
		}
		for _, object := range page.Contents {
			if object.Key == nil {
				continue
			}
			var modified time.Time
			if object.LastModified != nil {
				modified = *object.LastModified
			}
			objects = append(objects, domain.AdvanceSnapshotObject{Key: *object.Key, LastModified: modified})
		}
	}
	return objects, nil
}

// Delete elimina el objeto. S3 ya devuelve éxito si la clave no existe, así que es idempotente.
func (s *AdvanceSnapshotStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil
		}
		return fmt.Errorf("seaweedfs: eliminar snapshot: %w", err)
	}
	return nil
}
