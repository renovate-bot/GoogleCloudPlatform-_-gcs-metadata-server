package subscriber

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/gcs-metadata-server/internal/model"
	"github.com/GoogleCloudPlatform/gcs-metadata-server/internal/repo"
)

func TestHandleFinalize(t *testing.T) {
	testCases := []struct {
		name             string
		inMetadata       *model.Metadata
		existingMetadata *model.Metadata
		wantErr          bool
		wantInsertCalls  int
		wantUpdateCalls  int
		wantUpsertCalls  int
		wantArchiveCalls int
	}{
		{
			name: "Adds new metadata",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: nil,
			wantErr:          false,
			wantInsertCalls:  1,
			wantUpsertCalls:  1,
		},
		{
			name: "Updates metadata",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket-2",
				Name:         "mock-object-2",
				Size:         256,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket-2",
				Name:         "mock-object-2",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now(),
			},
			wantErr:         false,
			wantUpdateCalls: 1,
			wantUpsertCalls: 1,
		},
		{
			name: "Skip if incoming metadata is older",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now().Add(-time.Hour),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         512,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Updates storage class by passing to handleArchive()",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "ARCHIVE",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now(),
			},
			wantErr:          false,
			wantUpdateCalls:  1,
			wantArchiveCalls: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := repo.NewDatabase(":memory:", 1)
			db.Connect(context.Background())
			defer db.Close()

			if err := db.Setup(); err != nil {
				t.Fatal(err)
			}

			if err := db.CreateTables(); err != nil {
				t.Fatal(err)
			}

			dirRepo := repo.NewDirectoryRepository(db)
			metadataRepo := repo.NewMetadataRepository(db)

			// Insert existing metadata if available
			if tc.existingMetadata != nil {
				if err := metadataRepo.Insert(tc.existingMetadata); err != nil {
					t.Fatal(err)
				}
			}

			// Mock repositories
			mockMetadataRepo := &mockMetadataRepository{
				MetadataRepository: metadataRepo,
			}
			mockDirRepo := &mockDirectoryRepository{
				DirectoryRepository: dirRepo,
			}

			s := &SubscriberService{
				directoryRepo: mockDirRepo,
				metadataRepo:  mockMetadataRepo,
			}

			// Call handleFinalize
			if err := s.handleFinalize(tc.inMetadata); err != nil {
				if tc.wantErr {
					return
				}
				t.Fatal(err)
			}

			if tc.wantErr {
				t.Fatal("Expected error but did not fail")
			}

			// Check call counts
			if mockMetadataRepo.insertCalls != tc.wantInsertCalls {
				t.Errorf("metadata insert calls mismatch: got %d, want %d", mockMetadataRepo.insertCalls, tc.wantInsertCalls)
			}
			if mockMetadataRepo.updateCalls != tc.wantUpdateCalls {
				t.Errorf("metadata update calls mismatch: got %d, want %d", mockMetadataRepo.updateCalls, tc.wantUpdateCalls)
			}
			if mockDirRepo.upsertCalls != tc.wantUpsertCalls {
				t.Errorf("directory upsert calls mismatch: got %d, want %d", mockDirRepo.upsertCalls, tc.wantUpsertCalls)
			}
			if mockDirRepo.upsertArchiveCalls != tc.wantArchiveCalls {
				t.Errorf("directory upsertArchive calls mismatch: got %d, want %d", mockDirRepo.upsertArchiveCalls, tc.wantArchiveCalls)
			}
		})
	}
}

func TestHandleArchive(t *testing.T) {
	testCases := []struct {
		name                   string
		inMetadata             *model.Metadata
		existingMetadata       *model.Metadata
		wantErr                bool
		wantInsertCalls        int
		wantUpdateCalls        int
		wantUpsertArchiveCalls int
	}{
		{
			name: "Updates storage class",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "NEARLINE",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now(),
			},
			wantErr:                false,
			wantUpdateCalls:        1,
			wantUpsertArchiveCalls: 1,
		},
		{
			name: "Skip if storage class is already updated",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "NEARLINE",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "NEARLINE",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now(),
			},
			wantErr:                false,
			wantUpdateCalls:        0,
			wantUpsertArchiveCalls: 0,
		},
		{
			name: "Inserts metadata if does not exist",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "NEARLINE",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: nil,
			wantErr:          false,
			wantInsertCalls:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := repo.NewDatabase(":memory:", 1)
			db.Connect(context.Background())
			defer db.Close()

			if err := db.Setup(); err != nil {
				t.Fatal(err)
			}

			if err := db.CreateTables(); err != nil {
				t.Fatal(err)
			}

			dirRepo := repo.NewDirectoryRepository(db)
			metadataRepo := repo.NewMetadataRepository(db)

			// Insert existing metadata if available
			if tc.existingMetadata != nil {
				if err := metadataRepo.Insert(tc.existingMetadata); err != nil {
					t.Fatal(err)
				}
			}

			// Mock repositories
			mockMetadataRepo := &mockMetadataRepository{
				MetadataRepository: metadataRepo,
			}
			mockDirRepo := &mockDirectoryRepository{
				DirectoryRepository: dirRepo,
			}

			s := &SubscriberService{
				directoryRepo: mockDirRepo,
				metadataRepo:  mockMetadataRepo,
			}

			// Call handleArchive
			if err := s.handleArchive(tc.inMetadata); err != nil {
				if tc.wantErr {
					return
				}
				t.Fatal(err)
			}

			if tc.wantErr {
				t.Fatal("Expected error but did not fail")
			}

			// Check call counts
			if mockMetadataRepo.insertCalls != tc.wantInsertCalls {
				t.Errorf("metadata insert calls mismatch: got %d, want %d", mockMetadataRepo.insertCalls, tc.wantInsertCalls)
			}
			if mockMetadataRepo.updateCalls != tc.wantUpdateCalls {
				t.Errorf("metadata update calls mismatch: got %d, want %d", mockMetadataRepo.updateCalls, tc.wantUpdateCalls)
			}
			if mockDirRepo.upsertArchiveCalls != tc.wantUpsertArchiveCalls {
				t.Errorf("directory upsertArchive calls mismatch: got %d, want %d", mockDirRepo.upsertArchiveCalls, tc.wantUpsertArchiveCalls)
			}
		})
	}
}

