package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestGetSignedURLByAssetID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:asset_service_test?mode=memory&cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "t_", SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Asset{}); err != nil {
		t.Fatalf("migrate asset: %v", err)
	}
	sqls.SetDB(db)
	config.SetCurrent(&config.Config{Storage: config.StorageConfig{
		Default: enums.AssetProviderLocal,
		Local:   config.LocalStorageConfig{BaseURL: "/storage"},
	}})
	asset := &models.Asset{
		AssetID:    "avatar_asset_1",
		Provider:   enums.AssetProviderLocal,
		StorageKey: "avatars/avatar.png",
		MimeType:   "image/png",
		Status:     enums.AssetStatusSuccess,
	}
	if err := db.Create(asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}

	got, err := AssetService.GetSignedURLByAssetID(asset.AssetID)
	if err != nil {
		t.Fatalf("GetSignedURLByAssetID() error = %v", err)
	}
	if got != "/storage/avatars/avatar.png" {
		t.Fatalf("GetSignedURLByAssetID() = %q", got)
	}
}