func TestHandleDelete(t *testing.T) {
	testCases := []struct {
		name             string
		inMetadata       *model.Metadata
		existingMetadata *model.Metadata
		wantErr          bool
		wantDeleteCalls  int
		wantUpsertCalls  int
	}{
		{
			name: "Deletes existing metadata",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(-time.Hour),
				Created:      time.Now(),
			},
			wantErr:         false,
			wantDeleteCalls: 1,
			wantUpsertCalls: 1,
		},
		{
			name: "Fails if metadata does not exist",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: nil,
			wantErr:          true,
			wantDeleteCalls:  0,
			wantUpsertCalls:  0,
		},
		{
			name: "Skip delete if metadata is newer",
			inMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now(),
				Created:      time.Now(),
			},
			existingMetadata: &model.Metadata{
				Bucket:       "mock-bucket",
				Name:         "mock-object",
				Size:         1024,
				StorageClass: "STANDARD",
				Updated:      time.Now().Add(time.Hour),
				Created:      time.Now(),
			},
			wantErr:         false,
			wantDeleteCalls: 0,
			wantUpsertCalls: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := repo.NewDatabase(":memory:", 1)
			db.Connect(context.Background())
			defer db.Close()

			if err := db.Setup(); err != nil {
				t.Fatal(err)
			}

			if err := db.CreateTables(); err != nil {
				t.Fatal(err)
			}

			dirRepo := repo.NewDirectoryRepository(db)
			metadataRepo := repo.NewMetadataRepository(db)

			// Insert existing metadata if available
			if tc.existingMetadata != nil {
				if err := metadataRepo.Insert(tc.existingMetadata); err != nil {
					t.Fatal(err)
				}
			}

			// Mock repositories
			mockMetadataRepo := &mockMetadataRepository{
				MetadataRepository: metadataRepo,
			}
			mockDirRepo := &mockDirectoryRepository{
				DirectoryRepository: dirRepo,
			}

			s := &SubscriberService{
				directoryRepo: mockDirRepo,
				metadataRepo:  mockMetadataRepo,
			}

			// Call handleDelete
			if err := s.handleDelete(tc.inMetadata); err != nil {
				if tc.wantErr {
					return
				}
				t.Fatal(err)
			}

			if tc.wantErr {
				t.Fatal("Expected error but did not fail")
			}

			// Check call counts
			if mockMetadataRepo.deleteCalls != tc.wantDeleteCalls {
				t.Errorf("metadata delete calls mismatch: got %d, want %d", mockMetadataRepo.deleteCalls, tc.wantDeleteCalls)
			}
			if mockDirRepo.upsertCalls != tc.wantUpsertCalls {
				t.Errorf("directory upsert calls mismatch: got %d, want %d", mockDirRepo.upsertCalls, tc.wantUpsertCalls)
			}
		})
	}
}

type mockMetadataRepository struct {
	repo.MetadataRepository
	insertCalls int
	updateCalls int
	deleteCalls int
}

func (m *mockMetadataRepository) Get(bucket, name string) (*model.Metadata, error) {
	return m.MetadataRepository.Get(bucket, name)
}

func (m *mockMetadataRepository) Insert(obj *model.Metadata) error {
	m.insertCalls++
	return m.MetadataRepository.Insert(obj)
}

func (m *mockMetadataRepository) Update(bucket, name, storageClass string, size int64, updated time.Time) error {
	m.updateCalls++
	return m.MetadataRepository.Update(bucket, name, storageClass, size, updated)
}

func (m *mockMetadataRepository) Delete(bucket, name string) error {
	m.deleteCalls++
	return m.MetadataRepository.Delete(bucket, name)
}

type mockDirectoryRepository struct {
	repo.DirectoryRepository
	upsertCalls        int
	upsertArchiveCalls int
}

func (m *mockDirectoryRepository) Insert(dir model.Directory) error {
	return m.DirectoryRepository.Insert(dir)
}

func (m *mockDirectoryRepository) Delete(bucket, name string) error {
	return m.DirectoryRepository.Delete(bucket, name)
}

func (m *mockDirectoryRepository) UpsertParentDirs(storageClass repo.StorageClass, bucket string, objName string, newSize int64, newCount int64) error {
	m.upsertCalls++
	return m.DirectoryRepository.UpsertParentDirs(storageClass, bucket, objName, newSize, newCount)
}

func (m *mockDirectoryRepository) UpsertArchiveParentDirs(oldStorageClass repo.StorageClass, newStorageClass repo.StorageClass, bucket, objName string, size int64) error {
	m.upsertArchiveCalls++
	return m.DirectoryRepository.UpsertArchiveParentDirs(oldStorageClass, newStorageClass, bucket, objName, size)
}
